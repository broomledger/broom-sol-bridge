package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	netnode "github.com/canavan-a/broom/node/netnode"
)

const LOOKBACK_BLOCKS = 15

const LOOKBACK_CUTOFF = 40

const BRIDGE_BLOCK_DIR = "bridgeblocks"

const FULFILLED_TXN_DIR = "bridgetxns"

const BROOM_BRIDGE_WALLET_ADDRESS = "my wallet address here"

type BridgeRunner struct {
}

func NewBridgeRunner() (br *BridgeRunner) {
	if _, err := os.Stat(BRIDGE_BLOCK_DIR); os.IsNotExist(err) {
		_ = os.MkdirAll(BRIDGE_BLOCK_DIR, 0755) // create if missing
	}
	if _, err := os.Stat(FULFILLED_TXN_DIR); os.IsNotExist(err) {
		_ = os.MkdirAll(FULFILLED_TXN_DIR, 0755) // create if missing
	}

	return &BridgeRunner{}
}

// assume we are passing a validated block (already stored in broombase)
func (br *BridgeRunner) ReceiveBlock(b netnode.Block) error {
	// check if block has and bridge txns

	var bridgeTransactions []netnode.Transaction

	for _, txn := range b.Transactions {
		if txn.To == BROOM_BRIDGE_WALLET_ADDRESS {
			bridgeTransactions = append(bridgeTransactions, txn)
		}
	}

	if len(bridgeTransactions) == 0 {
		return errors.New("no bridge txns found")
	}

	_, found := br.GetBlock(b.Hash, b.Height)
	if found {
		return errors.New("bridge block already exists")
	}

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%d_%s.broom", BRIDGE_BLOCK_DIR, b.Height, b.Hash)
	return os.WriteFile(path, data, 0644)

}

func (br *BridgeRunner) SaveBridgeTxn(hash string, txn netnode.Transaction) error {
	data, err := json.Marshal(txn)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s.txn", FULFILLED_TXN_DIR, hash)
	return os.WriteFile(path, data, 0644)
}

func (bb *BridgeRunner) GetBlock(hash string, height int64) (block netnode.Block, found bool) {

	path := fmt.Sprintf("%s/%d_%s.broom", BRIDGE_BLOCK_DIR, height, hash)
	data, err := os.ReadFile(path)
	if err != nil {
		// file not found or read error
		return netnode.Block{}, false
	}

	var b netnode.Block
	err = json.Unmarshal(data, &b)
	if err != nil {
		return netnode.Block{}, false
	}

	return b, true
}

func (bb *BridgeRunner) GetBridgeTransaction(hash string) (block netnode.Transaction, found bool) {

	path := fmt.Sprintf("%s/%s.txn", FULFILLED_TXN_DIR, hash)
	data, err := os.ReadFile(path)
	if err != nil {
		// file not found or read error
		return netnode.Transaction{}, false
	}

	var txn netnode.Transaction
	err = json.Unmarshal(data, &txn)
	if err != nil {
		return netnode.Transaction{}, false
	}

	return txn, true
}

// hashChain is all the valid hashes we can process bridge txns on
// example: we are on block
func (bb *BridgeRunner) ProcessBridge(currentHeight int, hashChain []netnode.HashHeight) error {
	files, err := os.ReadDir(BRIDGE_BLOCK_DIR)
	if err != nil {
		return err
	}

	var pendingBridgeBlocks []netnode.HashHeight

	for _, file := range files {
		splitName := strings.Split(file.Name(), "_")
		heightValue, _ := strconv.Atoi(splitName[0])

		hashValue := strings.Split(splitName[1], ".broom")[0]

		pendingBridgeBlocks = append(pendingBridgeBlocks, netnode.HashHeight{
			Height: heightValue,
			Hash:   hashValue,
		})

	}

	for _, pending := range pendingBridgeBlocks {

		critialDepth := currentHeight - pending.Height
		if critialDepth < LOOKBACK_BLOCKS {
			// these blocks have not been around long enough, could lose to a fork
			continue
		}

		if critialDepth > LOOKBACK_CUTOFF {
			// these are likely on a fork
			// remove them
			if bb.RemoveBridgeBlock(pending.Hash, pending.Height) != nil {
				panic("should exist")
			}
			// skip
			continue

		}

		// TODO:
		// add valid txns to txn dir
		// start process to mint

		block, found := bb.GetBlock(pending.Hash, int64(pending.Height))
		if !found {
			continue
		}

		for _, txn := range block.Transactions {
			if txn.To == BROOM_BRIDGE_WALLET_ADDRESS {
				bb.ProcessBridgeTransaction(txn)
			}
		}

		if bb.RemoveBridgeBlock(pending.Hash, pending.Height) != nil {
			panic("should exist")
		}

	}

	return nil

}

func (br *BridgeRunner) ProcessBridgeTransaction(txn netnode.Transaction) {
	serialized := txn.Serialize()

	hash := sha256.Sum256(serialized)

	hashValue := hex.EncodeToString(hash[:])

	_, found := br.GetBridgeTransaction(hashValue)
	if found {
		// txn already processed
		return
	}

	br.SendSol(txn)

	err := br.SaveBridgeTxn(hashValue, txn)
	if err != nil {
		panic("txn sent but could not be saved")

	}
}

func (br *BridgeRunner) SendSol(txn netnode.Transaction) error {
	fmt.Println("sending sol to account")

	return nil

}

func (bb *BridgeRunner) RemoveBridgeBlock(hash string, height int) error {
	path := fmt.Sprintf("%s/%d_%s.broom", BRIDGE_BLOCK_DIR, height, hash)
	return os.Remove(path)
}

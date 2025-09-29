package main

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"time"

	netnode "github.com/canavan-a/broom/node/netnode"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
)

// need to wait 15 blocks before we can mint to the address

// REQUIREMENTS
// txn to this node's address
// note contains format:
// 							sol:adress
//
// Block comes in, we check for txns to "me"
// Stash the block in some storage location
//
// every time block comes in we check the current height
// scan through our bridge blocks if its greater than our block buffer
//       if it is, trace back through to see if the chain connects back from highest to bridge block

//  extra condition: if bridge block never connects to tip, we have a maximum threshold, if its too far back without connecting throw out the bridge block that was originally stashed

const TOKEN_ID = "EJtfMsN3qfh8QJJfpEmWVxW43MPH522xtXBfvJNA9Bdk"

type BroomBridge struct {
	*netnode.Executor
	private solana.PrivateKey
	public  solana.PublicKey
	client  rpc.Client
	// public solana
}

func NewBroomBridge(myAddress string, miningNote string, dir string, ledgerDir string) *BroomBridge {

	ex := netnode.NewExecutor(myAddress, "", netnode.BROOMBASE_DEFAULT_DIR, netnode.LEDGER_DEFAULT_DIR)

	bb := &BroomBridge{Executor: ex}

	bb.LoadKeys()

	bb.DialClient()

	return bb
}

func (bb *BroomBridge) LoadKeys() {
	data, _ := os.ReadFile("id.json")

	var keyBytes []byte
	json.Unmarshal(data, &keyBytes)

	privKey := solana.PrivateKey(keyBytes)
	pubKey := privKey.PublicKey()

	bb.private = privKey
	bb.public = pubKey

	fmt.Println("Public key: ", pubKey)

}

func (bb *BroomBridge) DialClient() {
	client := rpc.New(rpc.MainNetBeta_RPC)
	bb.client = *client
}

func (bb *BroomBridge) MakeAccount(address string) error {
	// peerWallet := solana.NewWallet()

	parsedAddress := solana.MustPublicKeyFromBase58(address)

	mint := solana.MustPublicKeyFromBase58(TOKEN_ID)
	ata, _, _ := solana.FindAssociatedTokenAddress(parsedAddress, mint)

	ix := token.NewInitializeAccount2Instruction(
		parsedAddress,
		ata,
		mint,
		solana.SysVarRentPubkey,
	).Build()

	recent, err := bb.client.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return err
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		recent.Value.Blockhash,
	)
	if err != nil {
		return err
	}

	x := func(solana.PublicKey) *solana.PrivateKey {
		return &bb.private
	}
	_, err = tx.Sign(x)
	if err != nil {
		return err
	}

	sig, err := bb.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}

	bb.client.GetConfirmedTransaction(context.Background(), sig)

	// confirm txn here

	return nil
}

func (bb *BroomBridge) bb_Start(self string, seeds ...string) {
	bb.Node = netnode.ActivateNode(bb.MsgChan, bb.BlockChan, bb.EgressBlockChan, bb.TxnChan, bb.EgressTxnChan, self, seeds...)

	if len(seeds) != 0 {
		for {
			if len(bb.Node.GetAddressSample()) != 0 {
				break
			}
			time.Sleep(1 * time.Second)
			fmt.Println("waiting for at least one peer")
		}
	}

	fmt.Println("Starting rest server")
	bb.SetupHttpServer()

	ctx, cancel := context.WithCancel(context.Background())

	fmt.Println("Syncing to network")
	bb.NetworkSync(ctx)

	// set backup schedule
	bb.Node.Schedule(func() {
		fmt.Println("runnning backup")
		bb.RunBackup()
		fmt.Println("backup done")
	}, time.Second*netnode.BACKUP_FREQUENCY)

	fmt.Println("escaped network sync height", bb.Database.Ledger.BlockHeight)

	bb.MiningBlock.Height = bb.Database.Ledger.BlockHeight + 1

	fmt.Println("Node running")
	bb.bb_Loop()

	cancel()

}

func (ex *BroomBridge) bb_Loop() {

	ctx := context.Background()
	doneChan := make(chan struct{})

	fmt.Println("mining started")

	// every 10 min without a block check up and sync
	timer := time.NewTimer(netnode.SYNC_CHECK_DURATION)
	defer timer.Stop()

	for {

		select {
		case <-timer.C:
			close(doneChan)
			// run a network sync every 5 ish minutes
			fmt.Println("running network sync")
			syncRequired := ex.NetworkSyncWithTracker(ctx)
			if syncRequired {

				ex.MiningBlock.Height = ex.Database.Ledger.BlockHeight + 1
				//clear the mempool, we had to sync and don't know what txns are added or not added
				ex.Mempool = make(map[string]netnode.Transaction)
			} else {
				// I may be tracking a higher fork but not using it
				fmt.Println("checking for fork disparity")
				height, hash, err := ex.Database.GetHighestBlock()
				if err != nil {
					panic("must be able to get highets block")
				}

				if height > ex.Database.Ledger.BlockHeight {
					fmt.Println("fixing height disparity")
					ledger, found := ex.Database.GetLedgerAt(hash, height)
					if found {
						ex.Database.Ledger.Mut.Lock()
						ledger.Mut = ex.Database.Ledger.Mut
						fmt.Println("setting ledger")
						ex.Database.Ledger = ledger
						ex.Database.Ledger.Mut.Unlock()

						// clear our mempool
						ex.Mempool = make(map[string]netnode.Transaction)
						// set out mining block height
						ex.MiningBlock.Height = ex.Database.Ledger.BlockHeight + 1

					}

				}

			}

			timer.Reset(netnode.SYNC_CHECK_DURATION)
			doneChan = make(chan struct{})

			fmt.Println("sync done")

			// send dead block to start mining again
		case block := <-ex.BlockChan:

			currentSolution := ex.Database.Ledger.BlockHeight+1 == block.Height && ex.Database.Ledger.BlockHash == block.PreviousBlockHash

			// stop mining, handle block validation and storage, start mining again
			fmt.Println("incoming block (network or self)")
			close(doneChan)
			fmt.Println(block.Height)
			fmt.Println("hash: ", block.Hash)
			fmt.Println("prev: ", block.PreviousBlockHash)
			err := ex.Database.ReceiveBlock(block)
			if err != nil {
				fmt.Println("Block invalid: ", err)

			} else {

				// share the block with the egress
				ex.EgressBlockChan <- block

				// no error
				if currentSolution {
					fmt.Println("block is current ledger solution")
					// smart clear the mempool because we might have valid txns not included in the block
					for txnSig := range block.Transactions {
						delete(ex.Mempool, txnSig)
					}
					// if this skips and syncs forward we may have txns we try to put in that are already in the chain

					ex.ResetMiningBlock()

					// copy remaining txns in the mempool into the new block
					maps.Copy(ex.MiningBlock.Transactions, ex.Mempool)
				}

			}

			doneChan = make(chan struct{})
		case txn := <-ex.TxnChan:
			// stop mining, add txn to block, handle block validation, start mining again
			fmt.Println("incoming network txn")
			close(doneChan)

			// TODO: validate the txn, need to validate the block on a specific ledger level

			sigValid, err := txn.ValidateSig()
			if err != nil {
				fmt.Println("Error validating txn sig, pass: ", err)
			}

			sizeValid := txn.ValidateSize()

			if sigValid && sizeValid {
				//TODO: validate nonce and balance against current ledger

				fmt.Println("txn from", txn.From)
				accountBalance, balanceFound := ex.Database.Ledger.GetAddressBalance(txn.From)
				fmt.Println("account balance", accountBalance)

				accountNonce, _ := ex.Database.Ledger.GetAddressNonce(txn.From)
				fmt.Println("txn nonce:", txn.Nonce)

				if balanceFound {
					fmt.Println("Found balance and nonce")

					addressTxns := ex.GetAddressTransactions(txn.From)

					addressTxns = append(addressTxns, txn)
					// toValidate := append(ex.miningBlock.Transactions... )
					err := netnode.ValidateTransactionGroup(accountBalance, accountNonce, addressTxns)
					if err != nil {
						// pass: do nothing, txn is not validated
						fmt.Println("could not validate txn: ", err)
					} else {
						fmt.Println("adding txn to block, mempool, and sending out")
						// egress the txn
						ex.EgressTxnChan <- txn

						// we found a good txn, add it to the mempool
						ex.Mempool[txn.Sig] = txn
						ex.MiningBlock.Add(txn)
					}
				} else {
					fmt.Println("could not find balance and nonce")
				}

			}

			doneChan = make(chan struct{})

		}

	}

}

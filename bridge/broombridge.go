package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"time"

	netnode "github.com/canavan-a/broom/node/netnode"
	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
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

const BRIDGE_SOL_ADDRESS = "5EzYAKuTXtPadcmozKJW3GfBZ244K2fk79SfLrnzkwcu"

const TOKEN_ID = "EJtfMsN3qfh8QJJfpEmWVxW43MPH522xtXBfvJNA9Bdk"

const MIN_BRIDGE_AMT = 10_000 // base units

const INITIAL_FEE_PERCENTAGE = 0.10

const STANDARD_FEE_PERCENTAGE = 0.001

type BroomBridge struct {
	*netnode.Executor
	private solana.PrivateKey
	public  solana.PublicKey
	client  rpc.Client
	// public solana
	runner     *BridgeRunner
	rpcAddress string
}

func NewBroomBridge(myAddress string, miningNote string, dir string, ledgerDir string) *BroomBridge {

	ex := netnode.NewExecutor(myAddress, "", netnode.BROOMBASE_DEFAULT_DIR, netnode.LEDGER_DEFAULT_DIR)

	bb := &BroomBridge{Executor: ex}

	bb.LoadKeys()

	bb.LoadRpcCredentials()

	bb.DialClient()

	bb.runner = NewBridgeRunner(bb.TransactionHandler)

	return bb
}

// SOL -> Broom
func Uint64Ptr(v uint64) *uint64 { return &v }

func (bb *BroomBridge) RunSolScan() error {
	splAddress := solana.MustPublicKeyFromBase58(BRIDGE_SOL_ADDRESS)

	sigs, err := bb.client.GetSignaturesForAddress(
		context.Background(),
		splAddress,
	)
	if err != nil {
		fmt.Println("could not get sigs: ", err)
		return err
	}

	fmt.Println("starting txn loop")
	fmt.Println("len of sigs: ", len(sigs))

	for _, s := range sigs {

		fmt.Println(s.Signature)

		fmt.Println(s.Memo)

		txn, err := bb.client.GetTransaction(context.Background(), s.Signature, &rpc.GetTransactionOpts{
			Encoding:                       solana.EncodingBase64,
			Commitment:                     rpc.CommitmentFinalized,
			MaxSupportedTransactionVersion: Uint64Ptr(0),
		})
		if err != nil {
			return err
		}

		time.Sleep(time.Second)

		fmt.Println("found txn: ", txn)
		fmt.Println(txn)
		fmt.Println(txn.Meta)
		fmt.Println("pre")
		fmt.Println(txn.Meta.PreTokenBalances[0].Mint)
		fmt.Println("post")
		fmt.Println(txn.Meta.PostTokenBalances[0].Mint)
		err = bb.runner.ProcessSOLTxn(txn)
		if err != nil {
			return err
		}
		break

	}

	return nil

}

func (bb *BroomBridge) TransactionHandler(txn netnode.Transaction) error {

	solAddress := txn.Note

	pubKey, err := solana.PublicKeyFromBase58(solAddress)
	if err != nil {
		return err
	}

	mint, err := solana.PublicKeyFromBase58(TOKEN_ID)
	if err != nil {
		return err
	}

	accountExists := bb.FindAssociated(mint, pubKey)
	percent := INITIAL_FEE_PERCENTAGE
	if accountExists {
		fmt.Println("account exists use standard percent")

		percent = STANDARD_FEE_PERCENTAGE
		fmt.Println(percent)
	}

	if txn.Amount < MIN_BRIDGE_AMT {
		return errors.New("txn amount is too low")
	}

	fee := int64(float64(txn.Amount) * percent)

	fmt.Println(fee)

	amountWithoutFee := txn.Amount - fee

	exists, err := bb.MakeAndWaitForAta(solAddress)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("could not make ata")
	}

	ata, _, _ := solana.FindAssociatedTokenAddress(
		pubKey,
		mint,
	)

	err = bb.MintToken(ata, uint64(amountWithoutFee))
	if err != nil {
		return err
	}

	return nil
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

func (bb *BroomBridge) LoadRpcCredentials() {
	data, _ := os.ReadFile("rpc.address")
	fmt.Println(string(data))
	bb.rpcAddress = string(data)
}

func (bb *BroomBridge) DialClient() {
	if bb.rpcAddress == "" {
		panic("no rpc.address file loaded")
	}

	client := rpc.New(bb.rpcAddress)
	bb.client = *client
}

func (bb *BroomBridge) FindAssociated(mint, wallet solana.PublicKey) (found bool) {
	associatedTokenAddress, _, _ := solana.FindAssociatedTokenAddress(
		wallet,
		mint,
	)

	ata, err := bb.client.GetAccountInfo(context.Background(), associatedTokenAddress)
	if err != nil {
		fmt.Println("ATA does not exist or error:", err)
		return false
	} else {
		fmt.Println("pos ata: ", solana.PublicKey(ata.Bytes()).String())
		return true
	}

}

func (bb *BroomBridge) MintToken(ata solana.PublicKey, amount uint64) error {

	mint := solana.MustPublicKeyFromBase58(TOKEN_ID)

	out, err := bb.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return err
	}

	ix := token.NewMintToInstruction(
		amount,
		mint,
		ata,
		bb.private.PublicKey(), // mint authority
		nil,                    // multisig signers
	).Build()

	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		out.Value.Blockhash,
		solana.TransactionPayer(bb.private.PublicKey()),
	)
	if err != nil {
		return err
	}

	x := func(solana.PublicKey) *solana.PrivateKey {
		return &bb.private
	}

	if _, err := tx.Sign(x); err != nil {
		return err
	}

	_, err = bb.client.SendTransaction(context.Background(), tx)
	return err
}

func (bb *BroomBridge) MakeAndWaitForAta(address string) (exists bool, err error) {

	parsedAddress := solana.MustPublicKeyFromBase58(address)

	mint := solana.MustPublicKeyFromBase58(TOKEN_ID)

	payer := solana.MustPublicKeyFromBase58(BRIDGE_SOL_ADDRESS)

	// check if associated account exists

	found := bb.FindAssociated(mint, parsedAddress)
	if found {
		return true, nil
	}

	ix := associatedtokenaccount.NewCreateInstruction(
		payer,         // pays for rent
		parsedAddress, // token account owner
		mint,          // token mint
	).Build()

	fmt.Println("build txn")

	out, err := bb.client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return false, err
	}

	// create txn
	tx, err := solana.NewTransaction(
		[]solana.Instruction{ix},
		out.Value.Blockhash,
		solana.TransactionPayer(bb.private.PublicKey()),
	)
	if err != nil {
		return false, err
	}

	x := func(solana.PublicKey) *solana.PrivateKey {
		return &bb.private
	}

	_, err = tx.Sign(x)
	if err != nil {
		return false, err
	}

	_, err = bb.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return false, err
	}

	// spin until token account is confirmed
	for {
		found := bb.FindAssociated(mint, parsedAddress)
		if found {
			return true, nil
		}
		fmt.Println("trying again")
		time.Sleep(1 * time.Second)
	}

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

func (bb *BroomBridge) GenerateHashChain() (hashChain []netnode.HashHeight) {
	height, hash, err := bb.Database.GetHighestBlock()
	if err != nil {
		panic(err)
	}

	cur := netnode.HashHeight{
		Height: int(height),
		Hash:   hash,
	}

	hashChain = append(hashChain, cur)

	for {
		if len(hashChain) > LOOKBACK_CUTOFF+5 {
			return
		}

		block, found := bb.Database.GetBlock(cur.Hash, int64(cur.Height))
		if !found {
			panic("block not found")
		}

		cur = netnode.HashHeight{
			Height: int(block.Height),
			Hash:   block.Hash,
		}
		hashChain = append(hashChain, cur)
	}

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

					// run bridgerunner rec block
					err := ex.runner.ReceiveBlock(block)
					if err != nil {
						fmt.Println("error receiving bridge block")
						fmt.Println(err)
					}

					// get the hashchain to help validate blocks
					hc := ex.GenerateHashChain()

					//process txns that could send sol to an address
					err = ex.runner.ProcessBridge(int(ex.Database.Ledger.BlockHeight), hc)
					if err != nil {
						fmt.Println("error processing bridge")
						fmt.Println(err)
					}

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

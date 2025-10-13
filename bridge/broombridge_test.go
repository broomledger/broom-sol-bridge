package main

import (
	"fmt"
	"testing"

	"github.com/canavan-a/broom/node/netnode"
	"github.com/gagliardetto/solana-go"
)

func TestLoadKeys(t *testing.T) {
	fmt.Println("hello world")
	bb := BroomBridge{}
	bb.LoadKeys()
	fmt.Println(bb.public.String())
	bb.DialClient()

	// wallet := solana.NewWallet()
	// public := wallet.PublicKey().String()

	public := "5XGQBMBBPuzaA3nBbLbFhyytCT8dozt5U9WTubuFxpac"

	mint := solana.MustPublicKeyFromBase58(TOKEN_ID)

	pubKey := solana.MustPublicKeyFromBase58(public)

	bb.MakeAndWaitForAta(public)

	found := bb.FindAssociated(mint, pubKey)
	if found {
		fmt.Println("found")

		ata, _, _ := solana.FindAssociatedTokenAddress(
			pubKey,
			mint,
		)
		fmt.Println("ata: ", ata.String())

		bb.MintToken(ata, 19)

	}

	fmt.Println("public key for :", public)

	// exists, err := bb.MakeAccount(public)
	// fmt.Println(exists)
	// fmt.Println(err)
	panic("hello world")
}

func TestTransactionHandler(t *testing.T) {
	fmt.Println("hello world")
	bb := BroomBridge{}
	bb.LoadKeys()
	fmt.Println(bb.public.String())

	bb.LoadRpcCredentials()
	bb.DialClient()

	// wallet := solana.NewWallet()
	// public := wallet.PublicKey().String()
	// fmt.Println("public: ", public)
	//"5XGQBMBBPuzaA3nBbLbFhyytCT8dozt5U9WTubuFxpac"
	err := bb.TransactionHandler(netnode.Transaction{Note: "EL4FsBBKWDt1uJhQnANcG7NJ8r6prsX5mvbPe9N65QEv", Amount: 10_123})
	if err != nil {
		fmt.Println("err")
		fmt.Println(err)
	}

	panic("test failed")
}

func TestSolScan(t *testing.T) {
	fmt.Println("hello world")
	bb := BroomBridge{}
	bb.LoadKeys()
	bb.LoadRpcCredentials()
	fmt.Println(bb.public.String())
	bb.DialClient()

	// bridgeAddr := solana.MustPublicKeyFromBase58(BRIDGE_SOL_ADDRESS)

	// mint := solana.MustPublicKeyFromBase58(TOKEN_ID)

	// ata, _, _ := solana.FindAssociatedTokenAddress(
	// 	bridgeAddr,
	// 	mint,
	// )

	fmt.Println("starting 111")

	err := bb.RunSolScan()
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
	}

	panic("test failed")
}

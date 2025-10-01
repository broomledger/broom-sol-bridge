package main

import (
	"fmt"
	"testing"

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

	public := "CCv8rznjLSyoBDTsGE9XGZUV7tFUVF4vTMwp6JWEq5MU"

	mint := solana.MustPublicKeyFromBase58(TOKEN_ID)

	pubKey := solana.MustPublicKeyFromBase58(public)
	found := bb.FindAssociated(mint, pubKey)
	if found {
		associatedTokenAddress, _, _ := solana.FindAssociatedTokenAddress(
			pubKey,
			mint,
		)

		fmt.Println("ata: ", associatedTokenAddress)
	}

	fmt.Println("public key for :", public)

	// exists, err := bb.MakeAccount(public)
	// fmt.Println(exists)
	// fmt.Println(err)
	panic("hello world")
}

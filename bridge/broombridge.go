package main

import (
	netnode "github.com/canavan-a/broom/node/netnode"
)

type BroomBridge struct {
	*netnode.Executor
}

func NewBroomBridge(myAddress string, miningNote string, dir string, ledgerDir string) *BroomBridge {

	ex := netnode.NewExecutor(myAddress, "", netnode.BROOMBASE_DEFAULT_DIR, netnode.LEDGER_DEFAULT_DIR)

	return &BroomBridge{ex}
}

func (bb *BroomBridge) bb_Start(workers int, self string, seeds ...string) {

}

func (bb *BroomBridge) bb_Loop() {

}

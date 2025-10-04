package main

import (
	"flag"

	"github.com/canavan-a/broom/node/netnode"
)

func main() {
	// run sol bridge
	flag.Parse()
	args := flag.Args()

	bb := NewBroomBridge("", "", netnode.BROOMBASE_DEFAULT_DIR, netnode.LEDGER_DEFAULT_DIR)

	if args[0] == "backup" {
		bb.Executor.DownloadBackupFileFromPeer("node.broomledger.com")
	} else {
		bb.bb_Start(args[0], args[1:]...)

	}

}

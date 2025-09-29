package main

import (
	"fmt"
	"testing"
)

func TestLoadKeys(t *testing.T) {
	fmt.Println("hello world")
	bb := BroomBridge{}
	bb.LoadKeys()
	fmt.Println(bb.public.String())
	// bb.MakeAccount()
}

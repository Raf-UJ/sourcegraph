package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/sourcegraph/sourcegraph/dev/executionlogs/proto"

	delim "google.golang.org/protobuf/encoding/protodelim"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		var msg proto.SpawnExec
		err := delim.UnmarshalFrom(r, &msg)
		if err == io.EOF {
			return
		} else if err != nil {
			panic(err)
		}
		fmt.Printf("%s %s %+v\n", msg.Mnemonic, msg.TargetLabel, msg)
	}
}

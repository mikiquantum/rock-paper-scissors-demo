package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/mikiquantum/rock-paper-scissors-demo/game"
	"github.com/mikiquantum/rock-paper-scissors-demo/p2p"
)

func main() {
	bootstrapOnly := false

	// Do not do this, use something like cobra to handle cmd validation/parsing
	args := os.Args[1:]
	if len(args) < 2 {
		panic(errors.New(fmt.Sprintf("Not enough Arguments [%d] \nUsage: %s port prefix", len(args), os.Args[0])))
	}

	if len(args) == 3 && args[2] == "bootstrap" {
		bootstrapOnly = true
	}

	// Set up a libp2p host.
	port, err := strconv.Atoi(args[0])
	if err != nil {
		panic(err)
	}
	bhost, err := p2p.MakePlayerHost(port, args[1])
	if err != nil {
		panic(err)
	}

	p2p.RunDHT(context.Background(), bhost, bootstrapOnly)

	if !bootstrapOnly {
		player := game.NewPlayer(bhost)
		player.StartPlaying()
	} else {
		select {}
	}

}

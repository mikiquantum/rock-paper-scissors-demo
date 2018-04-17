package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/mikiquantum/rock-paper-scissors-demo/p2p"
	"github.com/mikiquantum/rock-paper-scissors-demo/game"
)

func main() {
	bootstrapOnly := false

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
	// p2p.MakePlayerHost creates the peer and starts listening to incoming connections on the specified port
	bhost, err := p2p.MakePlayerHost(port, args[1])
	if err != nil {
		panic(err)
	}

	// RunDHT sets up the distributed hash table to track other available peers to allow lookup by peer id.
	p2p.RunDHT(context.Background(), bhost, bootstrapOnly)

	// The bootstrap node provides resolution for peer discovery requests without participating in the game.
	if bootstrapOnly {
		select {}
	} else {
		player := game.NewPlayer(bhost)
		player.StartPlaying()
	}

}

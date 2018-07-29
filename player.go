package main

import (
	"context"
	"os"

	"github.com/mikiquantum/rock-paper-scissors-demo/p2p"
	"github.com/mikiquantum/rock-paper-scissors-demo/game"
	"flag"
	"fmt"
	"log"
)

func usage() {
	fmt.Printf("Usage: %s -p port -prefix prefix [-bootstrap]\n", os.Args[0])
}

func main() {

	port := flag.Int("p", 0, "Listen Port")
	prefix := flag.String("prefix", "", "Prefix of Node - key match in resources")
	bootstrapOnly := flag.Bool("bootstrap", false, "As Bootstrap Node")
	help := flag.Bool("help", false, "Show Help")

	flag.Parse()

	if *help {
		usage()
		os.Exit(0)
	}

	if *port == 0 || *prefix == "" {
		usage()
		os.Exit(1)
	}

	if *bootstrapOnly {
		log.Printf("Forcing Bootstrap to run on port 30000\n")
		*port = 30000
	}

	// p2p.MakePlayerHost creates the peer and starts listening to incoming connections on the specified port
	bhost, err := p2p.MakePlayerHost(*port, *prefix)
	if err != nil {
		panic(err)
	}

	// RunDHT sets up the distributed hash table to track other available peers to allow lookup by peer id.
	p2p.RunDHT(context.Background(), bhost, *bootstrapOnly)

	// The bootstrap node provides resolution for peer discovery requests without participating in the game.
	if *bootstrapOnly {
		select {}
	} else {
		player := game.NewPlayer(bhost)
		player.StartPlaying()
	}

}

package p2p

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"golang.org/x/crypto/ed25519"
)

const (
	GAME_STREAM_PID          = "/game/rps"
	KEY_DIR                  = "./resources"
	HARDCODED_BOOTSTRAP_NODE = "/ip4/127.0.0.1/tcp/30000/ipfs/QmNYcCDjtCRdYaYPpNkTSiQTpLLxRapMe1P3EGsmA2wK7D"
)

func loadEncryptionKeyPair(prefix string) (pub crypto.PubKey, priv crypto.PrivKey) {
	key, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.pub", KEY_DIR, prefix))
	if err != nil {
		panic(err)
	}
	publicKey := ed25519.PublicKey(key)

	key, err = ioutil.ReadFile(fmt.Sprintf("%s/%s.key", KEY_DIR, prefix))
	if err != nil {
		panic(err)
	}
	privateKey := ed25519.PrivateKey(key)

	keyBytes := append(privateKey, publicKey...)

	priv, err = crypto.UnmarshalEd25519PrivateKey(keyBytes)
	if err != nil {
		panic(err)
	}
	pub = priv.GetPublic()

	return
}

func MakePlayerHost(listenPort int, prefix string) (host.Host, error) {
	// Get the signing key for the host.
	publicKey, privateKey := loadEncryptionKeyPair(prefix)

	// libp2p uses multihash formats to build up the PeerID
	pid, err := peer.IDFromPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	// Create a peerstore
	ps := peerstore.NewPeerstore()

	// Add the keys to the peerstore
	// for this peer ID.
	err = ps.AddPubKey(pid, publicKey)
	if err != nil {
		log.Printf("Could not enable encryption: %v\n", err)
		return nil, err
	}
	err = ps.AddPrivKey(pid, privateKey)
	if err != nil {
		log.Printf("Could not enable encryption: %v\n", err)
		return nil, err
	}

	// The node is now set up with encryption

	// Create a multiaddress for ipv4 on top of tcp listening on all interfaces
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort))
	if err != nil {
		return nil, err
	}

	// Create swarm (implements libP2P Network) identified by the PeerID
	// Enables listening on a set of addresses (multi-address)
	// Defines the transport settings
	swrm, err := swarm.NewSwarm(
		context.Background(),
		[]multiaddr.Multiaddr{addr},
		pid,
		ps,
		nil,
	)
	if err != nil {
		return nil, err
	}

	netw := (*swarm.Network)(swrm)
	basicHost := basichost.New(netw)

	return basicHost, nil
}

// Most of this code is taken from https://gist.github.com/whyrusleeping/169a28cffe1aedd4419d80aa62d361aa

// RunDHT maintains the DHT. It can be run in two modes:
// As a bootstrap node, it will provide response to discovery requests. As client only mode, the DHT is more lightweight
func RunDHT(ctx context.Context, h host.Host, asBootstrap bool) {
	var dhtClient *dht.IpfsDHT

	if asBootstrap {
		dhtClient = dht.NewDHT(ctx, h, datastore.NewMapDatastore()) // Run it as a Bootstrap Node
	} else {
		dhtClient = dht.NewDHTClient(ctx, h, datastore.NewMapDatastore()) // Just run it as a client, will not respond to discovery requests
	}

	bootstrapPeers := []string{HARDCODED_BOOTSTRAP_NODE}

	log.Printf("Bootstrapping %s\n", bootstrapPeers)
	for _, addr := range bootstrapPeers {
		iaddr, _ := ipfsaddr.ParseString(addr)

		pinfo, _ := peerstore.InfoFromP2pAddr(iaddr.Multiaddr())

		if err := h.Connect(ctx, *pinfo); err != nil {
			log.Println("Bootstrapping to peer failed: ", err)
		}
	}

	// Using the sha256 of our "topic" as our rendezvous value
	// Uses CID (https://github.com/ipld/cid). Naming standard for content used by IPFS
	c, _ := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256).Sum([]byte("rock-paper-scissors-dht"))

	// First, announce ourselves as participating in this topic
	log.Println("Announcing ourselves...")
	tctx, _ := context.WithTimeout(ctx, time.Second*10)
	if err := dhtClient.Provide(tctx, c, true); err != nil {
		// Important to keep this as Non-Fatal error, otherwise it will fail for a node that behaves as well as bootstrap one
		log.Printf("Error: %s\n", err.Error())
	}

	// Now, look for others who have announced
	// This queries the existing nodes, at the moment only the bootstrap one will respond to the request,
	// returning list of known peers
	log.Println("Searching for other peers ...")
	peers, err := dhtClient.FindProviders(tctx, c)
	if err != nil {
		panic(err)
	}
	log.Printf("Found %d peers!\n", len(peers))

	// Now connect to them, so they can added to the peerstore enabling connecting by PeerID
	// so we do not need to know the network/transport/location information of the remote peer
	for _, pe := range peers {
		if pe.ID == h.ID() {
			// No sense connecting to ourselves
			continue
		}

		tctx, _ := context.WithTimeout(ctx, time.Second*5)
		if err := h.Connect(tctx, pe); err != nil {
			log.Println("Failed to connect to peer: ", err)
		}
	}

	log.Println("Bootstrapping and discovery complete!")
}

func SendInt(host host.Host, move int, destination peer.ID) (err error) {
	var s net.Stream

	for {
		s, err = host.NewStream(context.Background(), destination, GAME_STREAM_PID)
		if err != nil {
			log.Println(err)
			time.Sleep(2 * time.Second)
		} else {
			break
		}
	}
	wrappedMessage := fmt.Sprintf("%d\n", move)
	_, err = s.Write([]byte(wrappedMessage))
	if err != nil {
		panic(err)
	}
	return
}
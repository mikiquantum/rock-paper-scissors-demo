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

func loadEncryptionKeyPair(prefix string) (publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey) {
	key, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.pub", KEY_DIR, prefix))
	if err != nil {
		panic(err)
	}
	publicKey = ed25519.PublicKey(key)

	key, err = ioutil.ReadFile(fmt.Sprintf("%s/%s.key", KEY_DIR, prefix))
	if err != nil {
		panic(err)
	}
	privateKey = ed25519.PrivateKey(key)
	return
}

func MakePlayerHost(listenPort int, prefix string) (host.Host, error) {
	// Get the signing key for the host.
	publicKey, privateKey := loadEncryptionKeyPair(prefix)
	var key []byte
	key = append(key, privateKey...)
	key = append(key, publicKey...)

	priv, err := crypto.UnmarshalEd25519PrivateKey(key)
	if err != nil {
		return nil, err
	}
	pub := priv.GetPublic()

	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, err
	}

	// Create a peerstore
	ps := peerstore.NewPeerstore()

	// Add the keys to the peerstore
	// for this peer ID.
	err = ps.AddPubKey(pid, pub)
	if err != nil {
		log.Printf("Could not enable encryption: %v\n", err)
		return nil, err
	}
	err = ps.AddPrivKey(pid, priv)
	if err != nil {
		log.Printf("Could not enable encryption: %v\n", err)
		return nil, err
	}

	// Create a multiaddress
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort))
	if err != nil {
		return nil, err
	}

	// Create swarm (implements libP2P Network)
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

	// Build host multiaddress
	hostAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("I am %s\n", fullAddr)

	return basicHost, nil
}

// Most of this code is taken from https://gist.github.com/whyrusleeping/169a28cffe1aedd4419d80aa62d361aa
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
	c, _ := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256).Sum([]byte("rock-paper-scissors-dht"))

	// First, announce ourselves as participating in this topic
	log.Println("Announcing ourselves...")
	tctx, _ := context.WithTimeout(ctx, time.Second*10)
	if err := dhtClient.Provide(tctx, c, true); err != nil {
		// Important to keep this as Non-Fatal error, otherwise it will fail for a node that behaves as well as bootstrap one
		log.Printf("Error: %s\n", err.Error())
	}

	// Now, look for others who have announced
	log.Println("Searching for other peers ...")
	peers, err := dhtClient.FindProviders(tctx, c)
	if err != nil {
		panic(err)
	}
	log.Printf("Found %d peers!\n", len(peers))

	// Now connect to them, so they are added to the PeerStore
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
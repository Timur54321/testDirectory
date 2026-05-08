package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/multiformats/go-multiaddr"
)

var (
	stream1 network.Stream
	stream2 network.Stream
	mu      sync.Mutex
)

var registeredConns map[string]network.Stream = make(map[string]network.Stream)

const keepAliveStreamProtocolID = "/keepAliveStream/1.0.0"
const getMediaProtocolID = "/getMedia/1.0.0"
const keyFile = "relay.key"

func handleStream(s network.Stream) {
	if stream1 == nil {
		stream1 = s

		return
	}

	if stream2 == nil {
		stream2 = s

		log.Println("stream2 registered")
	}

	go bridgeStreams(stream1, stream2)
}

func bridgeStreams(a, b network.Stream) {
	log.Println("Starting bidirectional bridge")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(b, a)
	}()

	go func() {
		defer wg.Done()
		io.Copy(a, b)
	}()

	wg.Wait()

	log.Println("Bridge finished, closing streams")

	a.Close()
	b.Close()

	mu.Lock()
	stream1 = nil
	stream2 = nil
	mu.Unlock()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := makeHost()
	if err != nil {
		log.Println(err)
		return
	}

	startPeer(ctx, h, handleStream)

	select {}
}

func makeHost() (host.Host, error) {
	priv := loadOrCreateKey()

	return libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/9000"), // 192.168.1.68
		libp2p.Identity(priv),
	)
}

func startPeer(_ context.Context, h host.Host, streamHandler network.StreamHandler) {

	h.SetStreamHandler(keepAliveStreamProtocolID, registerStream)
	h.SetStreamHandler(getMediaProtocolID, getPublicMedia)

	var port string
	for _, la := range h.Network().ListenAddresses() {
		if p, err := la.ValueForProtocol(multiaddr.P_TCP); err == nil {
			port = p
			break
		}
	}

	if port == "" {
		log.Println("was not able to find actual local port")
		return
	}

	log.Printf("Run './chat -d /ip4/127.0.0.1/tcp/%v/p2p/%s' on another console.\n", port, h.ID())
	log.Println("You can replace 127.0.0.1 with public IP as well.")
	log.Println("Waiting for incoming connection")
	log.Println()
}

func registerStream(s network.Stream) {
	log.Println("Registered a new connection")
	registeredConns[s.Conn().RemotePeer().String()] = s
}

func loadOrCreateKey() crypto.PrivKey {
	if data, err := os.ReadFile(keyFile); err == nil {
		b, _ := hex.DecodeString(string(data))
		k, _ := crypto.UnmarshalPrivateKey(b)
		return k
	}

	k, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	b, _ := crypto.MarshalPrivateKey(k)
	os.WriteFile(keyFile, []byte(hex.EncodeToString(b)), 0600)
	return k
}

func getPublicMedia(s network.Stream) {
	log.Println("trying to write something")
	for peerID, str := range registeredConns {
		if peerID == s.Conn().RemotePeer().String() {
			continue
		}
		log.Println("writen ok")
		str.Write([]byte("GF"))
		data, err := io.ReadAll(str)
		if err != nil {
			log.Println(err)
			return
		}
		s.Write(data)

		log.Println("Still waiting dawg")
	}

}

func initiateBridge(s network.Stream) {
	// get authorID
	// check if authorID has fileID
	// read for answer
	// generate transition ID 
	// 
	// add to map value
	// "transition id": {
	// 	stream1: s,
	// }
	// 
	// send transition ID to author
}

func takeFile(s network.Stream) {
	// get transition id from stream
	// check if it exists
	// get stream from existing object
	// run bridge streams()
}


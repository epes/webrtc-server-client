package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/epes/webrtc-server-client/client"
	"github.com/epes/webrtc-server-client/server"
)

func main() {
	rand.Seed(time.Now().Unix())
	port := flag.Int("port", 9090, "Port of the webrtc server")
	isClient := flag.Bool("c", false, "Is this a client?")
	isServer := flag.Bool("s", false, "Is this a server?")
	name := flag.String("name", randString(5), "Name of the client")
	flag.Parse()

	if *isClient && *isServer || !*isClient && !*isServer {
		fmt.Println(fmt.Errorf("define -c or -s to start a client or a server"))
		return
	}

	if *isClient {
		c(*port, *name)
	}

	if *isServer {
		s(*port)
	}
}

func s(port int) {
	fmt.Printf("[server] starting up server on port %d\n", port)
	server.Init(port)
}

func c(port int, name string) {
	fmt.Printf("[client] generating client '%s' connecting to port %d\n", name, port)
	client.Init(port, name)
}

func randString(length int) string {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes[i] = byte(65 + rand.Intn(26))
	}
	return string(bytes)
}

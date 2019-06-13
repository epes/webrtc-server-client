package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/epes/webrtc-server-client/client"
	"github.com/epes/webrtc-server-client/common"
	"github.com/epes/webrtc-server-client/server"
)

func main() {
	rand.Seed(time.Now().Unix())
	port := flag.Int("port", 9090, "Port of the webrtc server")
	isClient := flag.Bool("c", false, "Is this a client?")
	isServer := flag.Bool("s", false, "Is this a server?")
	name := flag.String("n", common.RandString(5), "Name of the client")
	group := flag.String("g", "example-group", "Group to connect to")
	flag.Parse()

	if *isClient && *isServer || !*isClient && !*isServer {
		log.Fatalln("Define -c for client or -s for server")
		return
	}

	if *isClient {
		c(*port, *name, *group)
	}

	if *isServer {
		s(*port)
	}
}

func s(port int) {
	log.Printf("[server] starting up server on port %d\n", port)
	server := server.NewServer(port)
	server.Start()
}

func c(port int, name string, group string) {
	log.Printf("[client] generating client '%s' connecting to group %s on port %d\n", name, group, port)
	client := client.NewClient(name, group, port)
	client.Start()
}

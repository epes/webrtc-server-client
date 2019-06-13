package server

import "log"

type fanout struct {
	// channel for broadcasting messages that get fanned out
	// to all the subscribed clients
	broadcast chan string
	// channel for connecting clients to the fanout
	connect chan chan string
	// channel for disconnecting clients from the fanout
	disconnect chan chan string
}

func newFanout(
	broadcast chan string,
	connect chan chan string,
	disconnect chan chan string,
) *fanout {
	return &fanout{
		broadcast:  broadcast,
		connect:    connect,
		disconnect: disconnect,
	}
}

func (f *fanout) begin() {
	clientSet := make(map[chan string]struct{})

	for {
		select {
		case client := <-f.connect:
			clientSet[client] = struct{}{}
		case client := <-f.disconnect:
			delete(clientSet, client)
		case message := <-f.broadcast:
			for client := range clientSet {
				log.Printf("[server][fanout] %s\n", message)
				client <- message
			}
		}
	}
}

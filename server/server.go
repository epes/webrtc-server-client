package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/epes/webrtc-server-client/common"
	"github.com/pion/webrtc"
)

type fanout struct {
	connectChan    chan chan string
	disconnectChan chan chan string
	messageChan    chan string
}

func Init(port int) {
	f := &fanout{
		connectChan:    make(chan chan string),
		disconnectChan: make(chan chan string),
		messageChan:    make(chan string),
	}
	go f.begin()

	http.HandleFunc("/offer", getHandleOffer(f))
	fmt.Printf("[server] setting up localhost:%d/offer\n", port)
	http.HandleFunc("/candidate", handleCandidate)
	fmt.Printf("[server] setting up localhost:%d/candidate\n", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	fmt.Printf("[server] Listening on localhost:%d\n", port)
}

func (f *fanout) begin() {
	clientSet := make(map[chan string]struct{})

	for {
		select {
		case client := <-f.connectChan:
			clientSet[client] = struct{}{}
		case client := <-f.disconnectChan:
			delete(clientSet, client)
		case message := <-f.messageChan:
			for client := range clientSet {
				fmt.Printf("[server] sending message: %s\n", message)
				client <- message
			}
		}
	}
}

func getHandleOffer(fout *fanout) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var offer common.Offer
		err := json.NewDecoder(r.Body).Decode(&offer)
		if err != nil {
			panic(err)
		}

		ansChan := make(chan webrtc.SessionDescription)
		errChan := make(chan error)
		closeChan := make(chan bool)

		go establishConnection(offer, fout, ansChan, errChan, closeChan)

		select {
		case err := <-errChan:
			panic(err)
		case answer := <-ansChan:
			err = json.NewEncoder(w).Encode(common.Answer{SDP: answer})
		}
	}
}

func establishConnection(
	offer common.Offer,
	fout *fanout,
	ansChan chan<- webrtc.SessionDescription,
	errChan chan<- error,
	closeChan chan bool,
) {
	messageChan := make(chan string)

	defer func() {
		// if it never connected, nothing will happen because
		// deleting a key that doesn't exist in a map is noop
		fout.disconnectChan <- messageChan
	}()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		errChan <- err
		return
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("[server] ICE Connection State has changed: %s\n", connectionState.String())

		switch state := connectionState.String(); state {
		case "connected":
			fmt.Printf("[server] connecting %s to the fanout\n", offer.ID)
			fout.connectChan <- messageChan
		case "disconnected":
			fmt.Printf("[server] disconnecting %s from the fanout\n", offer.ID)
			fout.disconnectChan <- messageChan
		}
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			fmt.Printf("[server] ICE Candidate: %s\n", candidate.Typ)
		}
	})

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("[server] new data channel %s\n", d.Label())

		d.OnOpen(func() {
			fmt.Printf("[server] data channel '%s' open\n", d.Label())
			fout.connectChan <- messageChan
		})

		d.OnClose(func() {
			fmt.Printf("[server] data channel '%s' closed\n", d.Label())
			fout.disconnectChan <- messageChan
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
			fout.messageChan <- string(msg.Data)
		})

		go func() {
			for {
				select {
				case message := <-messageChan:
					d.SendText(message)
				}
			}
		}()

	})

	fmt.Println("[server] setting offer to remote description")
	err = peerConnection.SetRemoteDescription(offer.SDP)
	if err != nil {
		errChan <- err
		return
	}

	fmt.Println("[server] generating answer")
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		errChan <- err
		return
	}

	fmt.Println("[server] setting answer to local description")
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		errChan <- err
		return
	}

	fmt.Println("[server] responding to offer with answer")
	ansChan <- answer

	select {
	// anything that closes the connection
	// should publish to this channel
	case <-closeChan:
	}
}

func handleCandidate(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Thanks for the candidate!\n")
}

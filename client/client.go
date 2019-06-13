package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/epes/webrtc-server-client/common"
	webrtc "github.com/pion/webrtc/v2"
)

type client struct {
	id    string
	group string
	port  int
}

func NewClient(id string, group string, port int) *client {
	return &client{
		id:    id,
		group: group,
		port:  port,
	}
}

func (c *client) Start() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("[%s] ICE Connection State has changed: %s\n", c.id, connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("[%s] ICE Candidate: %s\n", c.id, candidate.Typ)
		}
	})

	dataChannel.OnOpen(func() {
		log.Printf("[%s] data channel '%s' open\n", c.id, dataChannel.Label())

		i := 0

		for {
			time.Sleep(5 * time.Second)
			i++
			message := fmt.Sprintf("message #%d from %s", i, c.id)

			log.Printf("[%s] sending '%s'\n", c.id, message)
			dataChannel.Send([]byte(message))
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[%s] received message on channel '%s': %s\n", c.id, dataChannel.Label(), string(msg.Data))
	})

	log.Printf("[%s] generating offer\n", c.id)
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	log.Printf("[%s] setting local description\n", c.id)
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	log.Printf("[%s] exchanging SDP\n", c.id)
	answer, err := exchangeSDP(common.Offer{ID: c.id, Group: c.group, SDP: offer}, c.port)
	if err != nil {
		panic(err)
	}

	log.Printf("[%s] setting answer to remote description\n", c.id)
	err = peerConnection.SetRemoteDescription(answer.SDP)
	if err != nil {
		panic(err)
	}

	select {}
}

func exchangeSDP(offer common.Offer, port int) (common.Answer, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(offer)
	if err != nil {
		return common.Answer{}, err
	}

	url := fmt.Sprintf("http://localhost:%d/offer", port)

	log.Printf("[%s] sending offer to %s\n", offer.ID, url)
	resp, err := http.Post(url, "application/json; charset=utf8", buffer)
	if err != nil {
		return common.Answer{}, err
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			panic(closeErr)
		}
	}()

	var answer common.Answer
	err = json.NewDecoder(resp.Body).Decode(&answer)
	if err != nil {
		return common.Answer{}, err
	}

	log.Printf("[%s] received answer with streamID '%s'\n", offer.ID, answer.StreamID)

	return answer, nil
}

func exchangeCandidate(candidate common.ClientCandidate, port int) {
	// tbd
}

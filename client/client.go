package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/epes/webrtc-server-client/common"
	"github.com/pion/webrtc"
)

func Init(port int, name string) {
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
		fmt.Printf("[%s] ICE Connection State has changed: %s\n", name, connectionState.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			fmt.Printf("[%s] ICE Candidate: %s\n", name, candidate.Typ)
		}
	})

	dataChannel.OnOpen(func() {
		fmt.Printf("[%s] data channel '%s' open\n", name, dataChannel.Label())

		message := "login"

		fmt.Printf("[%s] sending '%s'\n", name, message)
		dataChannel.Send([]byte(message))
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("[%s] received message on channel '%s': %s\n", name, dataChannel.Label(), string(msg.Data))
	})

	fmt.Printf("[%s] generating offer\n", name)
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("[%s] setting local description\n", name)
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		panic(err)
	}

	fmt.Printf("[%s] exchanging SDP\n", name)
	answer, err := exchangeSDP(common.Offer{ID: name, SDP: offer}, port)
	if err != nil {
		panic(err)
	}

	fmt.Printf("[%s] setting answer to remote description\n", name)
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

	fmt.Printf("[%s] sending offer to %s\n", offer.ID, url)
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

	fmt.Printf("[%s] received answer\n", offer.ID)

	return answer, nil
}

func exchangeCandidate(candidate common.ClientCandidate, port int) {

}

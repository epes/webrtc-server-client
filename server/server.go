package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/epes/webrtc-server-client/common"
	webrtc "github.com/pion/webrtc/v2"
)

type Server struct {
	candidates  map[string]chan *webrtc.ICECandidate
	connections map[string]*webrtc.PeerConnection
	fanouts     map[string]*fanout
	port        int
}

func NewServer(port int) *Server {
	return &Server{
		candidates:  make(map[string]chan *webrtc.ICECandidate),
		connections: make(map[string]*webrtc.PeerConnection),
		fanouts:     make(map[string]*fanout),
		port:        port,
	}
}

func (s *Server) Start() {
	http.HandleFunc("/offer", s.handleOffer)
	log.Printf("[server] setting up localhost:%d/offer\n", s.port)

	http.HandleFunc("/candidate", s.handleCandidate)
	log.Printf("[server] setting up locahost:%d/candidate\n", s.port)

	log.Printf("[server] Listening on localhost:%d\n", s.port)
	http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) handleOffer(w http.ResponseWriter, req *http.Request) {
	var offer common.Offer
	err := json.NewDecoder(req.Body).Decode(&offer)
	if err != nil {
		log.Fatalln(err)
	}

	var fout *fanout
	// check if there is a fanout assigned to the group
	fout, ok := s.fanouts[offer.Group]
	// if there is not, create one and assign it
	if !ok {
		fout = newFanout(make(chan string), make(chan chan string), make(chan chan string))
		go fout.begin()

		s.fanouts[offer.Group] = fout
	}

	answer, err := s.establishConnection(&offer, fout)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewEncoder(w).Encode(*answer)
	if err != nil {
		log.Fatalln(err)
	}
}

func (s *Server) establishConnection(offer *common.Offer, fout *fanout) (*common.Answer, error) {
	streamID := common.RandString(10)
	answerChan := make(chan *webrtc.SessionDescription)
	errorChan := make(chan error)
	quit := make(chan struct{})

	go func() {
		// channel that will forward all messages to the client
		messageOut := make(chan string)

		defer func() {
			// cleanup

			// unregister peer connection from list of connections
			log.Printf("[server] unregistering '%s' connection '%s'", offer.ID, streamID)
			delete(s.connections, streamID)

			// unsubscribe from fanout messages
			log.Printf("[server] unsubscribing '%s' from the '%s' group\n", offer.ID, offer.Group)
			fout.disconnect <- messageOut
		}()

		// peer connection config
		config := webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		}

		// channel for registering ICE candidates as they trickle in
		iceCandidatesChan := make(chan *webrtc.ICECandidate, 32)

		peerConnection, err := webrtc.NewPeerConnection(config)
		if err != nil {
			errorChan <- err
			return
		}

		// register this connection under the generated streamID
		s.connections[streamID] = peerConnection
		// register the ice candidate channel under the generated streamID
		s.candidates[streamID] = iceCandidatesChan

		peerConnection.OnICEConnectionStateChange(func(connState webrtc.ICEConnectionState) {
			state := connState.String()

			log.Printf("[server] Connection state change with '%s': %s\n", offer.ID, state)

			if state == "closed" || state == "disconnected" {
				// if the connection is closed or disconnected, teardown the goroutine
				quit <- struct{}{}
			}
		})

		peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate != nil {
				log.Printf("[server] ICE Candidate: %s\n", candidate)
				iceCandidatesChan <- candidate
			}
		})

		peerConnection.OnICEGatheringStateChange(func(iceState webrtc.ICEGathererState) {
			state := iceState.String()
			log.Printf("[server] ICE Gathering state change with '%s': %s'\n", offer.ID, state)
		})

		peerConnection.OnSignalingStateChange(func(signalState webrtc.SignalingState) {
			state := signalState.String()
			log.Printf("[server] Signaling state change with '%s': %s'\n", offer.ID, state)
		})

		peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			log.Printf("[server] new data channel %s\n", dc.Label())

			dc.OnOpen(func() {
				log.Printf("[server] data channel '%s' open\n", dc.Label())
				log.Printf("[server] subscribing '%s' to the '%s' group\n", offer.ID, offer.Group)
				fout.connect <- messageOut
			})

			dc.OnClose(func() {
				log.Printf("[server] data channel '%s' closed\n", dc.Label())
				log.Printf("[server] unsubscribing '%s' from the '%s' group\n", offer.ID, offer.Group)
				fout.disconnect <- messageOut
			})

			dc.OnMessage(func(msg webrtc.DataChannelMessage) {
				data := string(msg.Data)
				log.Printf("[server][%s] %s: %s\n", offer.Group, offer.ID, data)
				fout.broadcast <- data
			})

			go func() {
				for {
					select {
					case message := <-messageOut:
						dc.SendText(message)
					}
				}
			}()
		})

		log.Printf("[server] setting offer from '%s' to remote description\n", offer.ID)
		err = peerConnection.SetRemoteDescription(offer.SDP)
		if err != nil {
			errorChan <- err
			return
		}

		log.Printf("[server] generating answer for '%s'", offer.ID)
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			errorChan <- err
			return
		}

		log.Printf("[server] setting answer for '%s' to local description", offer.ID)
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			errorChan <- err
			return
		}

		log.Printf("[server] responding to '%s' offer with an answer", offer.ID)
		answerChan <- &answer

		select {
		case <-quit:
		}
	}()

	select {
	case answer := <-answerChan:
		return &common.Answer{SDP: *answer, StreamID: streamID}, nil
	case err := <-errorChan:
		return nil, err
	}
}

func (s *Server) handleCandidate(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Thanks for the candidate\n")
}

package common

import (
	webrtc "github.com/pion/webrtc/v2"
)

type Offer struct {
	// unique identifier for the client
	ID string
	// identifier for the group the client wants to join
	Group string
	// session description for the offer
	SDP webrtc.SessionDescription
}

type Answer struct {
	// session description for the answer
	SDP webrtc.SessionDescription
	// unique identifier for the webrtc stream
	StreamID string
}

type ClientCandidate struct {
	ID        string
	Candidate webrtc.ICECandidate
	StreamID  string
}

type ServerCandidate struct {
	Candidate webrtc.ICECandidate
	Done      bool
}

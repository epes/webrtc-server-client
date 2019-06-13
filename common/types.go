package common

import (
	"github.com/pion/webrtc"
)

type Offer struct {
	ID  string
	SDP webrtc.SessionDescription
}

type Answer struct {
	SDP webrtc.SessionDescription
}

type ClientCandidate struct {
	ID        string
	Candidate webrtc.ICECandidate
}

type ServerCandidate struct {
	Candidate webrtc.ICECandidate
	Done      bool
}

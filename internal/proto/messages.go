package proto

// WS Client to Server
type Join struct {
	MatchID string `json:"matchId"`
	Token   string `json:"token"`
}

type Offer struct {
	SDP string `json:"sdp"`
}

type Answer struct {
	SDP string `json:"sdp"`
}

type ICE struct {
	Candidate string `json:"candidate"`
}

type Ready struct{}

type Resume struct{}

type AppendMirror struct {
	// Optional payload
}

type Heartbeat struct{}

// WS Server to Client
type Paired struct{}

type Correction struct {
	RewindTo int    `json:"rewind_to"`
	Snapshot string `json:"snapshot"`
}

type ClockCorrection struct {
	// Details
}

type Result struct {
	Result string `json:"result"`
	Reason string `json:"reason"`
}

type AdminNotice struct {
	Message string `json:"message"`
}

// P2P DataChannel
type Hello struct {
	Proto   int      `json:"proto"`
	MatchID string   `json:"matchId"`
	Side    string   `json:"side"`
	Caps    []string `json:"caps"`
}

type Move struct {
	Seq     int    `json:"seq"`
	UCI     string `json:"uci"`
	FEN     string `json:"fen"`
	MsWhite int    `json:"msWhite"`
	MsBlack int    `json:"msBlack"`
	Sig     string `json:"sig"`
}

type Resign struct {
	Seq int    `json:"seq"`
	By  string `json:"by"`
}

type DrawOffer struct {
	Seq int `json:"seq"`
}

type DrawResponse struct {
	Seq    int  `json:"seq"`
	Accept bool `json:"accept"`
}

type P2PHeartbeat struct {
	Seq int `json:"seq"`
	RTT int `json:"rtt"`
}

// TODO: Versioning with "proto":1

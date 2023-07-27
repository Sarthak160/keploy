package models


const ProtocolVersionNumber uint32 = 196608 // Replace with actual version number if different

type Packet interface{}

type PostgresReq struct {
	Identfier string `json:"identifier,omitempty"`
	Length uint32 `json:"length,omitempty"`
	Payload string `json:"payload,omitempty"`
	
}
type PostgresResp struct {
	Identfier string `json:"identifier,omitempty"`
	Length uint32 `json:"length,omitempty"`
	Payload string `json:"payload,omitempty"`
}

type StartupPacket struct {
	Length          uint32
	ProtocolVersion uint32
}

type RegularPacket struct {
	Identifier byte
	Length     uint32
	Payload    []byte
}
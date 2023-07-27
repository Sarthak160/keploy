package psqlparser

import (
	"net"

	// "time"
	// "sync"
	// "strings"

	"encoding/binary"
	// "encoding/json"

	// "fmt"
	"go.keploy.io/server/pkg/proxy/util"
	// "bytes"
	"encoding/base64"
	"errors"

	"github.com/jackc/pgproto3/v2"

	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"

	"go.uber.org/zap"
)

var Emoji = "\U0001F430" + " Keploy:"

func IsOutgoingPSQL(buffer []byte) bool {
	const ProtocolVersion = 0x00030000 // Protocol version 3.0

	if len(buffer) < 8 {
		// Not enough data for a complete header
		return false
	}

	// The first four bytes are the message length, but we don't need to check those

	// The next four bytes are the protocol version
	version := binary.BigEndian.Uint32(buffer[4:8])

	if version == 80877103 {
		return true
	}
	return version == ProtocolVersion
}

func ProcessOutgoingPSQL(requestBuffer []byte, clientConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger) {
	switch models.GetMode() {
	case models.MODE_RECORD:
		// *deps = append(*deps, encodeOutgoingHttp(requestBuffer,  clientConn,  destConn, logger))
		SaveOutgoingPSQL(requestBuffer, clientConn, destConn, logger, h)
		// encodeOutgoingPSQL(requestBuffer, clientConn, destConn, logger, h)
	case models.MODE_TEST:
		decodeOutgoingPSQL(requestBuffer, clientConn, destConn, h, logger)
	default:
		logger.Info(Emoji+"Invalid mode detected while intercepting outgoing http call", zap.Any("mode", models.GetMode()))
	}

}

type PSQLMessage struct {
	// Define fields to store the relevant information from the buffer
	ID      uint32
	Payload []byte
	Field1  string
	Field2  int
	// Add more fields as needed
}

func decodeBuffer(buffer []byte) (*PSQLMessage, error) {
	if len(buffer) < 6 {
		return nil, errors.New("invalid buffer length")
	}

	psqlMessage := &PSQLMessage{
		Field1: "test",
		Field2: 123,
	}

	// Decode the ID (4 bytes)
	psqlMessage.ID = binary.BigEndian.Uint32(buffer[:4])

	// Decode the payload length (2 bytes)
	payloadLength := binary.BigEndian.Uint16(buffer[4:6])

	// Check if the buffer contains the full payload
	if len(buffer[6:]) < int(payloadLength) {
		return nil, errors.New("incomplete payload in buffer")
	}

	// Extract the payload from the buffer
	psqlMessage.Payload = buffer[6 : 6+int(payloadLength)]

	return psqlMessage, nil
}

// func encodeOutgoingPSQL(requestBuffer []byte, clientConn, destConn net.Conn, logger *zap.Logger, h *hooks.Hook) []*models.Mock {

// 	logger.Info(Emoji + "Encoding outgoing Postgres call !!")
// 	// write the request message to the postgres server
// 	_, err := destConn.Write(requestBuffer)
// 	if err != nil {
// 		logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
// 		return nil
// 	}

// 	// // read reply message from the postgres server
// 	responseBuffer, err := util.ReadBytes(destConn)
// 	if err != nil {
// 		logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
// 		return nil
// 	}

// 	// write the reply to postgres client
// 	_, err = clientConn.Write(responseBuffer)
// 	if err != nil {
// 		logger.Error(Emoji+"failed to write the reply message to postgres client", zap.Error(err))
// 		return nil
// 	}

// 	// backend := pgproto3.NewBackend(pgproto3.NewChunkReader(clientConn), clientConn, clientConn)
// 	// frontend := pgproto3.NewFrontend(clientConn, clientConn)

// 	// frontend.Send(&pgproto3.Query{String: "SELECT * FROM users"})

// 	// Create a new PostgreSQL message reader
// 	backend := pgproto3.NewBackend(clientConn, clientConn)
// 	frontend := pgproto3.NewFrontend(destConn, destConn)
// 	// backend.ReceiveStartupMessage()
// 	// Loop to parse multiple messages in the buffer
// 	for {
// 		var msg pgproto3.FrontendMessage
// 		for {
// 			msg, err = backend.Receive()
// 			println(Emoji, "msg after  --------- ", msg, "|||", err)
// 			if err != nil {
// 				fmt.Println("Error parsing PostgreSQL message:", err)
// 				break
// 			}
// 			if msg == nil {
// 				break
// 			}
// 		}
// 		// Process the parsed message based on its type
// 		switch msg.(type) {
// 		case *pgproto3.Query:
// 			queryMsg := msg.(*pgproto3.Query)
// 			fmt.Println("Received Query:", queryMsg.String)
// 		case *pgproto3.Bind:
// 			bindMsg := msg.(*pgproto3.Bind)
// 			fmt.Println("Received Bind:", bindMsg.Parameters)
// 		// Add more cases for other message types if needed
// 		default:
// 			fmt.Println("Unsupported message type:", msg)
// 		}
// 		frontend.Send(msg)

// 		// // read the operation request message from the postgres client
// 		var msg2 pgproto3.BackendMessage
// 		for {
// 			msg2, err = frontend.Receive()
// 			if err != nil {
// 				fmt.Println("Error parsing PostgreSQL message from the server :", err)
// 				break
// 			}
// 		}
// 		backend.Send(msg2)

// 	}

// 	return nil
// }

func IdentifyPacket(data []byte) (models.Packet, error) {
	// At least 4 bytes are required to determine the length
	if len(data) < 4 {
		return nil, errors.New("data too short")
	}

	// Read the length (first 32 bits)
	length := binary.BigEndian.Uint32(data[:4])

	// If the length is followed by the protocol version, it's a StartupPacket
	if length > 4 && len(data) >= int(length) && binary.BigEndian.Uint32(data[4:8]) == models.ProtocolVersionNumber {
		return &models.StartupPacket{
			Length:          length,
			ProtocolVersion: binary.BigEndian.Uint32(data[4:8]),
		}, nil
	}

	// If we have an ASCII identifier, then it's likely a regular packet. Further validations can be added.
	if len(data) > 5 && len(data) >= int(length)+1 {
		return &models.RegularPacket{
			Identifier: data[4],
			Length:     length,
			Payload:    data[5 : length+1],
		}, nil
	}

	return nil, errors.New("unknown packet type or data too short for declared length")
}

func SaveOutgoingPSQL(requestBuffer []byte, clientConn, destConn net.Conn, logger *zap.Logger, h *hooks.Hook) []*models.Mock {

	logger.Info(Emoji + "Encoding outgoing Postgres call !!")
	// write the request message to the postgres server
	_, err := destConn.Write(requestBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return nil
	}

	// // read reply message from the postgres server
	responseBuffer, err := util.ReadBytes(destConn)
	if err != nil {
		logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return nil
	}

	// write the reply to postgres client
	_, err = clientConn.Write(responseBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the reply message to postgres client", zap.Error(err))
		return nil
	}

	backend := pgproto3.NewBackend(pgproto3.NewChunkReader(clientConn), clientConn, clientConn)
	frontend := pgproto3.NewFrontend(pgproto3.NewChunkReader(destConn), destConn, destConn)

	// may be bodylen 0 is reason
	x := 0
	// declare postgres mock here
	var deps []*models.Mock
	for {
		// read the operation request message from the postgres client
		x++
		msgRequestbuffer, err := util.ReadBytes(clientConn)
		if err != nil {
			logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
			return nil
		}
		if msgRequestbuffer == nil {
			logger.Info(Emoji + "msgRequestbuffer is nil")
			break
		}
		println(Emoji, "Inside for loop", string(msgRequestbuffer))
		msg, err := backend.Receive(msgRequestbuffer)
		encoded_req := msg.Encode(nil)
		println(encoded_req, "encoded_res", string(encoded_req))
		if err != nil {
			logger.Error(Emoji+"failed to receive the message from the client", zap.Error(err))
		}
		postgresMock := &models.Mock{
			Version: models.V1Beta1,
			Name:    "Postgres",
			Kind:    models.Postgres,
			Spec: models.MockSpec{
				PostgresReq: &models.PostgresReq{
					Identfier: "PostgresReq",
					Length:    uint32(len(msgRequestbuffer)),
					Payload:   base64.StdEncoding.EncodeToString(msgRequestbuffer),
				},
			},
		}

		if postgresMock != nil {
			deps = append(deps, postgresMock)
			h.AppendDeps(postgresMock)
		}

		// write the request message to postgres server
		_, err = destConn.Write(msgRequestbuffer)
		if err != nil {
			logger.Error(Emoji+"failed to write the request message to postgres server", zap.Error(err), zap.String("postgres server address", destConn.LocalAddr().String()))
			return nil
		}

		msgResponseBuffer, err := util.ReadBytes(destConn)
		if err != nil {
			logger.Error(Emoji+"failed to read the response message from postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
			return nil
		}
		msg2, err := frontend.Receive(msgResponseBuffer)
		if err != nil {
			logger.Error(Emoji+"failed to receive the response message from the server", zap.Error(err))
		}
		encoded_res := msg2.Encode(nil)
		println(encoded_res, "encoded_res", string(encoded_res))
		// _, err = clientConn.Write(encoded_res)
		println(Emoji, "After getting response from postgres server")
		postgresMock = &models.Mock{
			Version: models.V1Beta1,
			Name:    "Postgres",
			Kind:    models.Postgres,
			Spec: models.MockSpec{
				PostgresResp: &models.PostgresResp{
					Identfier: "PostgresResp",
					Length:    uint32(len(msgResponseBuffer)),
					Payload:   base64.StdEncoding.EncodeToString(msgResponseBuffer),
				},
			}}
		data, err := base64.StdEncoding.DecodeString(postgresMock.Spec.PostgresResp.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to decode the data", zap.Error(err))
		}
		logger.Info(Emoji+"Decoded data", zap.String("data", string(data)))

		if postgresMock != nil {
			deps = append(deps, postgresMock)
			h.AppendDeps(postgresMock)
		}
		// write the response message to postgres client
		_, err = clientConn.Write(data)
		if err != nil {
			logger.Error(Emoji+"failed to write the response wiremessage to postgres client ", zap.Error(err))
			return nil
		}

	}
	return nil
}

// we have to write a function to decode the response from the postgres server
func decodeOutgoingPSQL(requestBuffer []byte, clientConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger) {

}

func remainingBits(superset, subset []byte) []byte {
	// Find the length of the smaller array (subset).
	subsetLen := len(subset)

	// Initialize a result buffer to hold the differences.
	var difference []byte

	// Iterate through each byte in the 'subset' array.
	for i := 0; i < subsetLen; i++ {
		// Compare the bytes at the same index in both arrays.
		// If they are different, append the byte from 'superset' to the result buffer.
		if superset[i] != subset[i] {
			difference = append(difference, superset[i])
		}
	}

	// If 'superset' is longer than 'subset', append the remaining bytes to the result buffer.
	if len(superset) > subsetLen {
		difference = append(difference, superset[subsetLen:]...)
	}

	return difference
}

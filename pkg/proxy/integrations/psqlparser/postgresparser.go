package psqlparser

import (
	"net"
	"time"

	// "time"
	// "sync"
	// "strings"

	"encoding/binary"
	// "encoding/json"
	"encoding/base64"
	// "fmt"
	"go.keploy.io/server/pkg/proxy/util"
	// "bytes"

	"errors"
	"fmt"

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
		SaveOutgoingPSQL(requestBuffer, clientConn, destConn, logger, h)
		// startProxy(requestBuffer, clientConn, destConn, logger, h)
		encodeOutgoingPSQL(requestBuffer, clientConn, destConn, logger, h)
	case models.MODE_TEST:
		decodeOutgoingPSQL2(requestBuffer, clientConn, destConn, h, logger)
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

func encodeOutgoingPSQL(requestBuffer []byte, clientConn, destConn net.Conn, logger *zap.Logger, h *hooks.Hook) {

	_, err := destConn.Write(requestBuffer)

	if err != nil {
		logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return
	}

	if err != nil {
		logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return
	}

	backend := pgproto3.NewBackend(pgproto3.NewChunkReader(destConn), destConn)
	frontend := pgproto3.NewFrontend(pgproto3.NewChunkReader(clientConn), clientConn)
	backend.ReceiveStartupMessage()
	for {
		msg, err := frontend.Receive()
		if err != nil {
			fmt.Printf("Failed to receive message from client: %v", err)
			return
		}

		err = backend.Send(msg)
		if err != nil {
			fmt.Printf("Failed to send message to server: %v", err)
			return
		}

		response, err := backend.Receive()
		if err != nil {
			fmt.Printf("Failed to receive message from server: %v", err)
			return
		}

		err = frontend.Send(response)
		if err != nil {
			fmt.Printf("Failed to send message to client: %v", err)
			return
		}
	}

}

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

	// backend := pgproto3.NewBackend(pgproto3.NewChunkReader(clientConn), clientConn, clientConn)
	// frontend := pgproto3.NewFrontend(pgproto3.NewChunkReader(destConn), destConn, destConn)
	logger.Info(Emoji + "Encoding outgoing Postgres call !!")
	// write the request message to the postgres server

	_, err := destConn.Write(requestBuffer)

	if err != nil {
		logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return nil
	}
	// msg, err := backend.ReceiveStartupMessage2(requestBuffer)
	// if err != nil {
	// 	logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
	// 	return nil
	// }

	// pre_encoded_req := msg.Encode(nil)

	// println(Emoji, "pre_encoded_req", string(pre_encoded_req))

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
	println(Emoji, "responseBuffer", string(responseBuffer))

	// resMsg, err := frontend.Receive(responseBuffer)
	// if err != nil {
	// 	logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
	// 	return nil
	// }

	// pre_encoded_res := resMsg.Encode(nil)

	// println(Emoji, "pre_encoded_res", string(pre_encoded_res))
	postgresMock := &models.Mock{
		Version: models.V1Beta1,
		Name:    "config",
		Kind:    models.Postgres,
		Spec: models.MockSpec{
			PostgresReq: &models.Backend{
				Identfier: "PostgresReq",
				Length:    uint32(len(responseBuffer)),
				Payload:   base64.StdEncoding.EncodeToString(responseBuffer),
			},
		},
	}

	if postgresMock != nil {
		h.AppendMocks(postgresMock)
	}

	// may be bodylen 0 is reason
	x := 0
	// declare postgres mock here
	var deps []*models.Mock
	for {
		// read request message from the postgres client and see if it's authentication buffer
		// if its auth buffer then just save it in global config
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
		// msg, err := backend.Receive(msgRequestbuffer)
		// encoded_req := msg.Encode(nil)
		// println(encoded_req, "encoded_req", string(encoded_req))
		// if err != nil {
		// 	logger.Error(Emoji+"failed to receive the message from the client", zap.Error(err))
		// }
		name := "mocks"
		// check the packet type and add config name accordingly
		if x == 1 {
			logger.Info(Emoji + "This is an auth req")
			name = "config"
		}

		// for making readable first identify message type and add the Unmarshaled value for that query object
		postgresMock := &models.Mock{
			Version: models.V1Beta1,
			Name:    name,
			Kind:    models.Postgres,
			Spec: models.MockSpec{
				PostgresReq: &models.Backend{
					Identfier: "PostgresReq",
					Length:    uint32(len(msgRequestbuffer)),
					Payload:   base64.StdEncoding.EncodeToString(msgRequestbuffer),
				},
			},
		}

		if postgresMock != nil {
			deps = append(deps, postgresMock)
			h.AppendMocks(postgresMock)
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
		// msg2, err := frontend.Receive(msgResponseBuffer)

		// if err != nil {
		// 	logger.Error(Emoji+"failed to receive the response message from the server", zap.Error(err))
		// }
		// encoded_res := msg2.Encode(nil)
		// println(encoded_res, "encoded_res", string(encoded_res))
		// _, err = clientConn.Write(encoded_res)
		println(Emoji, "After getting response from postgres server")

		name = "mocks"

		if x == 1 {
			logger.Info(Emoji + "This is an auth req")
			name = "config"
		}

		postgresMock = &models.Mock{
			Version: models.V1Beta1,
			Name:    name,
			Kind:    models.Postgres,
			Spec: models.MockSpec{
				PostgresReq: &models.Backend{
					Identfier: "PostgresReq",
					Length:    uint32(len(msgResponseBuffer)),
					Payload:   base64.StdEncoding.EncodeToString(msgResponseBuffer),
				},
			}}

		data, err := base64.StdEncoding.DecodeString(postgresMock.Spec.PostgresReq.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to decode the data", zap.Error(err))
		}
		logger.Info(Emoji+"Decoded data", zap.String("data", string(data)))

		if postgresMock != nil {
			println(Emoji, "Inside if of second mock creation")
			deps = append(deps, postgresMock)
			h.AppendMocks(postgresMock)
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
	// decode the request buffer
	configMocks := h.GetConfigMocks()

	tcsMocks := h.GetTcsMocks()
	println(len(tcsMocks), "len of tcs mocks")
	println(len(configMocks), "len of config mocks")
	// backend := pgproto3.NewBackend(pgproto3.NewChunkReader(clientConn), clientConn, clientConn)
	// frontend := pgproto3.NewFrontend(pgproto3.NewChunkReader(destConn), destConn, destConn)
	logger.Info(Emoji + "Encoding outgoing Postgres call !!")

	// // read reply message from the postgres server
	encoded := configMocks[0].Spec.PostgresReq.Payload
	println(encoded, "encoded")
	responseBuffer, err := PostgresDecoder(encoded) //util.ReadBytes(destConn)
	if err != nil {
		logger.Error(Emoji+"failed to read reply from the postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return
	}

	// write the reply to postgres client
	_, err = clientConn.Write(responseBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the reply message to postgres client", zap.Error(err))
		return
	}

	// may be bodylen 0 is reason
	x, tc_count := 0, 0
	// declare postgres mock here
	// var deps []*models.Mock
	var msgRequestbuffer, msgResponseBuffer []byte
	for {
		// every time we have to read the request buffer from the client and then compare that request buffer with the mocks available in the mock file ..
		// read request message from the postgres client and see if it's authentication buffer
		// if its auth buffer then just save it in global config
		x++
		if tc_count == 3 {
			time.Sleep(5 * time.Second)
			break
		}
		if x == 1 {
			println("Inside x==1")
			msgRequestbuffer, err = PostgresDecoder(configMocks[x].Spec.PostgresReq.Payload)
			if err != nil {
				logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
				return
			}
			tc_count = 0
		}

		if msgRequestbuffer == nil {
			logger.Info(Emoji + "msgRequestbuffer is nil")
			break
		}

		// write the request message to postgres server
		msgRequestbuffer, err = PostgresDecoder(tcsMocks[tc_count].Spec.PostgresReq.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
		}

		if x == 1 {
			msgResponseBuffer, err = PostgresDecoder(configMocks[x+1].Spec.PostgresReq.Payload)
			if err != nil {
				logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
				return
			}
			tc_count = 0
		}

		tc_count++

		msgResponseBuffer, err = PostgresDecoder(tcsMocks[tc_count].Spec.PostgresReq.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
			return
		}
		// write the response message to postgres client
		_, err = clientConn.Write(msgResponseBuffer)
		if err != nil {
			logger.Error(Emoji+"failed to write the response wiremessage to postgres client ", zap.Error(err))
			return
		}
		tc_count++

	}
	return
}

func decodeOutgoingPSQL2(requestBuffer []byte, clientConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger) {
	// decode the request buffer
	configMocks := h.GetConfigMocks()

	tcsMocks := h.GetTcsMocks()
	println(len(tcsMocks), "len of tcs mocks")
	println(len(configMocks), "len of config mocks")
	// backend := pgproto3.NewBackend(pgproto3.NewChunkReader(clientConn), clientConn)
	// frontend := pgproto3.NewFrontend(pgproto3.NewChunkReader(destConn), destConn, destConn)
	logger.Info(Emoji + "Encoding outgoing Postgres call !!")
	// write the request message to the postgres server

	// _, err := destConn.Write(requestBuffer)
	encode, err := PostgresDecoder(configMocks[0].Spec.PostgresReq.Payload)
	if err != nil {
		logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
		return
	}

	// write the reply to postgres client
	_, err = clientConn.Write(encode)
	if err != nil {
		logger.Error(Emoji+"failed to write the reply message to postgres client", zap.Error(err))
		return
	}

	// may be bodylen 0 is reason
	x := 0
	y := 0
	// declare postgres mock here
	var msgRequestbuffer []byte
	for {
		// read request message from the postgres client and see if it's authentication buffer
		// if its auth buffer then just save it in global config
		x++
		if y == len(tcsMocks) {
			break
		}
		// read untill you get the data
		// for {
		msgRequestbuffer, _, err = util.ReadBytes1(clientConn)
		if err != nil {
			logger.Error(Emoji+"failed to read the message from the postgres client", zap.Error(err))
			return
		}
		// 	if len(msgRequestbuffer) != 0 {
		// 		break
		// 	}
		// }

		// check the packet type and add config name accordingly
		if x == 1 {
			logger.Info(Emoji + "This is an auth req")

			config2, err := PostgresDecoder(configMocks[x].Spec.PostgresReq.Payload)
			if err != nil {
				logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
				return
			}
			if string(msgRequestbuffer) == string(config2) {
				println(Emoji, "Auth req matched")
			}
			msgResponseBuffer, err := PostgresDecoder(configMocks[x+1].Spec.PostgresReq.Payload)
			if err != nil {
				logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
				return
			}
			_, err = clientConn.Write(msgResponseBuffer)
			if err != nil {
				logger.Error(Emoji+"failed to write the response wiremessage to postgres client ", zap.Error(err))
				return
			}
			continue
		}

		mock2, err := PostgresDecoder(tcsMocks[y].Spec.PostgresReq.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
			return
		}
		if string(msgRequestbuffer) == string(mock2) {
			println(Emoji, "After Auth req matched")
		}
		msgResponseBuffer, err := PostgresDecoder(tcsMocks[y+1].Spec.PostgresReq.Payload)
		if err != nil {
			logger.Error(Emoji+"failed to write the request buffer to postgres server", zap.Error(err), zap.String("postgres server address", destConn.RemoteAddr().String()))
			return
		}
		_, err = clientConn.Write(msgResponseBuffer)
		if err != nil {
			logger.Error(Emoji+"failed to write the response wiremessage to postgres client ", zap.Error(err))
			return
		}
		y++
		y++
	}
	// return
}

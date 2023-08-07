package psqlparser

import (
	"encoding/base64"
	"fmt"
	"net"
	// "time"
	// "sync"
	// "strings"

	// "encoding/json"

	// "fmt"
	"go.keploy.io/server/pkg/proxy/util"
	// "bytes"
	"encoding/binary"

	"errors"
	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"
	"go.uber.org/zap"
	"io"
)

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

func startProxy(buffer []byte, clientConn, destConn net.Conn, logger *zap.Logger, h *hooks.Hook) []*models.Mock {

	logger.Info(Emoji + "Encoding outgoing Postgres call !!")
	// write the request message to the postgres server
	_, err := destConn.Write(buffer)
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

	clientToDest := make(chan []byte)
	destToClient := make(chan []byte)

	// Client to Destination communication
	go func() {
		defer clientConn.Close()
		for data := range clientToDest {
			_, err := destConn.Write(data)
			if err != nil {
				fmt.Printf("Error writing to destination server: %v", err)
				return
			}
		}
	}()

	// Destination to Client communication
	go func() {
		defer destConn.Close()
		for {
			bytesRead, err := destConn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading from destination server: %v", err)
				}
				return
			}
			destToClient <- buffer[:bytesRead]
		}
	}()

	// Main loop for client-destination server communication
	for {
		bytesRead, err := clientConn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading from client: %v", err)
			}
			return nil
		}
		clientToDest <- buffer[:bytesRead]

		// Read from destToClient channel and send back to client
		response := <-destToClient
		_, err = clientConn.Write(response)
		if err != nil {
			fmt.Printf("Error writing to client: %v", err)
			return nil
		}
	}
}

func PostgresDecoder(encoded string) ([]byte, error) {
	// decode the base 64 encoded string to buffer ..

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		fmt.Println(Emoji+"failed to decode the data", err)
		return nil, err
	}
	println("Decoded data is :", string(data))
	return data, nil
}

func PostgresEncoder(buffer []byte) string {
	// encode the buffer to base 64 string ..
	encoded := base64.StdEncoding.EncodeToString(buffer)
	return encoded
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

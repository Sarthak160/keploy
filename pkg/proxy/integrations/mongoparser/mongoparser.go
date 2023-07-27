package mongoparser

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"
	"go.keploy.io/server/pkg/proxy/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
	"go.uber.org/zap"
)

var Emoji = "\U0001F430" + " Keploy:"

// IsOutgoingMongo function determines if the outgoing network call is Mongo by comparing the
// message format with that of a mongo wire message.
func IsOutgoingMongo(buffer []byte) bool {
	if len(buffer) < 4 {
		return false
	}
	messageLength := binary.LittleEndian.Uint32(buffer[0:4])
	return int(messageLength) == len(buffer)
}

func ProcessOutgoingMongo(requestBuffer []byte, clientConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger) {
	switch models.GetMode() {
	case models.MODE_RECORD:
		capturedDeps := encodeOutgoingMongo(requestBuffer, clientConn, destConn, logger)
		// *deps = append(*deps, capturedDeps...)
		for _, v := range capturedDeps {
			h.AppendDeps(v)
		}
	case models.MODE_TEST:
		decodeOutgoingMongo(requestBuffer, clientConn, destConn, h, logger)
	default:
	}
}

func decodeOutgoingMongo(requestBuffer []byte, clientConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger) {

	_, requestHeader, _, err := Decode(requestBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to decode the mongo request wiremessage", zap.Error(err))
		return
	}

	if h.GetDepsSize() <= 1 {
		return
	}

	mongoSpec := h.FetchDep(0)

	replySpec, ok := mongoSpec.Spec.MongoResponse.(*models.MongoOpReply)
	if !ok {
		logger.Error(Emoji + "failed to decode the yaml for mongo OpReply wiremessage")
		return
	}
	
	// opReply
	replyDocs := []bsoncore.Document{}
	for _, v := range replySpec.Documents {
		var unmarshaledDoc bsoncore.Document
		err = bson.UnmarshalExtJSON([]byte(v), false, &unmarshaledDoc)
		if err != nil {
			logger.Error(Emoji+"failed to unmarshal the recorded document string of OpReply", zap.Error(err))
			return
		}
		replyDocs = append(replyDocs, unmarshaledDoc)
	}

	var heathCheckReplyBuffer []byte
	heathCheckReplyBuffer = wiremessage.AppendHeader(heathCheckReplyBuffer, mongoSpec.Spec.MongoResponseHeader.Length, requestHeader.RequestID, requestHeader.ResponseTo, mongoSpec.Spec.MongoResponseHeader.Opcode)
	heathCheckReplyBuffer = wiremessage.AppendReplyFlags(heathCheckReplyBuffer, wiremessage.ReplyFlag(replySpec.ResponseFlags))
	heathCheckReplyBuffer = wiremessage.AppendReplyCursorID(heathCheckReplyBuffer, replySpec.CursorID)
	heathCheckReplyBuffer = wiremessage.AppendReplyStartingFrom(heathCheckReplyBuffer, replySpec.StartingFrom)
	heathCheckReplyBuffer = wiremessage.AppendReplyNumberReturned(heathCheckReplyBuffer, replySpec.NumberReturned)
	for _, doc := range replyDocs {
		heathCheckReplyBuffer = append(heathCheckReplyBuffer, doc...)
	}

	_, err = clientConn.Write(heathCheckReplyBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the health check reply to mongo client", zap.Error(err))
		return
	}

	operationBuffer, err := util.ReadBytes(clientConn)
	if err != nil {
		logger.Error(Emoji+"failed to read the mongo wiremessage for operation query", zap.Error(err))
		return
	}

	opr1, _, _, err := Decode(operationBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to decode the mongo operation query", zap.Error(err))
		return
	}
	if strings.Contains(opr1.String(), "hello") {
		return
	}

	mongoSpec1 := h.FetchDep(1)

	msgSpec, ok := mongoSpec1.Spec.MongoResponse.(*models.MongoOpMessage)
	if !ok {
		logger.Error(Emoji + "failed to decode the yaml for mongo OpMessage wiremessage response")
		return
	}
	msg := &opMsg{
		flags:    wiremessage.MsgFlag(msgSpec.FlagBits),
		checksum: uint32(msgSpec.Checksum),
		sections: []opMsgSection{},
	}

	for i, v := range msgSpec.Sections {
		if strings.Contains(v, "SectionSingle identifier") {
			var identifier string
			var msgsStr string
			_, err := fmt.Sscanf(msgSpec.Sections[i], "{ SectionSingle identifier: %s, msgs: [%s] }", &identifier, &msgsStr)
			if err != nil {
				logger.Error(Emoji+"failed to extract the msg section from recorded message", zap.Error(err))
				return
			}
			msgs := strings.Split(msgsStr, ", ")
			docs := []bsoncore.Document{}
			for _, v := range msgs {
				var unmarshaledDoc bsoncore.Document
				err = bson.UnmarshalExtJSON([]byte(v), false, &unmarshaledDoc)
				if err != nil {
					logger.Error(Emoji+"failed to unmarshal the recorded document string of OpMsg", zap.Error(err))
					return
				}
				docs = append(docs, unmarshaledDoc)
			}

			msg.sections = append(msg.sections, &opMsgSectionSequence{
				identifier: identifier,
				msgs:       docs,
			})
		} else {
			var sectionStr string
			_, err := fmt.Sscanf(msgSpec.Sections[0], "{ SectionSingle msg: %s }", &sectionStr)
			if err != nil {
				logger.Error(Emoji+"failed to extract the msg section from recorded message single section", zap.Error(err))
				return
			}
			var unmarshaledDoc bsoncore.Document
			err = bson.UnmarshalExtJSON([]byte(sectionStr), false, &unmarshaledDoc)
			if err != nil {
				logger.Error(Emoji+"failed to unmarshal the recorded document string of OpMsg", zap.Error(err))
				return
			}
			msg.sections = append(msg.sections, &opMsgSectionSingle{
				msg: unmarshaledDoc,
			})
		}
	}

	_, err = clientConn.Write(msg.Encode(mongoSpec1.Spec.MongoRequestHeader.ResponseTo))
	if err != nil {
		logger.Error(Emoji+"failed to write the OpMsg response for the mongo operation", zap.Error(err))
		return
	}
	h.PopFront()
	h.PopFront()
}

func encodeOutgoingMongo(requestBuffer []byte, clientConn, destConn net.Conn, logger *zap.Logger) []*models.Mock {

	// write the request message to the mongo server
	_, err := destConn.Write(requestBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the request buffer to mongo server", zap.Error(err), zap.String("mongo server address", destConn.RemoteAddr().String()))
		return nil
	}

	// read reply message from the mongo server
	responseBuffer, err := util.ReadBytes(destConn)
	if err != nil {
		logger.Error(Emoji+"failed to read reply from the mongo server", zap.Error(err), zap.String("mongo server address", destConn.RemoteAddr().String()))
		return nil
	}

	// write the reply to mongo client
	_, err = clientConn.Write(responseBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the reply message to mongo client", zap.Error(err))
		return nil
	}

	// read the operation request message from the mongo client
	msgRequestbuffer, err := util.ReadBytes(clientConn)
	if err != nil {
		logger.Error(Emoji+"failed to read the message from the mongo client", zap.Error(err))
		return nil
	}

	opr1, _, _, err := Decode(msgRequestbuffer)
	if err != nil {
		// logger.Error("failed to decode t")
		return nil
	}
	
	// write the request message to mongo server
	_, err = destConn.Write(msgRequestbuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the request message to mongo server", zap.Error(err), zap.String("mongo server address", destConn.LocalAddr().String()))
		return nil
	}

	// read the response message form the mongo server
	msgResponseBuffer, err := util.ReadBytes(destConn)
	if err != nil {
		logger.Error(Emoji+"failed to read the response message from mongo server", zap.Error(err), zap.String("mongo server address", destConn.RemoteAddr().String()))
		return nil
	}

	// write the response message to mongo client
	_, err = clientConn.Write(msgResponseBuffer)
	if err != nil {
		logger.Error(Emoji+"failed to write the response wiremessage to mongo client", zap.Error(err))
		return nil
	}

	// capture if the wiremessage is a mongo operation call
	if !strings.Contains(opr1.String(), "hello") {
		deps := []*models.Mock{}

		// decode the mongo binary request wiremessage
		opr, requestHeader, mongoRequest, err := Decode((requestBuffer))
		if err != nil {
			logger.Error(Emoji+"failed tp decode the mongo wire message from the client", zap.Error(err))
			return nil
		}

		// decode the mongo binary response wiremessage
		op, responseHeader, mongoResp, err := Decode(responseBuffer)
		if err != nil {
			logger.Error(Emoji+"failed to decode the mongo wire message from the destination server", zap.Error(err))
			return nil
		}

		replyDocs := []string{}
		for _, v := range op.(*opReply).documents {
			replyDocs = append(replyDocs, v.String())
		}
		meta1 := map[string]string{
			"operation": opr.String(),
		}
		mongoMock := &models.Mock{
			Version: models.V1Beta2,
			Kind:    models.Mongo,
			Name:    "",
			Spec: models.MockSpec{
				Metadata:            meta1,
				MongoRequestHeader:  &requestHeader,
				MongoResponseHeader: &responseHeader,
				MongoRequest:        mongoRequest,
				MongoResponse:       mongoResp,
				Created:             time.Now().Unix(),
			},
		}

		if mongoMock != nil {
			deps = append(deps, mongoMock)
		}

		meta := map[string]string{
			"operation": opr1.String(),
		}

		opr, msgRequestHeader, mongoMsgRequest, err := Decode((msgRequestbuffer))
		if err != nil {
			logger.Error(Emoji+"failed tp decode the mongo wire message from the client", zap.Error(err))
			return nil
		}

		op, msgResponseHeader, mongoMsgResponse, err := Decode(msgResponseBuffer)
		if err != nil {
			logger.Error(Emoji+"failed to decode the mongo wire message from the destination server", zap.Error(err))
			return nil
		}
		mongoMock = &models.Mock{
			Version: models.V1Beta2,
			Kind:    models.Mongo,
			Name:    "",
			Spec: models.MockSpec{
				Metadata:            meta,
				MongoRequestHeader:  &msgRequestHeader,
				MongoResponseHeader: &msgResponseHeader,
				MongoRequest:        mongoMsgRequest,
				MongoResponse:       mongoMsgResponse,
				Created:             time.Now().Unix(),
			},
		}

		if mongoMock != nil {
			deps = append(deps, mongoMock)
		}
		 
	}

	return nil
}

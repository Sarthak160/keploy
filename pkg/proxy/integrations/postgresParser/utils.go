package postgresparser

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"strings"

	"errors"
	"fmt"

	"github.com/jackc/pgproto3/v2"
	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"
	"go.keploy.io/server/pkg/proxy/util"
	"go.uber.org/zap"
)

func PostgresDecoder(encoded string) ([]byte, error) {
	// decode the base 64 encoded string to buffer ..
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func PostgresDecoderFrontend(response models.Frontend) ([]byte, error) {
	// println("Inside PostgresDecoderFrontend")
	var resbuffer []byte
	// list of packets available in the buffer
	packets := response.PacketTypes
	var cc, dtr, ps int = 0, 0, 0
	for _, packet := range packets {
		var msg pgproto3.BackendMessage

		switch packet {
		case string('1'):
			msg = &pgproto3.ParseComplete{}
		case string('2'):
			msg = &pgproto3.BindComplete{}
		case string('3'):
			msg = &pgproto3.CloseComplete{}
		case string('A'):
			msg = &pgproto3.NotificationResponse{
				PID:     response.NotificationResponse.PID,
				Channel: response.NotificationResponse.Channel,
				Payload: response.NotificationResponse.Payload,
			}
		case string('c'):
			msg = &pgproto3.CopyDone{}
		case string('C'):
			msg = &pgproto3.CommandComplete{
				CommandTag: response.CommandCompletes[cc].CommandTag,
			}
			cc++
		case string('d'):
			msg = &pgproto3.CopyData{
				Data: response.CopyData.Data,
			}
		case string('D'):
			msg = &pgproto3.DataRow{
				RowValues: response.DataRows[dtr].RowValues,
				Values:    response.DataRows[dtr].Values,
			}
			dtr++
		case string('E'):
			msg = &pgproto3.ErrorResponse{
				Severity:         response.ErrorResponse.Severity,
				Code:             response.ErrorResponse.Code,
				Message:          response.ErrorResponse.Message,
				Detail:           response.ErrorResponse.Detail,
				Hint:             response.ErrorResponse.Hint,
				Position:         response.ErrorResponse.Position,
				InternalPosition: response.ErrorResponse.InternalPosition,
				InternalQuery:    response.ErrorResponse.InternalQuery,
				Where:            response.ErrorResponse.Where,
				SchemaName:       response.ErrorResponse.SchemaName,
				TableName:        response.ErrorResponse.TableName,
				ColumnName:       response.ErrorResponse.ColumnName,
				DataTypeName:     response.ErrorResponse.DataTypeName,
				ConstraintName:   response.ErrorResponse.ConstraintName,
				File:             response.ErrorResponse.File,
				Line:             response.ErrorResponse.Line,
				Routine:          response.ErrorResponse.Routine,
			}
		case string('G'):
			msg = &pgproto3.CopyInResponse{
				OverallFormat:     response.CopyInResponse.OverallFormat,
				ColumnFormatCodes: response.CopyInResponse.ColumnFormatCodes,
			}
		case string('H'):
			msg = &pgproto3.CopyOutResponse{
				OverallFormat:     response.CopyOutResponse.OverallFormat,
				ColumnFormatCodes: response.CopyOutResponse.ColumnFormatCodes,
			}
		case string('I'):
			msg = &pgproto3.EmptyQueryResponse{}
		case string('K'):
			msg = &pgproto3.BackendKeyData{
				ProcessID: response.BackendKeyData.ProcessID,
				SecretKey: response.BackendKeyData.SecretKey,
			}
		case string('n'):
			msg = &pgproto3.NoData{}
		case string('N'):
			msg = &pgproto3.NoticeResponse{
				Severity:         response.NoticeResponse.Severity,
				Code:             response.NoticeResponse.Code,
				Message:          response.NoticeResponse.Message,
				Detail:           response.NoticeResponse.Detail,
				Hint:             response.NoticeResponse.Hint,
				Position:         response.NoticeResponse.Position,
				InternalPosition: response.NoticeResponse.InternalPosition,
				InternalQuery:    response.NoticeResponse.InternalQuery,
				Where:            response.NoticeResponse.Where,
				SchemaName:       response.NoticeResponse.SchemaName,
				TableName:        response.NoticeResponse.TableName,
				ColumnName:       response.NoticeResponse.ColumnName,
				DataTypeName:     response.NoticeResponse.DataTypeName,
				ConstraintName:   response.NoticeResponse.ConstraintName,
				File:             response.NoticeResponse.File,
				Line:             response.NoticeResponse.Line,
				Routine:          response.NoticeResponse.Routine,
			}

		case string('R'):
			switch response.AuthType {
			case AuthTypeOk:
				msg = &pgproto3.AuthenticationOk{}
			case AuthTypeCleartextPassword:
				msg = &pgproto3.AuthenticationCleartextPassword{}
			case AuthTypeMD5Password:
				msg = &pgproto3.AuthenticationMD5Password{}
			case AuthTypeSCMCreds:
				return nil, errors.New("AuthTypeSCMCreds is unimplemented")
			case AuthTypeGSS:
				return nil, errors.New("AuthTypeGSS is unimplemented")
			case AuthTypeGSSCont:
				msg = &pgproto3.AuthenticationGSSContinue{}
			case AuthTypeSSPI:
				return nil, errors.New("AuthTypeSSPI is unimplemented")
			case AuthTypeSASL:
				msg = &pgproto3.AuthenticationSASL{}
			case AuthTypeSASLContinue:
				msg = &pgproto3.AuthenticationSASLContinue{}
			case AuthTypeSASLFinal:
				msg = &pgproto3.AuthenticationSASLFinal{}
			default:
				return nil, fmt.Errorf("unknown authentication type: %d", response.AuthType)
			}

		case string('s'):
			msg = &pgproto3.PortalSuspended{}
		case string('S'):
			msg = &pgproto3.ParameterStatus{
				Name:  response.ParameterStatusCombined[ps].Name,
				Value: response.ParameterStatusCombined[ps].Value,
			}
			ps++

		case string('t'):
			msg = &pgproto3.ParameterDescription{
				ParameterOIDs: response.ParameterDescription.ParameterOIDs,
			}
		case string('T'):
			msg = &pgproto3.RowDescription{
				Fields: response.RowDescription.Fields,
			}
		case string('V'):
			msg = &pgproto3.FunctionCallResponse{
				Result: response.FunctionCallResponse.Result,
			}
		case string('W'):
			msg = &pgproto3.CopyBothResponse{
				OverallFormat:     response.CopyBothResponse.OverallFormat,
				ColumnFormatCodes: response.CopyBothResponse.ColumnFormatCodes,
			}
		case string('Z'):
			msg = &pgproto3.ReadyForQuery{
				TxStatus: response.ReadyForQuery.TxStatus,
			}
		default:
			return nil, fmt.Errorf("unknown message type: %q", packet)
		}

		encoded := msg.Encode([]byte{})
		resbuffer = append(resbuffer, encoded...)
	}
	return resbuffer, nil
}

func PostgresDecoderBackend(request models.Backend) ([]byte, error) {
	// take each object , try to make it frontend or backend message so that it can call it's corresponding encode function
	// and then append it to the buffer, for a particular mock ..

	var reqbuffer []byte
	// list of packets available in the buffer
	var b, e, p int = 0, 0, 0
	packets := request.PacketTypes
	for _, packet := range packets {
		var msg pgproto3.FrontendMessage
		switch packet {
		case string('B'):
			msg = &pgproto3.Bind{
				DestinationPortal:    request.Binds[b].DestinationPortal,
				PreparedStatement:    request.Binds[b].PreparedStatement,
				ParameterFormatCodes: request.Binds[b].ParameterFormatCodes,
				Parameters:           request.Binds[b].Parameters,
				ResultFormatCodes:    request.Binds[b].ResultFormatCodes,
			}
			b++
		case string('C'):
			msg = &pgproto3.Close{
				Object_Type: request.Close.Object_Type,
				Name:        request.Close.Name,
			}
		case string('D'):
			msg = &pgproto3.Describe{
				ObjectType: request.Describe.ObjectType,
				Name:       request.Describe.Name,
			}
		case string('E'):
			msg = &pgproto3.Execute{
				Portal:  request.Executes[e].Portal,
				MaxRows: request.Executes[e].MaxRows,
			}
			e++
		case string('F'):
			// *msg.(*pgproto3.Flush) = request.Flush
			msg = &pgproto3.Flush{}
		case string('f'):
			// *msg.(*pgproto3.FunctionCall) = request.FunctionCall
			msg = &pgproto3.FunctionCall{
				Function:         request.FunctionCall.Function,
				Arguments:        request.FunctionCall.Arguments,
				ArgFormatCodes:   request.FunctionCall.ArgFormatCodes,
				ResultFormatCode: request.FunctionCall.ResultFormatCode,
			}
		case string('d'):
			msg = &pgproto3.CopyData{
				Data: request.CopyData.Data,
			}
		case string('c'):
			msg = &pgproto3.CopyDone{}
		case string('H'):
			msg = &pgproto3.CopyFail{
				Message: request.CopyFail.Message,
			}
		case string('P'):
			msg = &pgproto3.Parse{
				Name:          request.Parses[p].Name,
				Query:         request.Parses[p].Query,
				ParameterOIDs: request.Parses[p].ParameterOIDs,
			}
			p++
		case string('p'):
			switch request.AuthType {
			case pgproto3.AuthTypeSASL:
				msg = &pgproto3.SASLInitialResponse{
					AuthMechanism: request.SASLInitialResponse.AuthMechanism,
					Data:          request.SASLInitialResponse.Data,
				}
			case pgproto3.AuthTypeSASLContinue:
				msg = &pgproto3.SASLResponse{
					Data: request.SASLResponse.Data,
				}
			case pgproto3.AuthTypeSASLFinal:
				msg = &pgproto3.SASLResponse{
					Data: request.SASLResponse.Data,
				}
			case pgproto3.AuthTypeGSS, pgproto3.AuthTypeGSSCont:
				msg = &pgproto3.GSSResponse{
					Data: []byte{}, // TODO: implement
				}
			case pgproto3.AuthTypeCleartextPassword, pgproto3.AuthTypeMD5Password:
				fallthrough
			default:
				// to maintain backwards compatability
				// println("Here is decoded PASSWORD", request.PasswordMessage.Password)
				msg = &pgproto3.PasswordMessage{Password: request.PasswordMessage.Password}
			}
		case string('Q'):
			msg = &pgproto3.Query{
				String: request.Query.String,
			}
		case string('S'):
			msg = &pgproto3.Sync{}
		case string('X'):
			// *msg.(*pgproto3.Terminate) = request.Terminate
			msg = &pgproto3.Terminate{}
		default:
			return nil, fmt.Errorf("unknown message type: %q", packet)
		}
		if msg == nil {
			return nil, errors.New("msg is nil")
		}
		encoded := msg.Encode([]byte{})

		reqbuffer = append(reqbuffer, encoded...)
	}
	return reqbuffer, nil
}

func PostgresEncoder(buffer []byte) string {
	// encode the buffer to base 64 string ..
	encoded := base64.StdEncoding.EncodeToString(buffer)
	return encoded
}

func findBinaryStreamMatch(tcsMocks []*models.Mock, requestBuffers [][]byte, logger *zap.Logger, h *hooks.Hook, isSorted bool) int {

	mxSim := -1.0
	mxIdx := -1

	for idx, mock := range tcsMocks {

		if len(mock.Spec.PostgresRequests) == len(requestBuffers) {
			for requestIndex, reqBuff := range requestBuffers {

				expectedPgReq := mock.Spec.PostgresRequests[requestIndex]
				encoded, err := PostgresDecoderBackend(expectedPgReq)
				if err != nil {
					logger.Debug("Error while decoding postgres request", zap.Error(err))
				}
				var encoded64 []byte
				if expectedPgReq.Payload != "" {
					encoded64, err = PostgresDecoder(mock.Spec.PostgresRequests[requestIndex].Payload)
					if err != nil {
						logger.Debug("Error while decoding postgres request", zap.Error(err))
						return -1
					}
				}
				var similarity1, similarity2 float64
				if len(encoded) > 0 {
					similarity1 = FuzzyCheck(encoded, reqBuff)
				}
				if len(encoded64) > 0 {
					similarity2 = FuzzyCheck(encoded64, reqBuff)
				}

				// calculate the jaccard similarity between the two buffers one with base64 encoding and another via that ..
				similarity := max(similarity1, similarity2)
				if mxSim < similarity {
					mxSim = similarity
					mxIdx = idx
					continue
				}
			}
		}
	}

	if isSorted {
		if mxIdx != -1 && mxSim >= 0.78 {
			logger.Debug("Matched with Sorted Stream", zap.Float64("similarity", mxSim))
		} else {
			mxIdx = -1
		}
	} else {
		if mxIdx != -1 {
			logger.Debug("Matched with Unsorted Stream", zap.Float64("similarity", mxSim))
		}
	}
	return mxIdx
}

func CheckValidEncode(tcsMocks []*models.Mock, h *hooks.Hook, log *zap.Logger) {
	for _, mock := range tcsMocks {
		for _, reqBuff := range mock.Spec.PostgresRequests {
			encode, err := PostgresDecoderBackend(reqBuff)
			if err != nil {
				log.Debug("Error in decoding")
			}
			actual_encode, err := PostgresDecoder(reqBuff.Payload)
			if err != nil {
				log.Debug("Error in decoding")
			}

			if len(encode) != len(actual_encode) {
				log.Debug("Not Equal Length of buffer in request", zap.Any("payload", reqBuff.Payload))
				log.Debug("Length of encode", zap.Int("length", len(encode)), zap.Int("length", len(actual_encode)))
				log.Debug("Encode via readable", zap.Any("encode", encode))
				log.Debug("Actual Encode", zap.Any("actual_encode", actual_encode))
				log.Debug("Payload", zap.Any("payload", reqBuff.Payload))
				continue
			}
			log.Debug("Matched")

		}
		for _, resBuff := range mock.Spec.PostgresResponses {
			encode, err := PostgresDecoderFrontend(resBuff)
			if err != nil {
				log.Debug("Error in decoding")
			}
			actual_encode, err := PostgresDecoder(resBuff.Payload)
			if err != nil {
				log.Debug("Error in decoding")
			}
			if len(encode) != len(actual_encode) {
				log.Debug("Not Equal Length of buffer in response")
				log.Debug("Length of encode", zap.Any("length", len(encode)), zap.Any("length", len(actual_encode)))
				log.Debug("Encode via readable", zap.Any("encode", encode))
				log.Debug("Actual Encode", zap.Any("actual_encode", actual_encode))
				log.Debug("Payload", zap.Any("payload", resBuff.Payload))
				continue
			}
			log.Debug("Matched")
		}
	}
	h.SetTcsMocks(tcsMocks)
}

func matchingReadablePG(requestBuffers [][]byte, logger *zap.Logger, h *hooks.Hook, ConnectionId string, recorded_prep PrepMap) (bool, []models.Frontend, error) {
	for {
		tcsMocks, err := h.GetConfigMocks()
		if err != nil {
			return false, nil, fmt.Errorf("error while getting tcs mocks %v", err)
		}

		var isMatched, sortFlag bool = false, true
		var sortedTcsMocks []*models.Mock
		var matchedMock *models.Mock

		for _, mock := range tcsMocks {
			if mock == nil {
				continue
			}

			if sortFlag {
				if !mock.TestModeInfo.IsFiltered {
					sortFlag = false
				} else {
					sortedTcsMocks = append(sortedTcsMocks, mock)
				}
			}

			if len(mock.Spec.PostgresRequests) == len(requestBuffers) {
				for requestIndex, reqBuff := range requestBuffers {
					bufStr := base64.StdEncoding.EncodeToString(reqBuff)
					encoded, err := PostgresDecoderBackend(mock.Spec.PostgresRequests[requestIndex])
					if err != nil {
						logger.Debug("Error while decoding postgres request", zap.Error(err))
					}
					if mock.Spec.PostgresRequests[requestIndex].Identfier == "StartupRequest" {
						logger.Debug("CHANGING TO MD5 for Response")
						mock.Spec.PostgresResponses[requestIndex].AuthType = 5
						continue
					} else {
						if len(encoded) > 0 && encoded[0] == 'p' {
							logger.Debug("CHANGING TO MD5 for Request and Response")
							mock.Spec.PostgresRequests[requestIndex].PasswordMessage.Password = "md5fe4f2f657f01fa1dd9d111d5391e7c07"

							mock.Spec.PostgresResponses[requestIndex].PacketTypes = []string{"R", "S", "S", "S", "S", "S", "S", "S", "S", "S", "S", "S", "K", "Z"}
							mock.Spec.PostgresResponses[requestIndex].AuthType = 0
							mock.Spec.PostgresResponses[requestIndex].BackendKeyData = pgproto3.BackendKeyData{
								ProcessID: 2613,
								SecretKey: 824670820,
							}
							mock.Spec.PostgresResponses[requestIndex].ReadyForQuery.TxStatus = 73
							mock.Spec.PostgresResponses[requestIndex].ParameterStatusCombined = []pgproto3.ParameterStatus{
								{
									Name:  "application_name",
									Value: "",
								},
								{
									Name:  "client_encoding",
									Value: "UTF8",
								},
								{
									Name:  "DateStyle",
									Value: "ISO, MDY",
								},
								{
									Name:  "integer_datetimes",
									Value: "on",
								},
								{
									Name:  "IntervalStyle",
									Value: "postgres",
								},
								{
									Name:  "is_superuser",
									Value: "UTF8",
								},
								{
									Name:  "server_version",
									Value: "13.12 (Debian 13.12-1.pgdg120+1)",
								},
								{
									Name:  "session_authorization",
									Value: "keploy-user",
								},
								{
									Name:  "standard_conforming_strings",
									Value: "on",
								},
								{
									Name:  "TimeZone",
									Value: "Etc/UTC",
								},
								{
									Name:  "TimeZone",
									Value: "Etc/UTC",
								},
							}
						}
					}

					if bufStr == "AAAACATSFi8=" {
						ssl := models.Frontend{
							Payload: "Tg==",
						}
						return true, []models.Frontend{ssl}, nil
					}
				}
			}

			// maintain test prepare statement map for each connection id
			getTestPS(requestBuffers, logger, ConnectionId)
			// match, _ := compareExactMatch(mock, requestBuffers, logger, h, ConnectionId)
			// // if err != nil {
			// // 	return false, nil, err
			// // }
			// if match {
			// 	fmt.Println("Matched In Absolute Custom Matching for ", mock.Name)
			// 	isMatched = true
			// 	matchedMock = mock
			// 	break
			// }
		}

		logger.Debug("Sorted Mocks: ", zap.Any("Len of sortedTcsMocks", len(sortedTcsMocks)))

		isSorted := false
		var idx int
		if !isMatched {
			//use findBinaryMatch twice one for sorted and another for unsorted
			// give more priority to sorted like if you find more than 0.5 in sorted then return that
			if len(sortedTcsMocks) > 0 {
				isSorted = true
				idx1, _ := findPGStreamMatch(sortedTcsMocks, requestBuffers, logger, h, isSorted, ConnectionId, recorded_prep)
				if idx1 != -1 {
					isMatched = true
					matchedMock = tcsMocks[idx1]
					fmt.Println("Matched In Absolute Custom Matching for sorted!!!", matchedMock.Name)
				}
				idx = findBinaryStreamMatch(sortedTcsMocks, requestBuffers, logger, h, isSorted)
				if idx != -1 && !isMatched {
					isMatched = true
					matchedMock = tcsMocks[idx]
					fmt.Println("Matched In Binary Matching for sorted!!!", matchedMock.Name)
				}
			}
		}

		if !isMatched {
			isSorted = false
			idx1, _ := findPGStreamMatch(tcsMocks, requestBuffers, logger, h, isSorted, ConnectionId, recorded_prep)
			if idx1 != -1 {
				isMatched = true
				matchedMock = tcsMocks[idx1]
				fmt.Println("Matched In Absolute Custom Matching for Unsorted", matchedMock.Name)
			}
			idx = findBinaryStreamMatch(tcsMocks, requestBuffers, logger, h, isSorted)
			if idx != -1 && !isMatched {
				isMatched = true
				matchedMock = tcsMocks[idx]
				fmt.Println("Matched In Binary Matching for Unsorted", matchedMock.Name)
			}
		}

		if isMatched {
			logger.Info("Matched mock", zap.String("mock", matchedMock.Name))
			if matchedMock.TestModeInfo.IsFiltered {
				originalMatchedMock := *matchedMock
				matchedMock.TestModeInfo.IsFiltered = false
				matchedMock.TestModeInfo.SortOrder = math.MaxInt
				isUpdated := h.UpdateConfigMock(&originalMatchedMock, matchedMock)
				if !isUpdated {
					continue
				}
			}
			return true, matchedMock.Spec.PostgresResponses, nil
		}

		break
	}
	return false, nil, nil
}

func decodePgRequest(buffer []byte, logger *zap.Logger) *models.Backend {

	pg := NewBackend()

	if !isStartupPacket(buffer) && len(buffer) > 5 {
		bufferCopy := buffer
		for i := 0; i < len(bufferCopy)-5; {
			logger.Debug("Inside the if condition")
			pg.BackendWrapper.MsgType = buffer[i]
			pg.BackendWrapper.BodyLen = int(binary.BigEndian.Uint32(buffer[i+1:])) - 4
			if len(buffer) < (i + pg.BackendWrapper.BodyLen + 5) {
				logger.Debug("failed to translate the postgres request message due to shorter network packet buffer")
				break
			}
			msg, err := pg.TranslateToReadableBackend(buffer[i:(i + pg.BackendWrapper.BodyLen + 5)])
			if err != nil && buffer[i] != 112 {
				logger.Debug("failed to translate the request message to readable", zap.Error(err))
			}
			if pg.BackendWrapper.MsgType == 'p' {
				pg.BackendWrapper.PasswordMessage = *msg.(*pgproto3.PasswordMessage)
			}

			if pg.BackendWrapper.MsgType == 'P' {
				pg.BackendWrapper.Parse = *msg.(*pgproto3.Parse)
				pg.BackendWrapper.Parses = append(pg.BackendWrapper.Parses, pg.BackendWrapper.Parse)
			}

			if pg.BackendWrapper.MsgType == 'B' {
				pg.BackendWrapper.Bind = *msg.(*pgproto3.Bind)
				pg.BackendWrapper.Binds = append(pg.BackendWrapper.Binds, pg.BackendWrapper.Bind)
			}

			if pg.BackendWrapper.MsgType == 'E' {
				pg.BackendWrapper.Execute = *msg.(*pgproto3.Execute)
				pg.BackendWrapper.Executes = append(pg.BackendWrapper.Executes, pg.BackendWrapper.Execute)
			}

			pg.BackendWrapper.PacketTypes = append(pg.BackendWrapper.PacketTypes, string(pg.BackendWrapper.MsgType))

			i += (5 + pg.BackendWrapper.BodyLen)
		}

		pg_mock := &models.Backend{
			PacketTypes: pg.BackendWrapper.PacketTypes,
			Identfier:   "ClientRequest",
			Length:      uint32(len(buffer)),
			// Payload:             bufStr,
			Bind:                pg.BackendWrapper.Bind,
			Binds:               pg.BackendWrapper.Binds,
			PasswordMessage:     pg.BackendWrapper.PasswordMessage,
			CancelRequest:       pg.BackendWrapper.CancelRequest,
			Close:               pg.BackendWrapper.Close,
			CopyData:            pg.BackendWrapper.CopyData,
			CopyDone:            pg.BackendWrapper.CopyDone,
			CopyFail:            pg.BackendWrapper.CopyFail,
			Describe:            pg.BackendWrapper.Describe,
			Execute:             pg.BackendWrapper.Execute,
			Executes:            pg.BackendWrapper.Executes,
			Flush:               pg.BackendWrapper.Flush,
			FunctionCall:        pg.BackendWrapper.FunctionCall,
			GssEncRequest:       pg.BackendWrapper.GssEncRequest,
			Parse:               pg.BackendWrapper.Parse,
			Parses:              pg.BackendWrapper.Parses,
			Query:               pg.BackendWrapper.Query,
			SSlRequest:          pg.BackendWrapper.SSlRequest,
			StartupMessage:      pg.BackendWrapper.StartupMessage,
			SASLInitialResponse: pg.BackendWrapper.SASLInitialResponse,
			SASLResponse:        pg.BackendWrapper.SASLResponse,
			Sync:                pg.BackendWrapper.Sync,
			Terminate:           pg.BackendWrapper.Terminate,
			MsgType:             pg.BackendWrapper.MsgType,
			AuthType:            pg.BackendWrapper.AuthType,
		}
		return pg_mock
	}

	return nil
}

func FuzzyCheck(encoded, reqBuff []byte) float64 {
	k := util.AdaptiveK(len(reqBuff), 3, 8, 5)
	shingles1 := util.CreateShingles(encoded, k)
	shingles2 := util.CreateShingles(reqBuff, k)
	similarity := util.JaccardSimilarity(shingles1, shingles2)
	return similarity
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func PreparedStatementMatch(mock *models.Mock, actualPgReq *models.Backend, logger *zap.Logger, h *hooks.Hook, ConnectionId string, recorded_prep PrepMap) (bool, []string, error) {
	// fmt.Println("Inside PreparedStatementMatch")
	// check the current Query associated with the connection id and Identifier
	ifps := checkIfps(actualPgReq.PacketTypes)
	if !ifps {
		return false, nil, nil
	}
	// get all the binds from the actualPgReq
	binds := actualPgReq.Binds
	newBinPreparedStatement := make([]string, 0)
	for _, bind := range binds {
		current_ps := bind.PreparedStatement
		current_querydata := testmap[ConnectionId]
		current_query := ""
		// check in the map that what's the current query for this preparedstatement
		// then will check what is the recorded prepared statement for this query
		for _, v := range current_querydata {
			if v.PrepIdentifier == current_ps {
				// fmt.Println("Current query for this identifier is ", v.Query)
				current_query = v.Query
				break
			}
		}
		// check what was the prepared statement recorded
		// old_ps := ""
		for _, ps := range recorded_prep {
			for _, v := range ps {
				if current_query == v.Query && current_ps != v.PrepIdentifier {
					// fmt.Println("Matched with the recorded prepared statement with Identifier and connectionID is", v.PrepIdentifier, ", conn- ", conn, "and current identifier is", current_ps, "FOR QUERY", current_query)
					// fmt.Println("MOCK NUMBER IS ", mock.Name)
					current_ps = v.PrepIdentifier
					break
				}
			}
		}

		if strings.Contains(current_ps, "S_") && current_ps != "" {
			newBinPreparedStatement = append(newBinPreparedStatement, current_ps)
		}
	}
	if len(newBinPreparedStatement) > 0 && len(binds) == len(newBinPreparedStatement) {
		return true, newBinPreparedStatement, nil
	}
	return false, nil, nil
}

func compareExactMatch(mock *models.Mock, reqBuff []byte, logger *zap.Logger, h *hooks.Hook, ConnectionId string, isSorted bool, recorded_prep PrepMap) (bool, error) {

	// fmt.Println("Inside Compare Exact Match", len(reqBuff), string(reqBuff), "MOCK NAME - ", mock.Name)
	actualPgReq := decodePgRequest(reqBuff, logger)
	if actualPgReq == nil {
		return false, nil
	}
	if mock.Name == "mock-196" || mock.Name == "mock-197" || mock.Name == "mock-198" || mock.Name == "mock-199" || mock.Name == "mock-204" {
		fmt.Println("Inside Compare Exact Match", actualPgReq, "MOCK NAME - ", mock.Name, "WITH CONN ID", ConnectionId)
	}

	// have to ignore first parse message of begin read only
	// should compare only query in the parse message
	if len(actualPgReq.PacketTypes) != len(mock.Spec.PostgresRequests[0].PacketTypes) {
		//check for begin read only
		if math.Abs(float64(len(actualPgReq.PacketTypes))-float64(len(mock.Spec.PostgresRequests[0].PacketTypes))) == 1 {
			if len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {
				if mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && mock.Spec.PostgresRequests[0].Parses[0].Query == "BEGIN READ ONLY" {
					if mock.Spec.PostgresRequests[0].Parses[1].Query == actualPgReq.Parses[0].Query {
						sliceCommandTag(mock, logger, testmap[ConnectionId], actualPgReq, false)
						return true, nil
					}
				}
			}
		}

		if isSorted && len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {
			if mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && actualPgReq.PacketTypes[0] == "B" && mock.Name == "mock-197" {
				// is_prep := checkIfps(actualPgReq.PacketTypes)
				// if is_prep {
				// 	// fmt.Println("Inside Prepared Statement")
				// 	// sliceCommandTag(mock, logger, testmap[ConnectionId], actualPgReq, true)
				// }

				return true, nil
			}
		}
		return false, nil
	}

	// call a separate function for matching prepared statements
	for idx, v := range actualPgReq.PacketTypes {
		if v != mock.Spec.PostgresRequests[0].PacketTypes[idx] {
			return false, nil
		}
	}
	// IsPreparedStatement(mock, actualPgReq, logger, ConnectionId)
	is_prep, newBindPs, err := PreparedStatementMatch(mock, actualPgReq, logger, h, ConnectionId, recorded_prep)
	if err != nil {
		logger.Error("Error while matching prepared statements", zap.Error(err))
	}
	// this will give me the
	var (
		p, b, e int = 0, 0, 0
	)
	for i := 0; i < len(actualPgReq.PacketTypes); i++ {
		switch actualPgReq.PacketTypes[i] {
		case "P":
			// fmt.Println("Inside P")
			p++
			if actualPgReq.Parses[p-1].Name != mock.Spec.PostgresRequests[0].Parses[p-1].Name {
				return false, nil
			}
			if actualPgReq.Parses[p-1].Query != mock.Spec.PostgresRequests[0].Parses[p-1].Query {
				return false, nil
			}
			if len(actualPgReq.Parses[p-1].ParameterOIDs) != len(mock.Spec.PostgresRequests[0].Parses[p-1].ParameterOIDs) {
				return false, nil
			}
			for j := 0; j < len(actualPgReq.Parses[p-1].ParameterOIDs); j++ {
				if actualPgReq.Parses[p-1].ParameterOIDs[j] != mock.Spec.PostgresRequests[0].Parses[p-1].ParameterOIDs[j] {
					return false, nil
				}
			}

		case "B":
			// fmt.Println("Inside B")
			b++
			if actualPgReq.Binds[b-1].DestinationPortal != mock.Spec.PostgresRequests[0].Binds[b-1].DestinationPortal {
				return false, nil
			}
			// if Is Prep statement true hai to wo se jo aya hai usko S_ identifier ko and connection Id ko compare karo
			if is_prep && len(newBindPs) > 0 {
				fmt.Println("New Bind Prepared Statement", newBindPs, "for mock", mock.Name)
				if mock.Spec.PostgresRequests[0].Binds[b-1].PreparedStatement != newBindPs[b-1] {
					return false, nil
				}
			} else {
				if actualPgReq.Binds[b-1].PreparedStatement != mock.Spec.PostgresRequests[0].Binds[b-1].PreparedStatement {
					return false, nil
				}
			}

			if len(actualPgReq.Binds[b-1].ParameterFormatCodes) != len(mock.Spec.PostgresRequests[0].Binds[b-1].ParameterFormatCodes) {
				return false, nil
			}
			for j := 0; j < len(actualPgReq.Binds[b-1].ParameterFormatCodes); j++ {
				if actualPgReq.Binds[b-1].ParameterFormatCodes[j] != mock.Spec.PostgresRequests[0].Binds[b-1].ParameterFormatCodes[j] {
					return false, nil
				}
			}
			if len(actualPgReq.Binds[b-1].Parameters) != len(mock.Spec.PostgresRequests[0].Binds[b-1].Parameters) {
				return false, nil
			}
			for j := 0; j < len(actualPgReq.Binds[b-1].Parameters); j++ {
				for _, v := range actualPgReq.Binds[b-1].Parameters[j] {
					if v != mock.Spec.PostgresRequests[0].Binds[b-1].Parameters[j][0] {
						return false, nil
					}
				}
			}
			if len(actualPgReq.Binds[b-1].ResultFormatCodes) != len(mock.Spec.PostgresRequests[0].Binds[b-1].ResultFormatCodes) {
				return false, nil
			}
			for j := 0; j < len(actualPgReq.Binds[b-1].ResultFormatCodes); j++ {
				if actualPgReq.Binds[b-1].ResultFormatCodes[j] != mock.Spec.PostgresRequests[0].Binds[b-1].ResultFormatCodes[j] {
					return false, nil
				}
			}

		case "E":
			// fmt.Println("Inside E")
			e++
			if actualPgReq.Executes[e-1].Portal != mock.Spec.PostgresRequests[0].Executes[e-1].Portal {
				return false, nil
			}
			if actualPgReq.Executes[e-1].MaxRows != mock.Spec.PostgresRequests[0].Executes[e-1].MaxRows {
				return false, nil
			}
		// case "d":
		// 	if actualPgReq.CopyData.Data != mock.Spec.PostgresRequests[0].CopyData.Data {
		// 		return false, nil
		// 	}
		case "c":
			if actualPgReq.CopyDone != mock.Spec.PostgresRequests[0].CopyDone {
				return false, nil
			}
		case "H":
			if actualPgReq.CopyFail.Message != mock.Spec.PostgresRequests[0].CopyFail.Message {
				return false, nil
			}
		default:
			return false, nil
		}
	}
	return true, nil
}

var testmap TestPrepMap

func getTestPS(reqBuff [][]byte, logger *zap.Logger, ConnectionId string) {
	// maintain a map of current prepared statements and their corresponding connection id
	// if it's the prepared statement match the query with the recorded prepared statement and return the response of that matched prepared statement at that connection
	// so if parse is coming save to a same map
	actualPgReq := decodePgRequest(reqBuff[0], logger)
	if actualPgReq == nil {
		return
	}
	testmap2 := make(TestPrepMap)
	if testmap != nil {
		testmap2 = testmap
	}
	querydata := make([]QueryData, 0)
	if len(actualPgReq.PacketTypes) > 0 && actualPgReq.PacketTypes[0] != "p" && actualPgReq.Identfier != "StartupRequest" {
		p := 0
		for _, header := range actualPgReq.PacketTypes {
			if header == "P" {
				if strings.Contains(actualPgReq.Parses[p].Name, "S_") && !IsValuePresent(ConnectionId, actualPgReq.Parses[p].Name) {
					querydata = append(querydata, QueryData{PrepIdentifier: actualPgReq.Parses[p].Name, Query: actualPgReq.Parses[p].Query})
				}
				p++
			}
		}
	}
	// also append the query data for the prepared statement
	if len(querydata) > 0 {
		testmap2[ConnectionId] = append(testmap2[ConnectionId], querydata...)
		fmt.Println("Test Prepared statement Map", testmap2)
		testmap = testmap2
	}

}

func IsValuePresent(connectionid string, value string) bool {
	if testmap != nil {
		for _, v := range testmap[connectionid] {
			if v.PrepIdentifier == value {
				return true
			}
		}
	}
	return false
}

func findPGStreamMatch(tcsMocks []*models.Mock, requestBuffers [][]byte, logger *zap.Logger, h *hooks.Hook, isSorted bool, connectionId string, recorded_prep PrepMap) (int, map[string]string) {

	mxIdx := -1
	// matchedMockConnectionId := "y"
	var newPrepareStatementMapAfterMatching map[string]string
	// var isPrepareStatementMapMatched bool

	for idx, mock := range tcsMocks {
		if len(mock.Spec.PostgresRequests) == len(requestBuffers) {
			for _, reqBuff := range requestBuffers {
				// here handle cases of prepared statement very carefully
				match, err := compareExactMatch(mock, reqBuff, logger, h, connectionId, isSorted, recorded_prep)
				if err != nil || match == false {
					break
				}
				if match {
					return idx, newPrepareStatementMapAfterMatching
				}
			}
		}
	}
	return mxIdx, newPrepareStatementMapAfterMatching
}

func checkIfps(array []string) bool {
	n := len(array)
	if n%2 != 0 {
		// If the array length is odd, it cannot match the pattern
		return false
	}

	for i := 0; i < n; i += 2 {
		// Check if consecutive elements are "B" and "E"
		if array[i] != "B" || array[i+1] != "E" {
			return false
		}
	}

	return true
}

func sliceCommandTag(mock *models.Mock, logger *zap.Logger, prep []QueryData, actualPgReq *models.Backend, isDatareq bool) {

	mockBuffer := mock.Spec.PostgresResponses[0].Payload
	buffer, _ := PostgresDecoder(mockBuffer)
	//check if the bind is prepared statement for begin read only
	// Identfier := actualPgReq.Binds[0].PreparedStatement
	// foo := false
	// for _, v := range prep {
	// 	if (v.PrepIdentifier == Identfier) && (v.Query == "BEGIN READ ONLY") {
	// 		foo = true
	// 		break
	// 	}
	// }

	// if !foo {
	// 	return
	// }

	fmt.Println("Inside Slice Command Tag")
	expectedHeader := []string{"1"}
	// if isDatareq {
	// 	expectedHeader = []string{"1", "T"}
	// }

	var i int = 0
	for exp := 0; exp < len(expectedHeader); exp++ {
		if string(buffer[i]) != expectedHeader[exp] {
			logger.Error("Incorrect mock response provided for slicing")
			return
		}
		BodyLen := int(binary.BigEndian.Uint32(buffer[i+1:])) - 4
		i += (5 + BodyLen)
	}

	buffer = buffer[i:]
	payload := PostgresEncoder(buffer)
	mock.Spec.PostgresResponses[0].Payload = payload
}

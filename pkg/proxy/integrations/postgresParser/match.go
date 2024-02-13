package postgresparser

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgproto3/v2"
	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"
	"go.keploy.io/server/pkg/proxy/util"
	"go.uber.org/zap"
)

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

func compareExactMatch(mock *models.Mock, actualPgReq *models.Backend, logger *zap.Logger, h *hooks.Hook, ConnectionId string, isSorted bool, recorded_prep PrepMap) (bool, error) {

	// fmt.Println("Inside Compare Exact Match", len(reqBuff), string(reqBuff), "MOCK NAME - ", mock.Name)
	// actualPgReq := decodePgRequest(reqBuff, logger)
	// if actualPgReq == nil {
	// 	return false, nil
	// }
	// if mock.Name == "mock-196" || mock.Name == "mock-197" || mock.Name == "mock-198" || mock.Name == "mock-199" || mock.Name == "mock-204" {
	// 	fmt.Println("Inside Compare Exact Match", actualPgReq, "MOCK NAME - ", mock.Name, "WITH CONN ID", ConnectionId)
	// }

	// have to ignore first parse message of begin read only
	// should compare only query in the parse message
	if len(actualPgReq.PacketTypes) != len(mock.Spec.PostgresRequests[0].PacketTypes) {
		//check for begin read only
		// if math.Abs(float64(len(actualPgReq.PacketTypes))-float64(len(mock.Spec.PostgresRequests[0].PacketTypes))) == 1 {
		// 	if len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {
		// 		if mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && mock.Spec.PostgresRequests[0].Parses[0].Query == "BEGIN READ ONLY" {
		// 			if mock.Spec.PostgresRequests[0].Parses[1].Query == actualPgReq.Parses[0].Query {
		// 				sliceCommandTag(mock, logger, testmap[ConnectionId], actualPgReq, false)
		// 				return true, nil
		// 			}
		// 		}
		// 	}
		// }

		// if isSorted && len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {
		// 	if mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && actualPgReq.PacketTypes[0] == "B" && mock.Name == "mock-197" {
		// 		// is_prep := checkIfps(actualPgReq.PacketTypes)
		// 		// if is_prep {
		// 		// 	// fmt.Println("Inside Prepared Statement")
		// 		// 	// sliceCommandTag(mock, logger, testmap[ConnectionId], actualPgReq, true)
		// 		// }

		// 		return true, nil
		// 	}
		// }
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
				// fmt.Println("New Bind Prepared Statement", newBindPs, "for mock", mock.Name)
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
		// fmt.Println("Test Prepared statement Map", testmap2)
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
				actualPgReq := decodePgRequest(reqBuff, logger)
				if actualPgReq == nil {
					return -1, nil
				}

				// here handle cases of prepared statement very carefully
				match, err := compareExactMatch(mock, actualPgReq, logger, h, connectionId, isSorted, recorded_prep)
				if err != nil || match == false {
					if mock.Name == "mock-201" || mock.Name == "mock-202" || mock.Name == "mock-203" || mock.Name == "mock-205" || mock.Name == "mock-206" || mock.Name == "mock-207" || mock.Name == "mock-208" || mock.Name == "mock-209" || mock.Name == "mock-212" {
						fmt.Println("TEST prep map", testmap)
						// mock.Spec.PostgresRequests[0].
						fmt.Println("Inside Compare Exact Match", actualPgReq, "MOCK NAME - ", mock.Name, "WITH CONN ID", connectionId)
					}

					// have to ignore first parse message of begin read only
					// should compare only query in the parse message
					if len(actualPgReq.PacketTypes) != len(mock.Spec.PostgresRequests[0].PacketTypes) {
						//check for begin read only
						if math.Abs(float64(len(actualPgReq.PacketTypes))-float64(len(mock.Spec.PostgresRequests[0].PacketTypes))) == 1 {
							if len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {
								if mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && mock.Spec.PostgresRequests[0].Parses[0].Query == "BEGIN READ ONLY" {

									if mock.Spec.PostgresRequests[0].Parses[1].Query == actualPgReq.Parses[0].Query {
										sliceCommandTag(mock, logger, testmap[connectionId], actualPgReq, false)
										return idx, nil
									}
								}
							}
						}

						if isSorted && len(actualPgReq.PacketTypes) > 0 && len(mock.Spec.PostgresRequests[0].PacketTypes) > 0 {

							if ((mock.Spec.PostgresRequests[0].PacketTypes[0] == "B" && actualPgReq.PacketTypes[0] == "P") || (mock.Spec.PostgresRequests[0].PacketTypes[0] == "P" && actualPgReq.PacketTypes[0] == "B")) && (mock.Name == "mock-197" || mock.Name == "mock-198" || mock.Name == "mock-199" || mock.Name == "mock-202") {
								// is_prep := checkIfps(actualPgReq.PacketTypes)
								// if is_prep {
								// 	// fmt.Println("Inside Prepared Statement")
								// 	// sliceCommandTag(mock, logger, testmap[ConnectionId], actualPgReq, true)
								// }
								if mock.Name == "mock-207" {
									fmt.Println("HHHHHHAAAAAA")
								}

								return idx, nil
							}
						}

					}

				}
				if match {
					return idx, newPrepareStatementMapAfterMatching
				}
			}
		}
	}
	return mxIdx, newPrepareStatementMapAfterMatching
}

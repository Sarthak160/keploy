package httpparser

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/fatih/color"
	"go.keploy.io/server/pkg"
	"go.keploy.io/server/pkg/hooks"
	"go.keploy.io/server/pkg/models"
	"go.keploy.io/server/pkg/proxy/util"
	"go.uber.org/zap"
)

type HttpParser struct {
	logger  *zap.Logger
	hooks   *hooks.Hook
	baseUrl string
}

// ProcessOutgoing implements proxy.DepInterface.
func (http *HttpParser) ProcessOutgoing(request []byte, clientConn, destConn net.Conn, ctx context.Context) {
	switch models.GetMode() {
	case models.MODE_RECORD:
		err := encodeOutgoingHttp(request, clientConn, destConn, http.logger, http.hooks, ctx)
		if err != nil {
			http.logger.Error("failed to encode the http message into the yaml", zap.Error(err))
			return
		}

	case models.MODE_TEST:
		decodeOutgoingHttp(request, clientConn, destConn, http.hooks, http.logger, http.baseUrl)
	default:
		http.logger.Info("Invalid mode detected while intercepting outgoing http call", zap.Any("mode", models.GetMode()))
	}

}

func NewHttpParser(logger *zap.Logger, h *hooks.Hook, baseUrl string) *HttpParser {
	return &HttpParser{
		logger:  logger,
		hooks:   h,
		baseUrl: baseUrl,
	}
}

// IsOutgoingHTTP function determines if the outgoing network call is HTTP by comparing the
// message format with that of an HTTP text message.
func (h *HttpParser) OutgoingType(buffer []byte) bool {
	return bytes.HasPrefix(buffer[:], []byte("HTTP/")) ||
		bytes.HasPrefix(buffer[:], []byte("GET ")) ||
		bytes.HasPrefix(buffer[:], []byte("POST ")) ||
		bytes.HasPrefix(buffer[:], []byte("PUT ")) ||
		bytes.HasPrefix(buffer[:], []byte("PATCH ")) ||
		bytes.HasPrefix(buffer[:], []byte("DELETE ")) ||
		bytes.HasPrefix(buffer[:], []byte("OPTIONS ")) ||
		bytes.HasPrefix(buffer[:], []byte("HEAD "))
}

func isJSON(body []byte) bool {
	var js interface{}
	return json.Unmarshal(body, &js) == nil
}

func mapsHaveSameKeys(map1 map[string]string, map2 map[string][]string) bool {
	if len(map1) != len(map2) {
		return false
	}
	var rv, rv2 string
	for key, v := range map1 {
		// if _, exists := map2[key]; !exists {
		// 	return false
		// }
		if key == "Keploy-Header" {
			rv = v
		}
	}

	for key, v := range map2 {
		// if _, exists := map1[key]; !exists {
		// 	return false
		// }
		if key == "Keploy-Header" {
			rv2 = v[0]
		}
	}
	if rv != rv2 {
		return mapsHaveSameKeys2(map1, map2)
	}
	return true
}
func mapsHaveSameKeys2(map1 map[string]string, map2 map[string][]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key := range map1 {
		if _, exists := map2[key]; !exists {
			return false
		}
	}

	for key := range map2 {
		if _, exists := map1[key]; !exists {
			return false
		}
	}

	return true
}

// Handled chunked requests when content-length is given.
func contentLengthRequest(finalReq *[]byte, clientConn, destConn net.Conn, logger *zap.Logger, contentLength int) {
	for contentLength > 0 {
		clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		requestChunked, err := util.ReadBytes(clientConn)
		if err != nil {
			if err == io.EOF {
				logger.Error("connection closed by the user client", zap.Error(err))
				break
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("Stopped getting data from the connection", zap.Error(err))
				break
			} else {
				logger.Error("failed to read the response message from the destination server", zap.Error(err))
				return
			}
		}
		logger.Debug("This is a chunk of request[content-length]: " + string(requestChunked))
		*finalReq = append(*finalReq, requestChunked...)
		contentLength -= len(requestChunked)

		// fmt.Println("requestChunked::::", requestChunked, "len::::", len(requestChunked))
		//Not to be used in test mode
		_, err = destConn.Write(requestChunked)
		if err != nil {
			logger.Error("failed to write request message to the destination server", zap.Error(err))
			return
		}
	}
	clientConn.SetReadDeadline(time.Time{})
}

// Handled chunked requests when transfer-encoding is given.
func chunkedRequest(finalReq *[]byte, clientConn, destConn net.Conn, logger *zap.Logger, transferEncodingHeader string) {
	if transferEncodingHeader == "chunked" {
		for {
			clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			requestChunked, err := util.ReadBytes(clientConn)
			if err != nil && err != io.EOF {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					break
				} else {
					logger.Error("failed to read the response message from the destination server", zap.Error(err))
					return
				}
			}
			*finalReq = append(*finalReq, requestChunked...)
			_, err = destConn.Write(requestChunked)
			if err != nil {
				logger.Error("failed to write request message to the destination server", zap.Error(err))
				return
			}

			//check if the intial request is completed
			if strings.HasSuffix(string(requestChunked), "0\r\n\r\n") {
				break
			}
		}
		clientConn.SetReadDeadline(time.Time{})
	}
}

// Handled chunked responses when content-length is given.
func contentLengthResponse(finalResp *[]byte, clientConn, destConn net.Conn, logger *zap.Logger, contentLength int, Overallcounter *int64, PerConncounter int) {
	for contentLength > 0 {
		//Set deadline of 5 seconds
		destConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		resp, err := util.ReadBytes(destConn)
		if err != nil {
			//Check if the connection closed.
			if err == io.EOF {
				logger.Error(fmt.Sprint(PerConncounter)+"connection closed by the destination server", zap.Error(err))
				break
			} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info(fmt.Sprint(PerConncounter)+"Stopped getting data from the connection", zap.Error(err))
				break
			} else {
				logger.Error(fmt.Sprint(PerConncounter)+"failed to read the response message from the destination server", zap.Error(err))
				return
			}
		}

		*finalResp = append(*finalResp, resp...)
		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(Overallcounter)) + "This is a chunk of response[content-length]: " + string(resp))
		contentLength -= len(resp)
		// write the response message to the user client
		_, err = clientConn.Write(resp)
		if err != nil {
			logger.Error("failed to write response message to the user client", zap.Error(err))
			return
		}
	}
	destConn.SetReadDeadline(time.Time{})

}

// Handled chunked responses when transfer-encoding is given.
func chunkedResponse(chunkedTime *[]int64, chunkedLength *[]int, finalResp *[]byte, clientConn, destConn net.Conn, logger *zap.Logger, transferEncodingHeader string, Overallcounter *int64, PerConncounter int) {
	if transferEncodingHeader == "chunked" {
		for {
			resp, err := util.ReadBytes(destConn)
			if err != nil {
				if err != io.EOF {
					logger.Debug("fmt.Sprint(PerConncounter) "+fmt.Sprint(PerConncounter)+" OverallCounter "+fmt.Sprint(atomic.LoadInt64(Overallcounter))+"failed to read the response message from the destination server", zap.Error(err))
					return
				} else {
					logger.Debug("fmt.Sprint(PerConncounter) "+fmt.Sprint(PerConncounter)+" OverallCounter "+fmt.Sprint(atomic.LoadInt64(Overallcounter))+"recieved EOF, exiting loop as response is complete", zap.Error(err))
					break
				}
			}

			// get all the chunks mapped with that time at least get the number of chunks at that time 
			if len(resp) != 0 {
				fmt.Println("resp::::", string(resp))
				t := time.Now().UnixMilli()
				fmt.Println("THIS IS TIME ---", t)
				//Get the length of the chunk <- check this

				count, err := countHTTPChunks(resp)
				if err != nil {
					fmt.Println("Error extracting length:", err)
				}
				fmt.Println("Count ", count)
				*chunkedTime = append(*chunkedTime, t)
				*chunkedLength = append(*chunkedLength, count)
			}

			//get the hexa decimal and then convert it to length
			*finalResp = append(*finalResp, resp...)
			logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(Overallcounter)) + "This is a chunk of response[chunked]: " + string(resp))
			// write the response message to the user client

			_, err = clientConn.Write(resp)
			if err != nil {
				logger.Error(fmt.Sprint(PerConncounter)+"failed to write response message to the user client", zap.Error(err))
				return
			}
			if string(resp) == "0\r\n\r\n" {
				break
			}
		}
	}
}

func handleChunkedRequests(finalReq *[]byte, clientConn, destConn net.Conn, logger *zap.Logger) {
	lines := strings.Split(string(*finalReq), "\n")
	var contentLengthHeader string
	var transferEncodingHeader string
	for _, line := range lines {
		if strings.HasPrefix(line, "Content-Length:") {
			contentLengthHeader = strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			break
		} else if strings.HasPrefix(line, "Transfer-Encoding:") {
			transferEncodingHeader = strings.TrimSpace(strings.TrimPrefix(line, "Transfer-Encoding:"))
			break
		}
	}
	//Handle chunked requests
	if contentLengthHeader != "" {
		contentLength, err := strconv.Atoi(contentLengthHeader)
		if err != nil {
			logger.Error("failed to get the content-length header", zap.Error(err))
			return
		}
		//Get the length of the body in the request.
		bodyLength := len(*finalReq) - strings.Index(string(*finalReq), "\r\n\r\n") - 4
		contentLength -= bodyLength
		if contentLength > 0 {
			contentLengthRequest(finalReq, clientConn, destConn, logger, contentLength)
		}
	} else if transferEncodingHeader != "" {
		// check if the intial request is the complete request.
		if strings.HasSuffix(string(*finalReq), "0\r\n\r\n") {
			return
		}
		chunkedRequest(finalReq, clientConn, destConn, logger, transferEncodingHeader)
	}
}

func handleChunkedResponses(chunkedTime *[]int64, chunkedLength *[]int, finalResp *[]byte, clientConn, destConn net.Conn, logger *zap.Logger, resp []byte, Overallcounter *int64, PerConncounter int) {
	//Getting the content-length or the transfer-encoding header
	var contentLengthHeader, transferEncodingHeader string
	lines := strings.Split(string(resp), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Content-Length:") {
			contentLengthHeader = strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			break
		} else if strings.HasPrefix(line, "Transfer-Encoding:") {
			transferEncodingHeader = strings.TrimSpace(strings.TrimPrefix(line, "Transfer-Encoding:"))
			break
		}
	}

	//Handle chunked responses
	if contentLengthHeader != "" {
		contentLength, err := strconv.Atoi(contentLengthHeader)
		if err != nil {
			logger.Error("failed to get the content-length header", zap.Error(err))
			return
		}
		bodyLength := len(resp) - strings.Index(string(resp), "\r\n\r\n") - 4
		contentLength -= bodyLength
		if contentLength > 0 {
			contentLengthResponse(finalResp, clientConn, destConn, logger, contentLength, Overallcounter, PerConncounter)
		}
	} else if transferEncodingHeader != "" {
		//check if the intial response is the complete response.
		if strings.HasSuffix(string(*finalResp), "0\r\n\r\n") {
			return
		}

		chunkedResponse(chunkedTime, chunkedLength, finalResp, clientConn, destConn, logger, transferEncodingHeader, Overallcounter, PerConncounter)
	}
}

// Checks if the response is gzipped
func checkIfGzipped(check io.ReadCloser) (bool, *bufio.Reader) {
	bufReader := bufio.NewReader(check)
	peekedBytes, err := bufReader.Peek(2)
	if err != nil && err != io.EOF {
		log.Debug("Error peeking:", err)
		return false, nil
	}
	if len(peekedBytes) < 2 {
		return false, nil
	}
	if peekedBytes[0] == 0x1f && peekedBytes[1] == 0x8b {
		return true, bufReader
	} else {
		return false, nil
	}
}

// Decodes the mocks in test mode so that they can be sent to the user application.
func decodeOutgoingHttp(requestBuffer []byte, clienConn, destConn net.Conn, h *hooks.Hook, logger *zap.Logger, baseUrl string) {
	//Matching algorithmm
	//Get the mocks

	tcsMocks := h.GetTcsMocks()

	var basePoints int
	if baseUrl != "" {
		fmt.Println("baseUrl::::", baseUrl)
		// sort the mocks based on the timestamp in the metadata
		// and then send the response to the user in the same order application
		// and as soon as the mocks of the base url gets empty just exit the loop

		var baseMocks []*models.Mock
		var Basetimeline []int64
		for _, mock := range tcsMocks {
			if mock.Spec.HttpReq.URL == baseUrl {
				baseMocks = append(baseMocks, mock)
				strArray := mock.Spec.Metadata["chunkedTime"]
				Basetimeline = getChunkTime(strArray)
			}
		}
		fmt.Println("timeline::::", Basetimeline)
		basePoints = len(baseMocks)
	}
	// x1 --- x2 --- x3
	// Now make events out of this chunk timeline by a delimeter,
	// I mean now sort or give priority to the other requests basis on the timeline
	// means match the event which came first in the timeline than the second that should return the response first
	for {
		var bestMatch *models.Mock
		//Check if the expected header is present
		if bytes.Contains(requestBuffer, []byte("Expect: 100-continue")) {
			//Send the 100 continue response
			_, err := clienConn.Write([]byte("HTTP/1.1 100 Continue\r\n\r\n"))
			if err != nil {
				logger.Error("failed to write the 100 continue response to the user application", zap.Error(err))
				return
			}
			//Read the request buffer again
			newRequest, err := util.ReadBytes(clienConn)
			if err != nil {
				logger.Error("failed to read the request buffer from the user application", zap.Error(err))
				return
			}
			//Append the new request buffer to the old request buffer
			requestBuffer = append(requestBuffer, newRequest...)
		}
		handleChunkedRequests(&requestBuffer, clienConn, destConn, logger)
		//Parse the request buffer
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(requestBuffer)))
		if err != nil {
			logger.Error("failed to parse the http request message", zap.Error(err))
			return
		}

		reqbody, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Error("failed to read from request body", zap.Error(err))
		}

		//parse request url
		reqURL, err := url.Parse(req.URL.String())
		if err != nil {
			logger.Error("failed to parse request url", zap.Error(err))
		}

		//check if req body is a json
		isReqBodyJSON := isJSON(reqbody)
		var eligibleMock []*models.Mock
		var mismatchlog string
		for _, mock := range tcsMocks {
			if mock.Kind == models.HTTP {
				isMockBodyJSON := isJSON([]byte(mock.Spec.HttpReq.Body))
				mismatchlog = ""
				// mock.Spec.HttpReq.Header.
				//the body of mock and request aren't of same type

				if isMockBodyJSON != isReqBodyJSON {
					mismatchlog = "body not matching for req" + mock.Spec.HttpResp.Header["Audit-Id"]
					logger.Debug("body not matching for req", zap.Bool("isMockBodyJSON", isMockBodyJSON), zap.Bool("isReqBodyJSON", isReqBodyJSON))
					continue
				}

				//parse request body url
				parsedURL, err := url.Parse(mock.Spec.HttpReq.URL)
				if err != nil {
					mismatchlog = "failed to parse mock url"
					logger.Error("failed to parse mock url", zap.Error(err))
					continue
				}

				//Check if the path matches
				if parsedURL.Path != reqURL.Path {
					//If it is not the same, continue
					mismatchlog = "path not matching for req"
					logger.Debug("path not matching for req", zap.String("path", parsedURL.Path), zap.String("req", reqURL.Path))
					continue
				}

				//Check if the method matches
				if mock.Spec.HttpReq.Method != models.Method(req.Method) {
					//If it is not the same, continue
					mismatchlog = "method not matching for req"
					logger.Debug("method not matching for req", zap.String("method", string(mock.Spec.HttpReq.Method)), zap.String("req", req.Method))
					continue
				}

				if !mapsHaveSameKeys(mock.Spec.HttpReq.Header, req.Header) {
					// Different headers, so not a match
					mismatchlog = "headers not matching for req"
					logger.Debug("headers not matching for req", zap.Any("mock", mock.Spec.HttpReq.Header), zap.Any("req", req.Header))
					continue
				}

				if !mapsHaveSameKeys(mock.Spec.HttpReq.URLParams, req.URL.Query()) {
					// Different query params, so not a match
					mismatchlog = "query params not matching for req"
					logger.Debug("query params not matching for req", zap.Any("mock", mock.Spec.HttpReq.URLParams), zap.Any("req", req.URL.Query()))
					continue
				}
				eligibleMock = append(eligibleMock, mock)
			}
		}

		if len(eligibleMock) == 0 {
			logger.Error("Didn't match any prexisting http mock because of " + mismatchlog)
			logger.Error("Cannot Find eligible mocks for the outgoing http call", zap.Any("request", string(requestBuffer)))
			// util.Passthrough(clienConn, destConn, [][]byte{requestBuffer}, h.Recover, logger)
			return
		}

		_, bestMatch = util.Fuzzymatch(eligibleMock, requestBuffer, h)

		var bestMatchIndex int
		for idx, mock := range tcsMocks {
			if reflect.DeepEqual(mock, bestMatch) {
				bestMatchIndex = idx
				break
			}
		}
		if h.GetDepsSize() == 0 {
			// logger.Error("failed to mock the output for unrecorded outgoing http call")
			return
		}

		// Fetch the mocked output from the dependency queue
		stub := h.FetchDep(bestMatchIndex)
		if string(reqbody) != "" {
			diffs, err := assertJSONReqBody(stub.Spec.HttpReq.Body, string(reqbody))
			if err != nil {
				logger.Error("failed to assert the json request body for the url", zap.String("url", stub.Spec.HttpReq.URL), zap.Error(err))
				// return
			}

			// Print the differences
			if len(diffs) > 0 {
				fmt.Println("Differences found URL: ", req.Method, ":", req.URL.String(), "Audit-id:: ", stub.Spec.HttpResp.Header["Audit-Id"], ":", color.RedString("MISMATCH"))
				for _, diff := range diffs {
					fmt.Println(diff)
				}
			} else {
				fmt.Println("No differences found.")
			}
		}
		var prevTime int64
		prevTime = getUnixMilliTime(stub.Spec.ReqTimestampMock)
		//calculate chunk time
		

		var chunkedResponses []string
		var chunkedTime []int64
		if stub.Spec.Metadata["chunkedLength"] != "" {
			chunkedTime = getChunkTime(stub.Spec.Metadata["chunkedTime"])
			// fmt.Println("chunkedLength Array:::", stub.Spec.Metadata["chunkedLength"])
			// v.Spec.HttpResp.Body = ""

			// Split the JSON input by newline
			jsonObjects := strings.Split(stub.Spec.HttpResp.Body, "\n")
			// fmt.Println("jsonObjects::::", jsonObjects)
			// Process each JSON object
			for i, jsonObject := range jsonObjects {
				// Skip empty lines
				if jsonObject == "" {
					continue
				}
				// Unmarshal the JSON object
				var data map[string]interface{}
				err := json.Unmarshal([]byte(jsonObject), &data)
				if err != nil {
					fmt.Printf("Error decoding JSON object %d: %v\n", i+1, err)
					continue
				}
				// Print or process the JSON object as needed
				// fmt.Println("json is here for ", i, " ", jsonObject)
				jsonSize := strconv.FormatInt(int64(len(jsonObject)), 16)
				chunkedResponse := fmt.Sprintf("%s\r\n%s\r\n", jsonSize, jsonObject)
				chunkedResponses = append(chunkedResponses, chunkedResponse)
			}
		}

		
		delay := make([]int64, len(chunkedTime))

		for _, chunktime := range chunkedTime {
			// Calculate the difference in milliseconds
			differenceInMilliseconds := chunktime - prevTime
			// Convert the difference to seconds
			delay = append(delay, differenceInMilliseconds/1000)
			prevTime = chunktime
		}
		fmt.Println("delay::::", delay)

		statusLine := fmt.Sprintf("HTTP/%d.%d %d %s\r\n", stub.Spec.HttpReq.ProtoMajor, stub.Spec.HttpReq.ProtoMinor, stub.Spec.HttpResp.StatusCode, http.StatusText(int(stub.Spec.HttpResp.StatusCode)))

		body := stub.Spec.HttpResp.Body
		var respBody string
		var responseString string

		// Fetching the response headers
		header := pkg.ToHttpHeader(stub.Spec.HttpResp.Header)

		//Check if the gzip encoding is present in the header
		if header["Content-Encoding"] != nil && header["Content-Encoding"][0] == "gzip" {
			var compressedBuffer bytes.Buffer
			gw := gzip.NewWriter(&compressedBuffer)
			_, err := gw.Write([]byte(body))
			if err != nil {
				logger.Error("failed to compress the response body", zap.Error(err))
				return
			}
			err = gw.Close()
			if err != nil {
				logger.Error("failed to close the gzip writer", zap.Error(err))
				return
			}
			// logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "the length of the response body: " + strconv.Itoa(len(compressedBuffer.String())))
			respBody = compressedBuffer.String()
			// responseString = statusLine + headers + "\r\n" + compressedBuffer.String()
		} else {
			respBody = body
			// responseString = statusLine + headers + "\r\n" + body
		}
		var headers string
		for key, values := range header {
			if key == "Content-Length" {
				values = []string{strconv.Itoa(len(respBody))}
			}
			for _, value := range values {
				headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
				headers += headerLine
			}
		}

		if len(chunkedResponses) == 0 {
			responseString = statusLine + headers + "\r\n" + "" + respBody
			logger.Debug("the content-length header" + headers)
			_, err = clienConn.Write([]byte(responseString))
			if err != nil {
				logger.Error("failed to write the mock output to the user application", zap.Error(err))
				return
			}
		} else {
			// calculate responsebody length for each chunk , and at last append 0 also.
			headers = "Transfer-Encoding: chunked\r\n"
			for key, values := range header {
				if key == "Content-Length" {
					continue
				}
				for _, value := range values {
					headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
					headers += headerLine
				}
			}

			for idx, v := range chunkedResponses {
				if idx == 0 {
					responseString = statusLine + headers + "\r\n" + v
				} else {
					responseString = v
				}
				if idx == len(chunkedResponses)-1 {
					responseString = responseString + "0\r\n\r\n"
				}
				// time.Sleep(time.Duration(delay[idx]) * time.Millisecond)
				// _, err = clienConn.Write([]byte(responseString))
				// TODO: read from the mocks and set the delay time according to chunkedTime array
				if reqURL.String() == "/apis/apps/v1/deployments?limit=500&resourceVersion=0" || reqURL.String() == "/apis/samplecontroller.k8s.io/v1alpha1/foos?limit=500&resourceVersion=0" {
					time.Sleep(500 * time.Millisecond)
					_, err = clienConn.Write([]byte(responseString))
				} else {
					_, err = clienConn.Write([]byte(responseString))
					time.Sleep(10 * time.Second)
				}

				if err != nil {
					logger.Error("failed to write the mock output to the user application", zap.Error(err))
					return
				}
			}
		}

		// h.PopIndex(bestMatchIndex)

		if reqURL.String() == baseUrl && baseUrl != "" {
			basePoints--
			if basePoints == 0 {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
				go func() {
					select {
					case <-sigChan:
						logger.Info("Received SIGTERM, exiting")
						h.StopUserApplication()
					}
				}()
			}
		}

		requestBuffer, err = util.ReadBytes(clienConn)
		if err != nil {
			logger.Debug("failed to read the request buffer from the client", zap.Error(err))
			logger.Debug("This was the last response from the server: " + string(responseString))
			break
		}

	}

}

// encodeOutgoingHttp function parses the HTTP request and response text messages to capture outgoing network calls as mocks.
func encodeOutgoingHttp(request []byte, clientConn, destConn net.Conn, logger *zap.Logger, h *hooks.Hook, ctx context.Context) error {
	// atomic.AddInt64(&fmt.Sprint(PerConncounter), 1)
	remoteAddr := clientConn.RemoteAddr().(*net.TCPAddr)
	PerConncounter := remoteAddr.Port
	var resp []byte
	var finalResp []byte
	var finalReq []byte
	var chunkedLength []int
	var chunkedTime []int64
	var Overallcounter int64 = 0
	atomic.AddInt64(&Overallcounter, 1)
	var err error
	defer destConn.Close()
	//Writing the request to the server.
	_, err = destConn.Write(request)
	if err != nil {
		logger.Error(fmt.Sprint(PerConncounter)+"failed to write request message to the destination server", zap.Error(err))
		return err
	}
	logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the initial request: " + string(request))
	finalReq = append(finalReq, request...)
	var reqTimestampMock, resTimestampcMock time.Time
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		select {
		case <-sigChan:
			logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "Received SIGTERM, exiting")
			if finalReq == nil || finalResp == nil {
				logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "Sorry request and response are nil")
				return
			} else {

				fmt.Println(fmt.Sprint(PerConncounter), ": Signal-finalReq:\n", string(finalReq))
				fmt.Println(fmt.Sprint(PerConncounter), ": Signal-finalResp:\n", string(finalResp))
				logger.Debug("fmt.Sprint(PerConncounter) "+fmt.Sprint(PerConncounter)+" OverallCounter "+fmt.Sprint(atomic.LoadInt64(&Overallcounter))+","+"", zap.Any("finalReq", len(finalReq)), zap.Any("finalResp", len(finalResp)))
				err := ParseFinalHttp(chunkedTime, chunkedLength, finalReq, finalResp, reqTimestampMock, resTimestampcMock, ctx, logger, h, &Overallcounter, PerConncounter)
				if err != nil {
					logger.Error("failed to parse the final http request and response", zap.Error(err))
					return
				}
			}
		}
	}()

	for {
		//check if the expect : 100-continue header is present

		lines := strings.Split(string(finalReq), "\n")
		var expectHeader string
		for _, line := range lines {
			if strings.HasPrefix(line, "Expect:") {
				expectHeader = strings.TrimSpace(strings.TrimPrefix(line, "Expect:"))
				break
			}
		}

		if expectHeader == "100-continue" {
			//Read if the response from the server is 100-continue
			resp, err = util.ReadBytes(destConn)
			if err != nil {
				logger.Error("failed to read the response message from the server after 100-continue request", zap.Error(err))
				return err
			}
			// write the response message to the client
			_, err = clientConn.Write(resp)
			if err != nil {
				logger.Error("failed to write response message to the user client", zap.Error(err))
				return err
			}
			logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the response from the server after the expect header" + string(resp))
			if string(resp) != "HTTP/1.1 100 Continue\r\n\r\n" {
				logger.Error("failed to get the 100 continue response from the user client")
				return err
			}
			//Reading the request buffer again
			request, err = util.ReadBytes(clientConn)
			if err != nil {
				logger.Error("failed to read the request message from the user client", zap.Error(err))
				return err
			}
			// write the request message to the actual destination server
			_, err = destConn.Write(request)
			if err != nil {
				logger.Error("failed to write request message to the destination server", zap.Error(err))
				return err
			}
			finalReq = append(finalReq, request...)
		}

		// Capture the request timestamp
		reqTimestampMock = time.Now()

		handleChunkedRequests(&finalReq, clientConn, destConn, logger)
		// read the response from the actual server
		resp, err = util.ReadBytes(destConn)
		if err != nil {
			if err == io.EOF {
				logger.Debug("fmt.Sprint(PerConncounter) " + "Response complete, exiting the loop.")
				logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "Got EOF for the response from the server")
				break
			} else {
				logger.Error("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "Got err for the response from the server")
				logger.Error("failed to read the response message from the destination server", zap.Error(err))
				return err
			}
		}

		// Capturing the response timestamp
		resTimestampcMock = time.Now()
		// write the response message to the user client
		_, err = clientConn.Write(resp)
		if err != nil {
			logger.Error("failed to write response message to the user client", zap.Error(err))
			return err
		}
		finalResp = append(finalResp, resp...)
		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the initial response: " + string(resp))
		handleChunkedResponses(&chunkedTime, &chunkedLength, &finalResp, clientConn, destConn, logger, resp, &Overallcounter, PerConncounter)

		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the final response: " + string(finalResp))

		err := ParseFinalHttp(chunkedTime, chunkedLength, finalReq, finalResp, reqTimestampMock, resTimestampcMock, ctx, logger, h, &Overallcounter, PerConncounter)
		if err != nil {
			logger.Error("failed to parse the final http request and response", zap.Error(err))
			return err
		}

		finalReq = nil
		finalResp = nil
		chunkedLength = nil
		chunkedTime = nil

		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "Reading another request on the same connection")
		finalReq, err = util.ReadBytes(clientConn)
		atomic.AddInt64(&Overallcounter, 1)
		if err != nil {
			if err != io.EOF {
				logger.Debug("fmt.Sprint(PerConncounter) "+fmt.Sprint(PerConncounter)+"failed to read the request message from the user client", zap.Error(err))
				logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + "This was the last response from the server: " + string(resp))
			}
			logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + "Response complete, exiting the loop.")
			break
		}
		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the final request: " + string(finalReq))
		// write the request message to the actual destination server
		_, err = destConn.Write(finalReq)
		if err != nil {
			logger.Error("failed to write request message to the destination server", zap.Error(err))
			return err
		}
	}

	return nil
}

func ParseFinalHttp(chunkedTime []int64, chunkedLength []int, finalReq []byte, finalResp []byte, reqTimestampMock, resTimestampcMock time.Time, ctx context.Context, logger *zap.Logger, h *hooks.Hook, Overallcounter *int64, PerConncounter int) error {
	var req *http.Request
	// converts the request message buffer to http request
	req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(finalReq)))
	if err != nil {
		logger.Error("failed to parse the http request message", zap.Error(err))
		return err
	}
	var reqBody []byte
	if req.Body != nil { // Read
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			// TODO right way to log errors
			logger.Error("failed to read the http request body", zap.Error(err))
			return err
		}
	}
	// converts the response message buffer to http response
	respParsed, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(finalResp)), req)
	logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(Overallcounter)) + "PARSED RESPONSE" + fmt.Sprint(respParsed))
	if err != nil {
		logger.Error("failed to parse the http response message", zap.Error(err))
		return err
	}
	//Add the content length to the headers.
	var respBody []byte
	//Checking if the body of the response is empty or does not exist.

	if respParsed.Body != nil { // Read
		if respParsed.Header.Get("Content-Encoding") == "gzip" {
			check := respParsed.Body
			ok, reader := checkIfGzipped(check)
			logger.Debug("fmt.Sprint(PerConncounter) " + "The body is gzip or not" + strconv.FormatBool(ok))
			logger.Debug("fmt.Sprint(PerConncounter) "+"", zap.Any("isGzipped", ok))
			if ok {
				gzipReader, err := gzip.NewReader(reader)
				if err != nil {
					logger.Error("failed to create a gzip reader", zap.Error(err))
					return err
				}
				respParsed.Body = gzipReader
			}
		}
		respBody, err = io.ReadAll(respParsed.Body)
		// logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(&Overallcounter)) + "This is the response body after reading: " + string(respBody))
		if err != nil && err.Error() != "unexpected EOF" {
			logger.Error("failed to read the the http response body", zap.Error(err))
			return err
		}

		logger.Debug("fmt.Sprint(PerConncounter) " + fmt.Sprint(PerConncounter) + " OverallCounter " + fmt.Sprint(atomic.LoadInt64(Overallcounter)) + "This is the response body: " + string(respBody))
		//Add the content length to the headers.
		respParsed.Header.Add("Content-Length", strconv.Itoa(len(respBody)))
	}
	// store the request and responses as mocks
	meta := map[string]string{
		"name":      "Http",
		"type":      models.HttpClient,
		"operation": req.Method,
	}

	if chunkedLength != nil || chunkedTime != nil || len(chunkedLength) != 0 || len(chunkedTime) != 0 {
		meta["chunkedLength"] = fmt.Sprint(chunkedLength)
		meta["chunkedTime"] = fmt.Sprint(chunkedTime)
	}

	err = h.AppendMocks(&models.Mock{
		Version: models.GetVersion(),
		Name:    "mocks",
		Kind:    models.HTTP,
		Spec: models.MockSpec{
			Metadata: meta,
			HttpReq: &models.HttpReq{
				Method:     models.Method(req.Method),
				ProtoMajor: req.ProtoMajor,
				ProtoMinor: req.ProtoMinor,
				URL:        req.URL.String(),
				Header:     pkg.ToYamlHttpHeader(req.Header),
				Body:       string(reqBody),
				URLParams:  pkg.UrlParams(req),
			},
			HttpResp: &models.HttpResp{
				StatusCode: respParsed.StatusCode,
				Header:     pkg.ToYamlHttpHeader(respParsed.Header),
				Body:       string(respBody),
			},
			Created:          time.Now().UnixMilli(),
			ReqTimestampMock: reqTimestampMock,
			ResTimestampMock: resTimestampcMock,
		},
	}, ctx)

	if err != nil {
		logger.Error("failed to store the http mock", zap.Error(err))
		return err
	}

	return nil
}

var (
	keyNotFoundColor  = color.New(color.FgHiBlue).SprintFunc()
	typeMismatchColor = color.New(color.FgYellow).SprintFunc()
	differenceColor   = color.New(color.FgRed, color.Bold).SprintFunc()
)

// to be used in mock assertion
func assertJSONReqBody(jsonBody1, jsonBody2 string) ([]string, error) {
	var data1, data2 map[string]interface{}
	var diffs []string

	if err := json.Unmarshal([]byte(jsonBody1), &data1); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON body 1: %v", err)
	}

	if err := json.Unmarshal([]byte(jsonBody2), &data2); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON body 2: %v", err)
	}

	// Recursive function to compare nested structures
	var compare func(interface{}, interface{}, string)
	compare = func(value1, value2 interface{}, path string) {
		switch v1 := value1.(type) {
		case map[string]interface{}:
			if v2, ok := value2.(map[string]interface{}); ok {
				for key := range v1 {
					newPath := fmt.Sprintf("%s.%s", path, key)
					if val2, ok := v2[key]; !ok {
						diffs = append(diffs, keyNotFoundColor(fmt.Sprintf("Key '%s' not found in second JSON body. Value in the first JSON body: %v", newPath, value1)))
					} else {
						compare(v1[key], val2, newPath)
					}
				}
			} else {
				diffs = append(diffs, typeMismatchColor(fmt.Sprintf("Type mismatch at '%s'. Expected map, actual %T. Value in the first JSON body: %v", path, value2, value1)))
			}
		default:
			// differenceColor(fmt.Sprintf("%s: Expected: %v, Actual: %v", path, value1, value2))
			if !reflect.DeepEqual(value1, value2) {
				diffs = append(diffs, keyNotFoundColor(fmt.Sprintf("%s: ", path))+typeMismatchColor(fmt.Sprintf(" Expected: %v", value1))+differenceColor(fmt.Sprintf(", Actual: %v", value2)))
			}
		}
	}

	compare(data1, data2, "")

	return diffs, nil
}

func ExtractChunkLength(chunkStr string) (int, error) {
	// fmt.Println("chunkStr::::", chunkStr)

	var Totalsize int = 0
	for {
		// Find the position of the first newline
		pos := strings.Index(chunkStr, "\r\n")
		if pos == -1 {
			break
		}

		// Extract the chunk size string
		sizeStr := chunkStr[:pos]

		// Parse the hexadecimal size
		var size int
		_, err := fmt.Sscanf(sizeStr, "%x", &size)
		if err != nil {
			fmt.Println("Error parsing size:", err)
			break
		}

		fmt.Printf("Chunk size: %d\n", size)
		Totalsize = Totalsize + size

		// Check for the last chunk (size 0)
		if size == 0 {
			break
		}

		// Skip past this chunk in the string
		chunkStr = chunkStr[pos+2+size*2+2:] // +2 for \r\n after size, +size*2 for the chunk data, +2 for \r\n after data
	}
	return Totalsize, nil
}

func getUnixMilliTime(parsedTime time.Time) int64 {
	// Convert to Unix timestamp
	unixTimestamp := parsedTime.UnixMilli()
	return unixTimestamp
}

func getChunkTime(strArray string) []int64 {

	// Remove brackets and split the string into individual elements
	strArray = strings.Trim(strArray, "[]")
	elements := strings.Fields(strArray)
	var timeline []int64
	for _, element := range elements {
		num, err := strconv.ParseInt(element, 10, 64)
		if err != nil {
			fmt.Println("Error parsing element:", err)
		}
		timeline = append(timeline, num)
	}
	return timeline
}


// countHTTPChunks takes a buffer containing a chunked HTTP response and returns the total number of chunks.
func countHTTPChunks(buffer []byte) (int, error) {
	reader := bufio.NewReader(bytes.NewReader(buffer))
	chunkCount := 0

	for {
		// Read the next chunk size line.
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			// End of the buffer, break the loop.
			break
		}
		if err != nil {
			return 0, err // Handle the error.
		}

		// Strip the line of \r\n and check if it's empty.
		sizeStr := strings.TrimSpace(line)
		if sizeStr == "" {
			continue // Skip empty lines.
		}

		// Parse the hexadecimal number.
		size, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			return 0, err // Handle the error.
		}

		if size == 0 {
			// Last chunk is of size 0.
			break
		}

		// Skip the chunk data and the trailing \r\n.
		_, err = reader.Discard(int(size + 2))
		if err != nil {
			return 0, err // Handle the error.
		}

		chunkCount++
	}

	return chunkCount, nil
}

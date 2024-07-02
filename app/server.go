package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

type Request struct {
	method  string
	url     string
	headers map[string]string
	body    string
}

type Response struct {
	protocol   string
	method     string
	headers    map[string]string
	body       string
	statusCode string
	statusDesc string
}

func checkEchoSubpath(url string) (string, bool) {
	echoSuffix, found := strings.CutPrefix(url, "/echo/")
	return echoSuffix, found
}

func checkFilesSubpath(url string) (string, bool) {
	filesSuffix, found := strings.CutPrefix(url, "/files/")
	return filesSuffix, found
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		panic(err)
	}
}

func getResponse(response *Response) string {
	var resp strings.Builder
	responseLine := response.protocol + " " + response.statusCode + " " + response.statusDesc + "\r\n"
	resp.WriteString(responseLine)
	for headerKey, headerVal := range response.headers {
		resp.WriteString(headerKey + ": " + headerVal + "\r\n")
	}
	resp.WriteString("\r\n")
	resp.WriteString(response.body)
	return resp.String()
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221", err.Error())
		os.Exit(1)
	}

	for {

		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go func() {
			reader := bufio.NewReader(conn)
			lineCnt := 0
			temp := ""
			request := Request{
				method:  "",
				url:     "",
				headers: map[string]string{},
				body:    "",
			}
			response := Response{
				protocol:   "HTTP/1.1",
				method:     "",
				headers:    map[string]string{},
				body:       "",
				statusCode: "404",
				statusDesc: "Not Found",
			}
			headerKey := ""
			// resp := "HTTP/1.1 404 Not Found\r\n\r\n"

			for {
				line, _, err := reader.ReadLine()

				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println("Error reading the request", err.Error())
				}

				temp = string(line)

				// request line
				if lineCnt == 0 {
					reqLine := strings.Fields(temp)
					request.method = reqLine[0]
					request.url = reqLine[1]
				} else if colonInd := strings.IndexRune(temp, ':'); colonInd != -1 {
					// capture request headers
					headerKey = temp[:colonInd]
					request.headers[headerKey] = strings.TrimSpace(temp[colonInd+1:])
				} else if len(line) == 0 {
					break
				}
				lineCnt++
			}

			if request.method == "GET" {
				if request.url == "/user-agent" || request.url == "/user-agent/" {
					usrAgnt := request.headers["User-Agent"]
					response.statusCode = "200"
					response.statusDesc = "OK"
					response.headers["Content-Type"] = "text/plain"
					response.headers["Content-Length"] = strconv.Itoa(len(usrAgnt))
					response.body = usrAgnt
					// resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(usrAgnt), usrAgnt)
				} else if echoSuff, found := checkEchoSubpath(request.url); found {
					response.statusCode = "200"
					response.statusDesc = "OK"
					response.headers["Content-Type"] = "text/plain"
					response.headers["Content-Length"] = strconv.Itoa(len(echoSuff))
					response.body = echoSuff
					// resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoSuff), echoSuff)
				} else if file, found := checkFilesSubpath(request.url); found {
					dirAbsPath := ""
					if len(os.Args) >= 2 {
						dirAbsPath = os.Args[2]
					}
					fileContent, err := os.Open(dirAbsPath + file)
					if err != nil {
						fmt.Println(err)
					} else {
						defer closeFile(fileContent)
						reader = bufio.NewReader(fileContent)
						temp := make([]byte, 1024)
						respContent := make([]byte, 0, 1024)
						respLength := 0

						for {
							n, err := reader.Read(temp)
							respLength += n
							if err == io.EOF {
								break
							}
							if err != nil {
								break
							}
							respContent = append(respContent, temp...)
						}
						response.statusCode = "200"
						response.statusDesc = "OK"
						response.headers["Content-Type"] = "application/octet-stream"
						response.headers["Content-Length"] = strconv.Itoa(respLength)
						response.body = string(respContent)
						// resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", respLength, string(respContent))
					}
				} else if request.url == "/" {
					response.statusCode = "200"
					response.statusDesc = "OK"
					// resp = "HTTP/1.1 200 OK\r\n\r\n"
				}
			} else if request.method == "POST" {
				if fileName, found := checkFilesSubpath(request.url); found {
					dirAbsPath := ""
					if len(os.Args) >= 2 {
						dirAbsPath = os.Args[2]
					}
					respFile, err := os.Create(dirAbsPath + fileName)
					if err != nil {
						panic(err)
					}
					defer closeFile(respFile)

					body := make([]byte, 0)
					if contentLength, err := strconv.Atoi(request.headers["Content-Length"]); err == nil {
						body = make([]byte, contentLength)
					}
					_, err = reader.Read(body)
					if err != nil {
						fmt.Println("Error reading request body", err)
					}
					_, err = respFile.Write(body)
					if err != nil {
						fmt.Println("Error writing to file", err)
					}
					response.statusCode = "201"
					response.statusDesc = "Created"
					// resp = "HTTP/1.1 201 Created\r\n\r\n"
				}
			}

			if encodings, found := request.headers["Accept-Encoding"]; found {
				for _, encoding := range getClientEncodings(encodings) {
					if encoding == "gzip" {
						if gzipBody, success := getCompressedBody(response.body); success {
							response.body = gzipBody
							response.headers["Content-Encoding"] = "gzip"
							response.headers["Content-Length"] = strconv.Itoa(len(gzipBody))
							response.headers["Content-Type"] = "text/plain"
						}
						break
					}
				}
			}

			resp := getResponse(&response)

			_, err = conn.Write([]byte(resp))
			if err != nil {
				panic(err)
			}

			err = conn.Close()
			if err != nil {
				panic(err)
			}
			fmt.Println("Connection closed...")
		}()

	}
}

func getClientEncodings(encodings string) []string {
	clientSupported := strings.Split(encodings, ",")
	for i, v := range clientSupported {
		clientSupported[i] = strings.TrimSpace(v)
	}
	return clientSupported
}

func getCompressedBody(body string) (string, bool) {
	var buff bytes.Buffer
	writer := gzip.NewWriter(&buff)
	_, err := writer.Write([]byte(body))
	if err != nil {
		fmt.Println("Error during compression ", err)
		return body, false
	}
	if err := writer.Close(); err != nil {
		fmt.Println("Error closing writer", err)
	}
	return buff.String(), true
}

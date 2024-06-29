package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type Request struct {
	method  string
	url     string
	headers map[string]string
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
			}
			headerKey := ""
			resp := "HTTP/1.1 404 Not Found\r\n\r\n"

			for {
				line, _, err := reader.ReadLine()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println("Error reading the request", err.Error())
				}
				if len(line) == 0 {
					break
				}
				temp = string(line)

				// request line
				if lineCnt == 0 {
					reqLine := strings.Fields(temp)
					request.method = reqLine[0]
					request.url = reqLine[1]
				} else {
					// capture request headers
					colonInd := strings.IndexRune(temp, ':')
					headerKey = temp[:colonInd]
					request.headers[headerKey] = strings.TrimSpace(temp[colonInd+1:])
				}
				lineCnt++
			}

			if request.url == "/user-agent" || request.url == "/user-agent/" {
				usrAgnt := request.headers["User-Agent"]
				resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(usrAgnt), usrAgnt)
			} else if echoSuff, found := checkEchoSubpath(request.url); found {
				resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoSuff), echoSuff)
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
							// panic(err)
							break
						}
						respContent = append(respContent, temp...)
					}
					resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", respLength, string(respContent))
				}
			} else if request.url == "/" {
				resp = "HTTP/1.1 200 OK\r\n\r\n"
			}

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

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	reader := bufio.NewReader(conn)
	line, _, err := reader.ReadLine()
	if err != nil && err != io.EOF {
		panic(err)
	}
	reqLine := strings.Fields(strings.TrimSpace(string(line)))
	urlPath := ""
	if len(reqLine) > 2 {
		urlPath = reqLine[1]
	}

	resp := "HTTP/1.1 404 Not Found\r\n\r\n"

	echoSuffix, foundEcho := strings.CutPrefix(urlPath, "/echo/")
	if foundEcho {
		resp = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(echoSuffix), echoSuffix)
	} else if urlPath == "/" {
		resp = "HTTP/1.1 200 OK\r\n\r\n"
	}
	fmt.Println(resp)

	_, err = conn.Write([]byte(resp))
	if err != nil {
		panic(err)
	}

	err = conn.Close()
	if err != nil {
		panic(err)
	}
}

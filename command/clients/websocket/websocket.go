package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"golang.org/x/net/websocket"
)

var stdinScanner = bufio.NewScanner(os.Stdin)

func in(s *string) {
	stdinScanner.Scan()
	*s = stdinScanner.Text()
}

func outf(format string, v ...interface{}) {
	fmt.Fprint(os.Stdout, fmt.Sprintf(format, v...))
}

func main() {
	var host, public, private string

	/*
		outf("Host: ")
		in(&host)
		outf("Public: ")
		in(&public)
		outf("Private: ")
		in(&private)
	*/
	host = "0.0.0.0:9999"
	public = "public"
	private = "private"

	params := url.Values{}
	params.Set("public", public)
	params.Set("private", private)
	wsURL := "ws://" + host + "/command/web/?" + params.Encode()
	log.Printf("URL: %s", wsURL)

	ws, err := websocket.Dial(wsURL, "", "http://"+host)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			var input string
			in(&input)
			websocket.Message.Send(ws, input)
		}
	}()

	for {
		var recieved string
		err := websocket.Message.Receive(ws, &recieved)
		if err == io.EOF {
			log.Print("Closed")
			return
		}

		if err != nil {
			log.Printf("Error: %s", err)
			return
		}

		outf(recieved + "\n")
	}
}

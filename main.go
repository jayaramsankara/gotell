// gotell project main.go
// author jayaramsankara
package main

import (
	"log"
	"os"
	"os/signal"
	"github.com/jayaramsankara/gotell/ws"
)


func init() {
	controlChannel := make(chan string)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		controlChannel <- "close"
		os.Exit(0)
	}()
}

func main() {
	// Start the web socket server and wait in a loop
	err := ws.InitServer()
	if err != nil {
		log.Println("Failed to initiate websocket server.", err)
		os.Exit(100)
	}
}


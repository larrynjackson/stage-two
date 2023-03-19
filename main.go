package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"lnj.com/unix/sockets/list"
	"lnj.com/unix/sockets/list/priorityqueue"
	"lnj.com/unix/sockets/message-handlers"
)

const (
	protocol           = "unix"
	sockAddrIn  string = "/temp/stage-two-socket"
	sockAddrOut string = "/temp/stage-final-socket"
)

type InputMessage struct {
	Id          int64  `json:"id"`
	ShortCode   string `json:"shortcode"`
	Destination string `json:"destination"`
	Tag         string `json:"tag"`
}

type Config struct {
	ConnIn     net.Conn
	ConnOut    net.Conn
	Queue      list.DListNode
	WriteCount int
	ReadCount  int
}

func main() {
	var app Config
	var writeCount int = 0
	var readCount int = 0

	app.WriteCount = writeCount
	app.ReadCount = readCount

	queue := priorityqueue.CreatePQueue()
	app.Queue = queue

	os.RemoveAll(sockAddrIn)
	if _, err := os.Stat(sockAddrIn); err == nil {
		if err := os.RemoveAll(sockAddrIn); err != nil {
			log.Fatal(err)
		}
	}

	listener, err := net.Listen(protocol, sockAddrIn)

	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		fmt.Println("ctrl-c pressed!")
		close(quit)
		os.Exit(0)
	}()

	go app.handleQueuedMessages()

	fmt.Println("> Server launched")
	for {
		connIn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		app.ConnIn = connIn
		go app.readInput()

	}
}

func (app *Config) handleQueuedMessages() {

	var connOut net.Conn
	var err error

	for {
		if !app.Queue.IsEmpty() {
			for {
				connOut, err = net.Dial(protocol, sockAddrOut)
				if err != nil {
					fmt.Println(err)
					time.Sleep(5 * time.Second)
				} else {
					defer connOut.Close()
					break
				}
			}
			fmt.Println("QC:", app.Queue.GetNodeCount())
			ok, m := app.Queue.Take()
			if ok {
				// represents 100 millisecond work on each message
				time.Sleep(time.Millisecond * 100)
				err := m.Write(connOut)
				if err != nil {
					log.Println(err)
				} else {
					app.WriteCount += 1
				}
			}
		} else {
			if app.WriteCount != app.ReadCount {
				fmt.Println("count mis-match: WC=", app.WriteCount, " RC=", app.ReadCount)
			}
			fmt.Println("counts: WC=", app.WriteCount, " RC=", app.ReadCount)
			time.Sleep(5 * time.Second)
		}
	}
}

func (app *Config) readInput() {
	defer app.ConnIn.Close()
	m := &message.Transport{}
	err := m.Read(app.ConnIn)
	if err != nil {
		log.Println(err)
		return
	}
	app.ReadCount += 1
	app.Queue.PriorityPut(m, 0)
}

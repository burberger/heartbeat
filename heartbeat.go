package main

import (
	"encoding/gob"
	"flag"
	"log"
	"net"
	"os"
	"time"
)

const (
	port = ":5656"
)

type Node struct {
	hostname string
}

func connection_handler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	beat := &Node{}
	dec.Decode(beat)
	log.Printf("Recieved : %s\n", beat.hostname)
}

func server() {
	ln, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalln("Could not start server: %s", err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Connection error: %s\n", err)
			continue
		}
		go connection_handler(conn)
	}
}

func client(ip string, sleepTime time.Duration) {
	for {
		conn, err := net.Dial("tcp", ip+port)
		if err != nil {
			log.Printf("Connection error: %s", err)
		} else {
			encoder := gob.NewEncoder(conn)
			hostname, _ := os.Hostname()
            log.Println(hostname)
			msg := &Node{hostname}
			encoder.Encode(msg)
			conn.Close()
		}
		time.Sleep(sleepTime)
	}
}

func main() {
	ip := flag.String("client", "", "Starts program in client mode.  Requires ip address of server.")
	sleepTime := flag.Duration("t", time.Minute*10, "Time between client beacons")
	flag.Parse()
	if *ip == "" {
		server()
	} else {
		client(*ip, *sleepTime)
	}
}

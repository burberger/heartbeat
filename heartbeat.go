package main

import (
	"encoding/gob"
	"flag"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Node struct {
	Hostname  string
	Timestamp time.Time
}

var (
	m          sync.Mutex
	live_hosts map[string]Node
)

func connection_handler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	var heartbeat Node
	dec.Decode(&heartbeat)
	addr := conn.RemoteAddr().String()
	addr = strings.Split(addr, ":")[0]
	m.Lock()
	live_hosts[addr] = heartbeat
	m.Unlock()
}

func map_check(sleepTime time.Duration) {
	for {
		m.Lock()
		for key, value := range live_hosts {
			log.Printf("Address: %s, Hostname: %s, Beacon Time: %q\n", key, value.Hostname, value.Timestamp)
			if time.Since(value.Timestamp) > sleepTime*3 {
				log.Printf("%s removed from map\n", key)
				delete(live_hosts, key)
			}
		}
		m.Unlock()
		time.Sleep(sleepTime)
	}
}

func server(port string) {
    ln, err := net.Listen("tcp", ":"+port)
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

func client(ip, port string, sleepTime time.Duration) {
	for {
		conn, err := net.Dial("tcp", ip+":"+port)
		if err != nil {
			log.Printf("Connection error: %s", err)
		} else {
			encoder := gob.NewEncoder(conn)
			hn, _ := os.Hostname()
			err := encoder.Encode(Node{hn, time.Now()})
			if err != nil {
				log.Println("Encoding error:", err)
			}
			conn.Close()
		}
		time.Sleep(sleepTime)
	}
}

func main() {
	ip := flag.String("client", "", "Starts program in client mode.  Requires address of server.")
	sleepTime := flag.Duration("t", time.Minute*10, "Expected beacon delay.  Affects both client and server modes.")
	port := flag.String("port", "5656", "Specify the port server is running on.")
	flag.Parse()
	if *ip == "" {
		live_hosts = make(map[string]Node)
		go map_check(*sleepTime)
		server(*port)
	} else {
		client(*ip, *port, *sleepTime)
	}
}

package main

import (
	"encoding/gob"
	"flag"
	"html/template"
	"log"
	"log/syslog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Node holds the information representing a heartbeat client
type Node struct {
	Hostname  string
	Timestamp time.Time
}

// Declare global node map and mutex for cross thread access
var (
	m          sync.RWMutex
	slog       *log.Logger
	live_hosts map[string]Node
)

// Precompile template
var templates = template.Must(template.ParseFiles("list.html"))

// When you recieve a new connection from a heartbeat client,
// Lock the table for writing and build / update a node.
func connection_handler(conn net.Conn) {
	dec := gob.NewDecoder(conn)
	var heartbeat Node
	dec.Decode(&heartbeat)
	// Strip transmit port number from connection
	addr := conn.RemoteAddr().String()
	addr = strings.Split(addr, ":")[0]
	m.Lock()
	live_hosts[addr] = heartbeat
	m.Unlock()
}

// Processes the map for dead hosts and logs who the system is aware of.
// Read lock allows web requests to come in during a map check cycle.
func map_check(sleepTime time.Duration) {
	for {
		m.RLock()
		for key, value := range live_hosts {
			if time.Since(value.Timestamp) > sleepTime*3 {
				slog.Printf("Machine %s : %s timed out, removed from map\n", key, value.Hostname)
				delete(live_hosts, key)
			}
		}
		m.RUnlock()
		time.Sleep(sleepTime)
	}
}

// Dispatches connections to goroutines for heartbeat server
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

// Responds to web server requests for map data
func root_handler(w http.ResponseWriter, r *http.Request) {
	m.RLock()
	err := templates.ExecuteTemplate(w, "list.html", live_hosts)
	m.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Open a connection to the server, attempt to beacon, close connection
// client loops forever, sleeping for sleepTime between beacons
func client(ip, port string, sleepTime time.Duration) {
	for {
		conn, err := net.Dial("tcp", ip+":"+port)
		if err != nil {
			slog.Println("Connection error:", err)
		} else {
			encoder := gob.NewEncoder(conn)
			hn, _ := os.Hostname()
			err := encoder.Encode(Node{hn, time.Now()})
			if err != nil {
				slog.Println("Encoding error:", err)
			}
			conn.Close()
		}
		time.Sleep(sleepTime)
	}
}

func main() {
	// Command line flags
	ip := flag.String("client", "", "Starts program in client mode.  Requires address of server.")
	sleepTime := flag.Duration("t", time.Minute*10, "Expected beacon delay.  Affects both client and server modes.")
	port := flag.String("port", "5656", "Specify the port server is running on.")
	flag.Parse()

	// Start up system logger connection, warn on fail
    var err error
    slog, err = syslog.NewLogger(syslog.LOG_WARNING|syslog.LOG_DAEMON, log.LstdFlags)
	if err != nil {
		log.Println("Syslog error:", err)
	}

	// No client flag, serve heartbeat listener and web interface
	if *ip == "" {
		live_hosts = make(map[string]Node)
		go map_check(*sleepTime)
		go server(*port)
		http.HandleFunc("/", root_handler)
		http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public/"))))
		http.ListenAndServe(":8080", nil)
	} else {
		client(*ip, *port, *sleepTime)
	}
}

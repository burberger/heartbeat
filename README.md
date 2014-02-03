heartbeat
=========

Simple client server heartbeat program written in go.

Usage:

  heartbeat -t (time) -client (servername) -p (port)

* Time is specified with units.  10s, 5m, 3h, 4d, etc.  Default 10m.
* Client flag places daemon in client mode.  Must be supplied with server address.  Default server mode.
* Port flag sets the port that the client and server communicate on.  Default 5656.

When running in server mode, a web server is started which serves all active client nodes in a table.
Default port for server mode is 8080.

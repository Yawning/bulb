// Onion example.
//
// To the extent possible under law, David Stainton waived all copyright
// and related or neighboring rights to this bulb source file, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/yawning/bulb"
	"github.com/yawning/bulb/utils"
)

func main() {
	// Connect to a running tor instance.
	// unix domain socket - c, err := bulb.Dial("unix", "/var/run/tor/control")
	c, err := bulb.Dial("tcp4", "127.0.0.1:9051")

	if err != nil {
		log.Fatalf("failed to connect to control port: %v", err)
	}
	defer c.Close()

	// See what's really going on under the hood.
	// Do not enable in production.
	c.Debug(true)

	// Authenticate with the control port.  The password argument
	// here can be "" if no password is set (CookieAuth, no auth).
	if err := c.Authenticate("ExamplePassword"); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	options := utils.OnionListenerOptions{
		OnionKeyFile:     "echoOnionKey",
		OnionServicePort: 80,
		LocalAddr:        "127.0.0.1:8080",
		ControlNetwork:   "tcp",
		ControlAddr:      "127.0.0.1:9051",
	}

	listener, err := utils.NewOnionListener(&options)
	if err != nil {
		log.Fatal(err)
	}

	addr := listener.Addr()
	onionAddr := addr.String()
	fmt.Printf("onion echo server: listening to %s\n", onionAddr)

	defer listener.Close()

	for {
		// Wait for a connection.
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			// Echo all incoming data.
			io.Copy(c, c)
			// Shut down the connection.
			c.Close()
		}(conn)
	}
}

// connection.go - A TorDialer and OnionListener...
//
// To the extent possible under law, David Stainton waived all copyright
// and related or neighboring rights to this bulb source file, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

// Package utils implements useful utilities for dealing with Tor and it's
// control port.
package utils

import (
	"fmt"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net"
	"strings"

	"github.com/yawning/bulb"
)

type TorDialer struct {
	dialer proxy.Dialer
}

func NewTorDialer(network string, addr string, auth *proxy.Auth) (*TorDialer, error) {
	var err error
	var socksDialer proxy.Dialer
	forwardDialer := net.Dialer{}

	socksDialer, err = proxy.SOCKS5(network, addr, auth, &forwardDialer)
	if err != nil {
		return nil, fmt.Errorf("TorDialer: Failed to create socks dialer: %v\n", err)
	}

	torDialer := TorDialer{
		dialer: socksDialer,
	}

	return &torDialer, nil
}

func (t *TorDialer) Dial(network, addr string) (net.Conn, error) {
	conn, err := t.dialer.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("TorDialer: failed to dial tor socks port: %s\n", err)
	}
	return conn, nil
}

type OnionAddr struct {
	network string
	address string
}

func (o OnionAddr) Network() string {
	return o.network
}

func (o OnionAddr) String() string {
	return o.address
}

type OnionListenerOptions struct {
	// OnionKeyFile is used to persist onion service key material
	OnionKeyFile string

	// OnionServicePort is the virtport of the tor onion service
	OnionServicePort int
	// LocalAddr is the address of the local TCP Listener
	LocalAddr string

	// ControlAddr is the Tor control port address
	ControlAddr string
	// ControlNetwork is the network type of the Tor control port
	ControlNetwork  string
	ControlPassword string
}

type OnionListener struct {
	options       *OnionListenerOptions
	controller    *bulb.Conn
	onionInfo     *bulb.OnionInfo
	localListener net.Listener
}

func NewOnionListener(options *OnionListenerOptions) (*OnionListener, error) {
	listener := OnionListener{
		options: options,
	}
	if options.ControlAddr == "" || options.ControlNetwork == "" {
		return nil, fmt.Errorf("Tor control port address not specified.")
	}
	err := listener.initOnion()
	if err != nil {
		return nil, err
	}
	return &listener, nil
}

func (o *OnionListener) controlAuth() error {
	var err error

	o.controller, err = bulb.Dial(o.options.ControlNetwork, o.options.ControlAddr)
	if err != nil {
		return fmt.Errorf("OnionListener: Failed to connect to Tor control port: %s\n", err)
	}

	err = o.controller.Authenticate(o.options.ControlPassword)
	if err != nil {
		return fmt.Errorf("OnionListener: Failed authenticate with Tor control port: %s\n", err)
	}

	return nil
}

// readOnionKeyFile returns the key type string, key content string and nil error
// ...otherwise error will be non-nil.
func (o *OnionListener) readOnionKeyFile() (string, string, error) {
	var fields []string
	onionKeyBytes := make([]byte, 100)
	onionKeyBytes, err := ioutil.ReadFile(o.options.OnionKeyFile)
	if err != nil {
		return "", "", err
	}
	fields = strings.Split(string(onionKeyBytes), ":")
	return fields[0], fields[1], nil
}

func (o *OnionListener) writeOnionKeyFile(keyType, keyContent string) error {
	keyData := fmt.Sprintf("%s:%s", keyType, keyContent)
	err := ioutil.WriteFile(o.options.OnionKeyFile, []byte(keyData), 0644)
	return err
}

func (o *OnionListener) initOnion() error {
	//var serviceId string
	var keyType, keyContent string
	var err error
	onionKeyContent := ""
	onionKeyType := ""

	if o.options.OnionKeyFile != "" {
		onionKeyType, onionKeyContent, err = o.readOnionKeyFile()
	}

	err = o.controlAuth()
	if err != nil {
		return err
	}

	if onionKeyType != "" {
		o.onionInfo, err = o.controller.AddOnion(o.options.OnionServicePort, o.options.LocalAddr, onionKeyType, onionKeyContent, false)
	} else {
		o.onionInfo, err = o.controller.AddOnion(o.options.OnionServicePort, o.options.LocalAddr, "", "", true)
	}

	if err != nil {
		return fmt.Errorf("OnionListener: create hidden service fail: %s\n", err)
	}

	if o.options.OnionKeyFile != "" {
		err = o.writeOnionKeyFile(keyType, keyContent)
		if err != nil {
			return fmt.Errorf("OnionListener: failed to write key file to disk: %s\n", err)
		}
	}

	o.localListener, err = net.Listen("tcp", o.options.LocalAddr)
	if err != nil {
		return fmt.Errorf("OnionListener: local TCP listen error: %s\n", err)
	}

	return nil
}

func (o *OnionListener) Accept() (net.Conn, error) {
	var conn net.Conn
	var err error

	if o.onionInfo.OnionId == "" {
		return nil, fmt.Errorf("OnionListener: onion service not initialized.\n")
	}

	conn, err = o.localListener.Accept()
	if err != nil {
		return nil, fmt.Errorf("OnionListener: local TCP connection Accept failure: %s\n", err)
	}

	return conn, nil
}

func (o *OnionListener) Close() error {
	err := o.controller.DeleteOnion(o.onionInfo.OnionId)
	if err != nil {
		return fmt.Errorf("OnionListener: DeleteHiddenService failure: %s\n", err)
	}
	return nil
}

func (o *OnionListener) Addr() net.Addr {
	return OnionAddr{
		network: "tor",
		address: fmt.Sprintf("%s.onion:%d", o.onionInfo.OnionId, o.options.OnionServicePort),
	}
}

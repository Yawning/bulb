// cmd_onion.go - various onion service commands: ADD_ONION, DEL_ONION...
//
// To the extent possible under law, David Stainton waived all copyright
// and related or neighboring rights to this module of bulb, using the creative
// commons "cc0" public domain dedication. See LICENSE or
// <http://creativecommons.org/publicdomain/zero/1.0/> for full details.

package bulb

import (
	"fmt"
	"strings"
)

// OnionInfo is the result of the AddOnion command.
type OnionInfo struct {
	OnionId string
	KeyType string
	Key     string

	RawResponse *Response
}

// AddOnion issues an ADD_ONION command and returns the parsed response.
func (c *Conn) AddOnion(virtPort int, target, keyType, keyContent string, new bool) (*OnionInfo, error) {
	var fields []string
	request := "ADD_ONION "
	onionInfo := OnionInfo{}

	if new {
		request += "NEW:BEST"
	} else {
		request += fmt.Sprintf("%s:%s", keyType, keyContent)
	}
	request += fmt.Sprintf(" Port=%d,%s\n", virtPort, target)
	fmt.Printf("DEBUG request: %s\n", request)
	response, err := c.Request(request)
	if err != nil {
		return nil, err
	}

	onionInfo.RawResponse = response
	fields = strings.Split(fmt.Sprintf("%s", response.Data), "ServiceID=")
	fields = strings.Split(fields[1], " ")
	onionInfo.OnionId = fields[0]

	if new {
		fields = strings.Split(fmt.Sprintf("%s", response.Data), "PrivateKey=")
		fields = strings.Split(fields[1], ":")
		onionInfo.KeyType = fields[0]
		fields = strings.Split(fields[1], "\n")
		onionInfo.Key = fields[0]
	}

	return &onionInfo, nil
}

func (c *Conn) DeleteOnion(serviceId string) error {
	var deleteCmd string = fmt.Sprintf("DEL_ONION %s\n", serviceId)
	_, err := c.Request(deleteCmd)
	return err
}

package gosocketio

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ligoo/golang-socketio/protocol"
)

var (
	ErrorSendTimeout     = errors.New("Timeout")
	ErrorSocketOverflood = errors.New("Socket overflood")
)

/**
Send message packet to socket
*/
func send(msg *protocol.Message, c *Channel, args interface{}) error {
	//preventing json/encoding "index out of range" panic
	defer c.rh.call(c)

	if args != nil {
		json, err := json.Marshal(&args)
		if err != nil {
			return err
		}

		msg.Args = string(json)
	}

	command, err := protocol.Encode(msg)
	if err != nil {
		return err
	}

	if len(c.out) == QueueBufferSize {
		return ErrorSocketOverflood
	}

	c.sendOut(command)

	return nil
}

/**
Create packet based on given data and send it
*/
func (c *Channel) Emit(method string, args interface{}) error {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeEmit,
		Method: method,
	}

	return send(msg, c, args)
}

/**
Create ack packet based on given data and send it and receive response
*/
func (c *Channel) Ack(method string, args interface{}, timeout time.Duration) (string, error) {
	msg := &protocol.Message{
		Type:   protocol.MessageTypeAckRequest,
		AckId:  c.ack.getNextId(),
		Method: method,
	}

	waiter := make(chan string)
	c.ack.addWaiter(msg.AckId, waiter)

	err := send(msg, c, args)
	if err != nil {
		c.ack.removeWaiter(msg.AckId)
	}

	select {
	case <-c.done:
		return "", nil
	case <-c.aliveC:
		return "", nil
	case result := <-waiter:
		return result, nil
	case <-time.After(timeout):
		c.ack.removeWaiter(msg.AckId)
		return "", ErrorSendTimeout
	}
}

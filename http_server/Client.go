package httpserver

import (
	"AirlinkStreamBridge/receivers"
	"AirlinkStreamBridge/senders"
	"errors"
)

type Client struct {
	wr     *receivers.WebrtcReceiver
	sender *senders.ISender
}

func NewClient(wr *receivers.WebrtcReceiver, sender *senders.ISender) (*Client, error) {
	client := &Client{wr: wr, sender: sender}
	return client, nil
}

func (client *Client) GetReceiver() (*receivers.WebrtcReceiver, error) {
	if (client != nil) && (client.wr != nil) {
		return client.wr, nil
	}
	return nil, errors.New("client or receiver is nil")
}

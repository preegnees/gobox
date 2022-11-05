package client

import (
	"context"

	pc "github.com/preegnees/gobox/pkg/client/file/protocol"
)

var _ IClient = (*client)(nil)

type IClient interface {
	SendError(int, context.CancelFunc, error)
	SendDeviation(pc.Info)
}

type client struct{}

func New() (IClient, error) {
	return client{}, nil
}

func (c client) SendError(indentifier int, ctx context.CancelFunc, err error) {
	
}

func (c client) SendDeviation(info pc.Info) {
	
}

package sandboxrt

import (
	"context"
)

type IPCPipe interface {
	Send([]byte) ([]byte, error)
	SetReceiveHandler(func([]byte) ([]byte, error))
}

type IPCHost interface {
	IPCPipe
	RegisterFunction(name string, fn interface{}) error
	Serve(context.Context) error
}

type IPCClient interface {
	IPCPipe
	// If call returns an error, it is not an error from this function
	CallFunction(name string, args []interface{}) ([]interface{}, error)
}

const localActivityPrefix = "__local_activity__"

func RegisterLocalActivityFunc(h IPCHost, name string, fn interface{}) error {
	panic("TODO")
}

func MakeLocalActivityProxyFunc(c IPCClient, name string, fn interface{}) (interface{}, error) {
	// TODO(cretz): reflect.MakeFunc
	panic("TODO")
}

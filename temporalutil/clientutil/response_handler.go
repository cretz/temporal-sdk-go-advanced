package clientutil

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type CallResponseHandlerOptions struct {
	TaskQueue string
	Worker    worker.Worker
}

type CallResponseHandler struct {
	taskQueue           string
	worker              worker.Worker
	pendingRequests     map[string]chan<- interface{}
	pendingRequestsLock sync.RWMutex
}

func NewCallResponseHandler(opts CallResponseHandlerOptions) (*CallResponseHandler, error) {
	if opts.TaskQueue == "" {
		return nil, fmt.Errorf("missing task queue")
	} else if opts.Worker == nil {
		return nil, fmt.Errorf("missing worker")
	}
	return &CallResponseHandler{
		taskQueue:       opts.TaskQueue,
		worker:          opts.Worker,
		pendingRequests: map[string]chan<- interface{}{},
	}, nil
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (c *CallResponseHandler) TaskQueue() string { return c.taskQueue }

func (c *CallResponseHandler) Call(
	ctx context.Context,
	cl client.Client,
	workflowID string,
	runID string,
	signalName string,
	newReq func(id string) interface{},
) (interface{}, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, okCh, errCh := c.PrepareCall(ctx)
	if err := cl.SignalWorkflow(ctx, workflowID, runID, signalName, newReq(id)); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-okCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

func (c *CallResponseHandler) PrepareCall(ctx context.Context) (id string, chOk <-chan interface{}, chErr <-chan error) {
	// Create ID and result channels
	id = uuid.NewString()
	okCh := make(chan interface{}, 1)
	errCh := make(chan error, 1)

	// Add response channel
	respCh := make(chan interface{}, 1)
	c.pendingRequestsLock.Lock()
	c.pendingRequests[id] = respCh
	c.pendingRequestsLock.Unlock()

	// Listen async
	go func() {
		// Remove when done
		defer func() {
			c.pendingRequestsLock.Lock()
			defer c.pendingRequestsLock.Unlock()
			delete(c.pendingRequests, id)
		}()

		// Wait for response or context done
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
		case resp := <-respCh:
			okCh <- resp
		}
	}()

	return id, okCh, errCh
}

func (c *CallResponseHandler) AddResponseType(activityName string, typ reflect.Type, idField string) error {
	// Check type and field
	structTyp := typ
	if typ.Kind() == reflect.Ptr {
		structTyp = typ.Elem()
	}
	if structTyp.Kind() != reflect.Struct {
		return fmt.Errorf("expect type to be struct or pointer to struct, got %v", typ)
	} else if _, fieldExists := structTyp.FieldByName(idField); !fieldExists {
		return fmt.Errorf("field %q does not exist on type %v", idField, typ)
	}

	// Make a dynamic func accepting context + type and returning error
	fnVal := reflect.MakeFunc(
		reflect.FuncOf([]reflect.Type{contextType, typ}, []reflect.Type{errorType}, false),
		func(args []reflect.Value) []reflect.Value {
			err := c.onResponse(args[1], idField)
			if err == nil {
				return []reflect.Value{reflect.Zero(errorType)}
			}
			return []reflect.Value{reflect.ValueOf(err)}
		},
	)
	c.worker.RegisterActivityWithOptions(fnVal.Interface(), activity.RegisterOptions{
		Name:                          activityName,
		DisableAlreadyRegisteredCheck: true,
	})
	return nil
}

func (c *CallResponseHandler) onResponse(val reflect.Value, idField string) error {
	// Extract ID
	structVal := val
	if structVal.Kind() == reflect.Ptr {
		structVal = val.Elem()
	}
	id := structVal.FieldByName(idField).String()

	// Get the channel to respond to
	c.pendingRequestsLock.RLock()
	respCh := c.pendingRequests[id]
	c.pendingRequestsLock.RUnlock()
	// We choose not to log or error if a response is not pending because it is
	// normal behavior for a requester to have closed the context and stop waiting
	if respCh == nil {
		return nil
	}

	// Send non-blocking since the channel should have enough room. Technically
	// during a situation where this worker was too busy for this activity to
	// return, the responseActivity could be called again for the same response
	// during retry from the other side. This will just result in a no-op since
	// the channel does not have room.
	select {
	case respCh <- val.Interface():
	default:
	}
	return nil
}

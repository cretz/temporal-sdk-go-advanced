package temporalng1

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Workflow[In, Out any] struct {
	Name string
}

func (*Workflow[In, Out]) Register(worker.Worker, func(Context, In) (Out, error)) {
	panic("TODO")
}

type WorkflowStartOptions struct {
	ID        string
	TaskQueue string
	// Other stuff
}

type WorkflowHandle[Out any] struct {
	Client      client.Client
	ID          string
	RunID       string
	ResultRunID string
}

func (*Workflow[In, Out]) GetWorkflowHandle(client client.Client, id string) *WorkflowHandle[Out] {
	panic("TODO")
}

func (*Workflow[In, Out]) GetWorkflowHandleForRunID(client client.Client, id, runID string) *WorkflowHandle[Out] {
	panic("TODO")
}

var _ WorkflowRef = &WorkflowHandle[struct{}]{}

func (*WorkflowHandle[Out]) GetID() string {
	panic("TODO")
}

func (*WorkflowHandle[Out]) GetRunID() string {
	panic("TODO")
}

type WorkflowRef interface {
	GetID() string
	GetRunID() string
}

func (*Workflow[In, Out]) Start(context.Context, client.Client, In, WorkflowStartOptions) (*WorkflowHandle[Out], error) {
	panic("TODO")
}

func (*Workflow[In, Out]) Execute(context.Context, client.Client, In, WorkflowStartOptions) (Out, error) {
	panic("TODO")
}

func (*WorkflowHandle[Out]) Get(context.Context) (Out, error) {
	panic("TODO")
}

type Activity[In, Out any] struct {
	Name string
}

func (*Activity[In, Out]) Register(worker.Worker, func(context.Context, In) (Out, error)) {
	panic("TODO")
}

type ActivityExecuteOptions struct {
	// ...
}

func (*Activity[In, Out]) Execute(Context, In, ActivityExecuteOptions) *Future[Out] {
	panic("TODO")
}

type Signal[In any] struct {
	Name string
}

type SignalSendOptions struct {
}

func (*Signal[In]) Send(context.Context, WorkflowRef, In, SignalSendOptions) error {
	panic("TODO")
}

func (*Signal[In]) GetChannel(Context) *ReceiveChannel[In] {
	panic("TODO")
}

type Query[In, Out any] struct {
	Name string
}

type QueryInvokeOptions struct {
}

func (*Query[In, Out]) Invoke(context.Context, WorkflowRef, In, QueryInvokeOptions) (Out, error) {
	panic("TODO")
}

func (*Query[In, Out]) SetHandler(Context, func(In) (Out, error)) {
	panic("TODO")
}

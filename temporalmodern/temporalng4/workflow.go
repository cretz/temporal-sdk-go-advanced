package temporalng4

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Workflow[In, Out any] struct {
}

func WorkflowSpec[Wf workflowSpec]() Wf {
	panic("TODO")
}

type workflowSpec interface {
	workflowSpec()
}

func (Workflow[In, Out]) workflowSpec() {}

func (Workflow[In, Out]) Register(
	worker.Worker,
	func(Context, In) (Out, error),
) {
}

type WorkflowSignal[In any] struct{}

type WorkflowQuery[In, Out any] struct{}

func SetWorkflowQueryHandler[In, Out any](Context, WorkflowQuery[In, Out], func(In) (Out, error)) {
}

func GetWorkflowSignalChannel[In any](Context, WorkflowSignal[In]) ReceiveChannel[In] {
	panic("TODO")
}

func (Workflow[In, Out]) Start(context.Context, client.Client, In, WorkflowStartOptions) (*WorkflowHandle[Out], error) {
	panic("TODO")
}

func (Workflow[In, Out]) GetHandle(client client.Client, id string) *WorkflowHandle[Out] {
	panic("TODO")
}

type WorkflowStartOptions struct {
	ID        string
	TaskQueue string
	// Other stuff
}

func (WorkflowSignal[In]) Send(context.Context, client.Client, In, WorkflowSignalOptions) error {
	panic("TODO")
}

type WorkflowSignalOptions struct {
	Workflow   WorkflowRef
	WorkflowID string
	RunID      string
}

func (WorkflowQuery[In, Out]) Invoke(context.Context, client.Client, In, WorkflowQueryOptions) (Out, error) {
	panic("TODO")
}

type WorkflowQueryOptions struct {
	Workflow   WorkflowRef
	WorkflowID string
	RunID      string
}

func RegisterActivity[In, Out any](
	worker.Worker,
	func(context.Context, In) (Out, error),
) {
}

type WorkflowHandle[Out any] struct {
	Client      client.Client
	ID          string
	RunID       string
	ResultRunID string
}

var _ WorkflowRef = (*WorkflowHandle[any])(nil)

func (*WorkflowHandle[Out]) Get(context.Context) (Out, error) {
	panic("TODO")
}

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

func StartActivity[In, Out any](
	Context,
	func(context.Context, In) (Out, error),
	In,
	StartActivityOptions,
) Future[Out] {
	panic("TODO")
}

type StartActivityOptions struct {
	// ...
}

package temporalng3

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Workflow[In, Out any] struct {
}

type WorkflowSpec interface {
	workflowSpec()
}

type WorkflowOutSpec[Out any] interface {
	WorkflowSpec
	workflowOutSpec()
}

type WorkflowInOutSpec[In, Out any] interface {
	WorkflowOutSpec[Out]
	workflowInOutSpec()
}

func (Workflow[In, Out]) workflowSpec() {}

func (Workflow[In, Out]) workflowOutSpec() {}

func (Workflow[In, Out]) workflowInOutSpec() {}

func WorkflowImpl[Wf WorkflowSpec]() Wf {
	panic("TODO")
}

func (Workflow[In, Out]) Start(context.Context, In, WorkflowStartOptions) (*WorkflowHandle[Out], error) {
	panic("TODO")
}

func (Workflow[In, Out]) GetHandle(id string) *WorkflowHandle[Out] {
	panic("TODO")
}

type WorkflowStartOptions struct {
	ID        string
	TaskQueue string
	// Other stuff
}

type WorkflowSignalOptions struct {
	Workflow   WorkflowRef
	WorkflowID string
	RunID      string
}

type WorkflowSignal[In any] func(context.Context, In, WorkflowSignalOptions) error

func (WorkflowSignal[In]) GetChannel(Context) ReceiveChannel[In] {
	panic("TODO")
}

type WorkflowQueryOptions struct {
	Workflow   WorkflowRef
	WorkflowID string
	RunID      string
}

type WorkflowQuery[In, Out any] func(context.Context, In, WorkflowQueryOptions) (Out, error)

func (WorkflowQuery[In, Out]) SetHandler(Context, func(In) (Out, error)) {
}

func NewWorkflowClient[Wf WorkflowSpec](client.Client) Wf {
	panic("TODO")
}

func GetWorkflowHandle[Wf WorkflowOutSpec[Out], Out any](client client.Client, id string) *WorkflowHandle[Out] {
	panic("TODO")
}

func RegisterWorkflow[Wf WorkflowInOutSpec[In, Out], In, Out any](
	worker.Worker,
	func(Context, Wf, In) (Out, error),
) {
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

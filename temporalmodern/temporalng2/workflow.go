package temporalng2

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type Workflow[In, Out any] interface {
	Run(Context, In) (Out, error)
}

type CustomNaming[Meta any] interface {
}

func RegisterWorkflow[Wf Workflow[In, Out], In, Out any](worker.Worker, Wf) {
	panic("TODO")
}

type WorkflowHandle[Wf Workflow[In, Out], In, Out any] struct {
	Client      client.Client
	ID          string
	RunID       string
	ResultRunID string
}

func (*WorkflowHandle[Wf, In, Out]) Get(context.Context) (Out, error) {
	panic("TODO")
}

func GetWorkflowHandle[Wf Workflow[In, Out], In, Out any](
	client client.Client,
	workflowRun func(Wf, Context, In) (Out, error),
	id string,
) *WorkflowHandle[Wf, In, Out] {
	panic("TODO")
}

func GetWorkflowHandleForRunID[Wf Workflow[In, Out], In, Out any](
	client client.Client,
	workflowRun func(Wf, Context, In) (Out, error),
	id, runID string,
) *WorkflowHandle[Wf, In, Out] {
	panic("TODO")
}

var _ WorkflowRef = (*WorkflowHandle[Workflow[any, any], any, any])(nil)

func (*WorkflowHandle[Wf, In, Out]) GetID() string {
	panic("TODO")
}

func (*WorkflowHandle[Wf, In, Out]) GetRunID() string {
	panic("TODO")
}

type WorkflowRef interface {
	GetID() string
	GetRunID() string
}

type StartWorkflowOptions struct {
	ID        string
	TaskQueue string
	// Other stuff
}

func StartWorkflow[Wf Workflow[In, Out], In, Out any](
	context.Context,
	client.Client,
	func(Wf, Context, In) (Out, error),
	In,
	StartWorkflowOptions,
) (*WorkflowHandle[Wf, In, Out], error) {
	panic("TODO")
}

type SignalWorkflowOptions struct {
}

func SignalWorkflow[Wf Workflow[WfIn, WfOut], WfIn, WfOut, In any](
	context.Context,
	*WorkflowHandle[Wf, WfIn, WfOut],
	func(Wf, Context, In),
	In,
	SignalWorkflowOptions,
) error {
	panic("TODO")
}

type QueryWorkflowOptions struct {
}

func QueryWorkflow[Wf Workflow[WfIn, WfOut], WfIn, WfOut, In, Out any](
	context.Context,
	*WorkflowHandle[Wf, WfIn, WfOut],
	func(Wf, In) (Out, error),
	In,
	QueryWorkflowOptions,
) (Out, error) {
	panic("TODO")
}

type Activity[In, Out any] func(context.Context, In) (Out, error)

func (Activity[In, Out]) WithName(string) Activity[In, Out] {
	panic("TODO")
}

func RegisterActivities[Act any](worker.Worker, Act) {
	panic("TODO")
}

func ExecuteActivity[Act any, In, Out any](
	Context,
	func(Act, context.Context, In) (Out, error),
	In,
	ExecuteActivityOptions,
) Future[Out] {
	panic("TODO")
}

type ExecuteActivityOptions struct {
	// ...
}

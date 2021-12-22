package proposed

import (
	"context"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type SomeWorkflow1Request struct{}
type SomeWorkflow1Response struct{}
type SomeQuery1Response struct{}
type SomeQuery2Request struct{}
type SomeQuery2Response struct{}
type SomeSignal2Request struct{}
type SomeCall1Response struct{}
type SomeCall3Request struct{}
type SomeCall3Response struct{}

type SimpleClient interface {
	SomeWorkflow1(
		ctx context.Context,
		req *SomeWorkflow1Request,
		opts client.StartWorkflowOptions,
	) (SimpleClient_SomeWorkflow1, error)
}

type SimpleClient_SomeWorkflow1 interface {
	client.WorkflowRun

	GetResult(context.Context) (*SomeWorkflow1Response, error)

	SomeQuery1(context.Context) (*SomeQuery1Response, error)
	SomeQuery2(context.Context, *SomeQuery2Request) (*SomeQuery2Response, error)

	SomeSignal1(context.Context) error
	SomeSignal2(context.Context, *SomeSignal2Request) error

	SomeCall1(context.Context) (*SomeCall1Response, error)
	SomeCall2(context.Context) error
	SomeCall3(context.Context, *SomeCall3Request) (*SomeCall3Response, error)
}

type simpleClient struct{ client client.Client }

func NewSimpleClient(c client.Client) SimpleClient { return simpleClient{c} }

func (s simpleClient) SomeWorkflow1(
	ctx context.Context,
	req *SomeWorkflow1Request,
	opts client.StartWorkflowOptions,
) (SimpleClient_SomeWorkflow1, error) {
	// TODO(cretz): Could set ID if a workflow_id_field is present
	run, err := s.client.ExecuteWorkflow(ctx, opts, "mycompany.simple.SomeWorkflow1", req)
	if run == nil || err != nil {
		return nil, err
	}
	return simpleClient_SomeWorkflow1{s.client, run}, nil
}

type simpleClient_SomeWorkflow1 struct {
	client client.Client
	client.WorkflowRun
}

func (s simpleClient_SomeWorkflow1) GetResult(context.Context) (*SomeWorkflow1Response, error) {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeQuery1(context.Context) (*SomeQuery1Response, error) {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeQuery2(context.Context, *SomeQuery2Request) (*SomeQuery2Response, error) {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeSignal1(context.Context) error {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeSignal2(context.Context, *SomeSignal2Request) error {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeCall1(context.Context) (*SomeCall1Response, error) {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeCall2(context.Context) error {
	panic("TODO")
}

func (s simpleClient_SomeWorkflow1) SomeCall3(context.Context, *SomeCall3Request) (*SomeCall3Response, error) {
	panic("TODO")
}

type simpleWorker struct{}

func (simpleWorker) SomeWorkflow1(ctx workflow.Context, req *SomeWorkflow1Request) (*SomeWorkflow1Response, error) {
	panic("TODO")
}

func BuildSomeWorkflow1(impl SimpleWorker_SomeWorkflow1) interface{} {
	panic("TODO")
}

func RegisterSomeWorkflow1(r worker.WorkflowRegistry, impl SimpleWorker_SomeWorkflow1) {
	panic("TODO")
}

type Simple_SomeWorkflow1Input struct {
	Req       *SomeWorkflow1Request
	Signal1   Simple_SomeSignal1Channel
	Signal2   Simple_SomeSignal2Channel
	SomeCall1 Simple_SomeCall1Channel
	SomeCall2 Simple_SomeCall2Channel
}

type Simple_SomeSignal1Channel struct{ workflow.ReceiveChannel }

func (s Simple_SomeSignal1Channel) Recv(workflow.Context) (stillOpen bool) { panic("TODO") }
func (s Simple_SomeSignal1Channel) RecvAsync(workflow.Context) (received, stillOpen bool) {
	panic("TODO")
}

func (s Simple_SomeSignal1Channel) Select(workflow.Selector, func(stillOpen bool)) workflow.Selector {
	panic("TODO")
}

type Simple_SomeSignal2Channel struct{ workflow.ReceiveChannel }

func (s Simple_SomeSignal2Channel) Recv(workflow.Context) *SomeQuery2Request { panic("TODO") }
func (s Simple_SomeSignal2Channel) RecvAsync(workflow.Context) (req SomeQuery2Request, stillOpen bool) {
	panic("TODO")
}

func (s Simple_SomeSignal2Channel) Select(workflow.Selector, func(*SomeQuery2Request)) workflow.Selector {
	panic("TODO")
}

type SimpleWorker_SomeWorkflow1 interface {
	Run(workflow.Context, *Simple_SomeWorkflow1Input)
	SomeQuery1() (*SomeQuery1Response, error)
	SomeQuery2(*SomeQuery2Request) (*SomeQuery1Response, error)
}

type SimpleWorker_ActivitiesClient interface {
}

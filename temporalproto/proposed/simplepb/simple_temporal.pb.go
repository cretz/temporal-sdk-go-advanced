package simplepb

import (
	"context"
	"fmt"
	"reflect"

	"go.temporal.io/sdk/activity"
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
type SomeCall1Request struct {
	Id                string
	ResponseTaskQueue string
	// If not present, cannot send between workflows
	ResponseWorkflowID string
}
type SomeCall1Response struct {
	Id string
}
type SomeActivity1Request struct{}
type SomeActivity1Response struct{}

const (
	SomeWorkflow1Name     = "mycompany.simple.Simple.SomeWorkflow1"
	SomeActivity1Name     = "mycompany.simple.Simple.SomeActivity1"
	SomeQuery1Name        = "mycompany.simple.Simple.SomeQuery1"
	SomeQuery2Name        = "mycompany.simple.Simple.SomeQuery2"
	SomeSignal1Name       = "mycompany.simple.Simple.SomeSignal1"
	SomeSignal2Name       = "mycompany.simple.Simple.SomeSignal2"
	SomeCall1SignalName   = "mycompany.simple.Simple.SomeCall1"
	SomeCall1ResponseName = SomeCall1SignalName + "-response"
)

type WorkflowClient interface {
	ExecuteSomeWorkflow1(
		ctx context.Context,
		opts *client.StartWorkflowOptions,
		req *SomeWorkflow1Request,
	) (SomeWorkflow1Run, error)

	GetSomeWorkflow1(ctx context.Context, workflowID, runID string) (SomeWorkflow1Run, error)

	// Foo
	SomeQuery1(ctx context.Context, workflowID, runID string) (*SomeQuery1Response, error)
	SomeQuery2(ctx context.Context, workflowID, runID string, req *SomeQuery2Request) (*SomeQuery2Response, error)
	SomeSignal1(ctx context.Context, workflowID, runID string) error
	SomeSignal2(ctx context.Context, workflowID, runID string, req *SomeSignal2Request) error
	SomeCall1(ctx context.Context, workflowID, runID string, req *SomeCall1Request) (*SomeCall1Response, error)
}

type WorkflowClientOptions struct {
	// Required
	Client client.Client
	// Must be present for calls to succeed
	CallResponseHandler CallResponseHandler
}

type CallResponseHandler interface {
	TaskQueue() string
	PrepareCall(ctx context.Context) (id string, chOk <-chan interface{}, chErr <-chan error)
	// Does not error if already added with same params. Only errors if ID field
	// does not exist or the activity name is registered with different params.
	AddResponseType(activityName string, typ reflect.Type, idField string) error
}

type workflowClient struct {
	client              client.Client
	callResponseHandler CallResponseHandler
}

func NewWorkflowClient(opts WorkflowClientOptions) WorkflowClient {
	c := &workflowClient{
		client:              opts.Client,
		callResponseHandler: opts.CallResponseHandler,
	}
	// Add response types to handler
	if opts.CallResponseHandler != nil {
		err := opts.CallResponseHandler.AddResponseType(SomeCall1ResponseName, reflect.TypeOf((*SomeCall1Response)(nil)), "Id")
		if err != nil {
			panic(err)
		}
	}
	return c
}

func (w *workflowClient) ExecuteSomeWorkflow1(
	ctx context.Context,
	opts *client.StartWorkflowOptions,
	req *SomeWorkflow1Request,
) (SomeWorkflow1Run, error) {
	// TODO(cretz): Could set ID if a workflow_id_field is present
	if opts == nil {
		opts = &client.StartWorkflowOptions{}
	}
	run, err := w.client.ExecuteWorkflow(ctx, *opts, SomeWorkflow1Name, req)
	if run == nil || err != nil {
		return nil, err
	}
	return &someWorkflow1Run{w, run}, nil
}

func (w *workflowClient) GetSomeWorkflow1(ctx context.Context, workflowID, runID string) (SomeWorkflow1Run, error) {
	return &someWorkflow1Run{w, w.client.GetWorkflow(ctx, workflowID, runID)}, nil
}

func (w *workflowClient) SomeQuery1(ctx context.Context, workflowID, runID string) (*SomeQuery1Response, error) {
	var resp SomeQuery1Response
	if val, err := w.client.QueryWorkflow(ctx, workflowID, runID, SomeQuery1Name); err != nil {
		return nil, err
	} else if err = val.Get(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (w *workflowClient) SomeQuery2(ctx context.Context, workflowID, runID string, req *SomeQuery2Request) (*SomeQuery2Response, error) {
	var resp SomeQuery2Response
	if val, err := w.client.QueryWorkflow(ctx, runID, workflowID, SomeQuery2Name, req); err != nil {
		return nil, err
	} else if err = val.Get(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (w *workflowClient) SomeSignal1(ctx context.Context, workflowID, runID string) error {
	return w.client.SignalWorkflow(ctx, workflowID, runID, SomeSignal1Name, nil)
}

func (w *workflowClient) SomeSignal2(ctx context.Context, workflowID, runID string, req *SomeSignal2Request) error {
	return w.client.SignalWorkflow(ctx, workflowID, runID, SomeSignal2Name, req)
}

func (w *workflowClient) SomeCall1(ctx context.Context, workflowID, runID string, req *SomeCall1Request) (*SomeCall1Response, error) {
	if w.callResponseHandler == nil {
		return nil, fmt.Errorf("missing response handler")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, chOk, chErr := w.callResponseHandler.PrepareCall(ctx)
	req.Id = id
	req.ResponseTaskQueue = w.callResponseHandler.TaskQueue()
	if err := w.client.SignalWorkflow(ctx, workflowID, runID, SomeCall1SignalName, req); err != nil {
		return nil, err
	}
	select {
	case resp := <-chOk:
		return resp.(*SomeCall1Response), nil
	case err := <-chErr:
		return nil, err
	}
}

type SomeWorkflow1Run interface {
	ID() string
	RunID() string
	Get(context.Context) (*SomeWorkflow1Response, error)
	SomeQuery1(context.Context) (*SomeQuery1Response, error)
	SomeQuery2(context.Context, *SomeQuery2Request) (*SomeQuery2Response, error)
	SomeSignal1(context.Context) error
	SomeSignal2(context.Context, *SomeSignal2Request) error
	// ID and response task queue are automatically set
	SomeCall1(context.Context, *SomeCall1Request) (*SomeCall1Response, error)
}

type someWorkflow1Run struct {
	client *workflowClient
	run    client.WorkflowRun
}

func (r *someWorkflow1Run) ID() string {
	return r.run.GetID()
}

func (r *someWorkflow1Run) RunID() string {
	return r.run.GetRunID()
}

func (r *someWorkflow1Run) Get(ctx context.Context) (*SomeWorkflow1Response, error) {
	var resp SomeWorkflow1Response
	if err := r.run.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *someWorkflow1Run) SomeQuery1(ctx context.Context) (*SomeQuery1Response, error) {
	return r.client.SomeQuery1(ctx, r.ID(), "")
}

func (r *someWorkflow1Run) SomeQuery2(ctx context.Context, req *SomeQuery2Request) (*SomeQuery2Response, error) {
	return r.client.SomeQuery2(ctx, r.ID(), "", req)
}

func (r *someWorkflow1Run) SomeSignal1(ctx context.Context) error {
	return r.client.SomeSignal1(ctx, r.ID(), "")
}

func (r *someWorkflow1Run) SomeSignal2(ctx context.Context, req *SomeSignal2Request) error {
	return r.client.SomeSignal2(ctx, r.ID(), "", req)
}

func (r *someWorkflow1Run) SomeCall1(ctx context.Context, req *SomeCall1Request) (*SomeCall1Response, error) {
	return r.client.SomeCall1(ctx, r.ID(), "", req)
}

func (r *SomeCall1Request) ResponseInfo() (id, targetName, taskQueue string) {
	return r.Id, SomeCall1ResponseName, r.ResponseTaskQueue
}

type SomeWorkflow1Impl interface {
	Run(workflow.Context) (*SomeWorkflow1Response, error)
	SomeQuery1() (*SomeQuery1Response, error)
	SomeQuery2(*SomeQuery2Request) (*SomeQuery1Response, error)
}

type SomeWorkflow1Input struct {
	Req         *SomeWorkflow1Request
	SomeSignal1 SomeSignal1
	SomeSignal2 SomeSignal2
	SomeCall1   SomeCall1
}

type someWorkflowWorker struct {
	newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)
}

func (w someWorkflowWorker) SomeWorkflow1(ctx workflow.Context, req *SomeWorkflow1Request) (*SomeWorkflow1Response, error) {
	in := &SomeWorkflow1Input{Req: req}
	in.SomeSignal1.Channel = workflow.GetSignalChannel(ctx, SomeSignal1Name)
	in.SomeSignal2.Channel = workflow.GetSignalChannel(ctx, SomeSignal2Name)
	in.SomeCall1.Channel = workflow.GetSignalChannel(ctx, SomeCall1SignalName)
	impl, err := w.newImpl(ctx, in)
	if err != nil {
		return nil, err
	}
	if err := workflow.SetQueryHandler(ctx, SomeQuery1Name, impl.SomeQuery1); err != nil {
		return nil, err
	}
	if err := workflow.SetQueryHandler(ctx, SomeQuery2Name, impl.SomeQuery2); err != nil {
		return nil, err
	}
	return impl.Run(ctx)
}

func BuildSomeWorkflow1(newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)) func(ctx workflow.Context, req *SomeWorkflow1Request) (*SomeWorkflow1Response, error) {
	return someWorkflowWorker{newImpl}.SomeWorkflow1
}

func RegisterSomeWorkflow1(r worker.WorkflowRegistry, newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)) {
	r.RegisterWorkflowWithOptions(BuildSomeWorkflow1(newImpl), workflow.RegisterOptions{Name: SomeWorkflow1Name})
}

type SomeSignal1 struct{ Channel workflow.ReceiveChannel }

func (s SomeSignal1) Receive(ctx workflow.Context) {
	s.Channel.Receive(ctx, nil)
}

func (s SomeSignal1) ReceiveAsync() (received bool) {
	return s.Channel.ReceiveAsync(nil)
}

func (s SomeSignal1) Select(sel workflow.Selector, fn func()) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		s.ReceiveAsync()
		if fn != nil {
			fn()
		}
	})
}

type SomeSignal2 struct{ Channel workflow.ReceiveChannel }

func (s SomeSignal2) Receive(ctx workflow.Context) *SomeSignal2Request {
	var resp SomeSignal2Request
	s.Channel.Receive(ctx, &resp)
	return &resp
}

// Nil if not received
func (s SomeSignal2) ReceiveAsync() *SomeSignal2Request {
	var resp SomeSignal2Request
	if !s.Channel.ReceiveAsync(&resp) {
		return nil
	}
	return &resp
}

func (s SomeSignal2) Select(sel workflow.Selector, fn func(*SomeSignal2Request)) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		req := s.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

type SomeCall1 struct{ Channel workflow.ReceiveChannel }

func (s SomeCall1) Receive(ctx workflow.Context) *SomeCall1Request {
	var resp SomeCall1Request
	if !s.Channel.Receive(ctx, &resp) {
		return nil
	}
	return &resp
}

func (s SomeCall1) ReceiveAsync() (req *SomeCall1Request, stillOpen bool) {
	var resp SomeCall1Request
	if ok, more := s.Channel.ReceiveAsyncWithMoreFlag(&resp); !ok || !more {
		return nil, more
	}
	return &resp, true
}

func (s SomeCall1) Select(sel workflow.Selector, fn func(req *SomeCall1Request)) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		req, _ := s.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

func (s SomeCall1) Respond(ctx workflow.Context, req *SomeCall1Request, resp *SomeCall1Response) workflow.Future {
	id, targetName, taskQueue := req.ResponseInfo()
	resp.Id = id
	if req.ResponseWorkflowID != "" {
		return workflow.SignalExternalWorkflow(ctx, req.ResponseWorkflowID, "", targetName+"-"+req.Id, resp)
	}
	opts := workflow.GetActivityOptions(ctx)
	opts.TaskQueue = taskQueue
	ctx = workflow.WithActivityOptions(ctx, opts)
	return workflow.ExecuteActivity(ctx, targetName, resp)
}

// Opts taken from context if not given
func SomeWorkflow1Child(
	ctx workflow.Context,
	opts *workflow.ChildWorkflowOptions,
	req *SomeWorkflow1Request,
) SomeWorkflow1ChildRun {
	// TODO(cretz): Use workflow_id field if present
	if opts == nil {
		ctxOpts := workflow.GetChildWorkflowOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithChildOptions(ctx, *opts)
	return SomeWorkflow1ChildRun{workflow.ExecuteChildWorkflow(ctx, SomeWorkflow1Name, req)}
}

type SomeWorkflow1ChildRun struct{ Future workflow.ChildWorkflowFuture }

func (r SomeWorkflow1ChildRun) WaitStart(ctx workflow.Context) (*workflow.Execution, error) {
	var exec workflow.Execution
	if err := r.Future.GetChildWorkflowExecution().Get(ctx, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r SomeWorkflow1ChildRun) SelectStart(sel workflow.Selector, fn func(SomeWorkflow1ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future.GetChildWorkflowExecution(), func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

func (r SomeWorkflow1ChildRun) GetResult(ctx workflow.Context) (*SomeWorkflow1Response, error) {
	var resp SomeWorkflow1Response
	if err := r.Future.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r SomeWorkflow1ChildRun) Select(sel workflow.Selector, fn func(SomeWorkflow1ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future, func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

func (r SomeWorkflow1ChildRun) SomeSignal1(ctx workflow.Context) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal1Name, nil)
}

func (r SomeWorkflow1ChildRun) SomeSignal2(ctx workflow.Context, req *SomeSignal2Request) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal2Name, req)
}

// This is done via ephemeral signal based on request ID
func (r SomeWorkflow1ChildRun) SomeCall1(ctx workflow.Context, req *SomeCall1Request) (SomeCall1ResponseExternal, error) {
	// Require request ID
	var resp SomeCall1ResponseExternal
	if req.Id == "" {
		return resp, fmt.Errorf("missing request ID")
	} else if req.ResponseTaskQueue != "" {
		return resp, fmt.Errorf("cannot have task queue for child")
	}
	req.ResponseWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
	resp.Channel = workflow.GetSignalChannel(ctx, SomeCall1ResponseName+"-"+req.Id)
	resp.Future = r.Future.SignalChildWorkflow(ctx, SomeCall1SignalName, req)
	return resp, nil
}

func SomeSignal1External(ctx workflow.Context, workflowID, runID string) workflow.Future {
	return workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeSignal1Name, nil)
}

func SomeSignal2External(ctx workflow.Context, workflowID, runID string, req *SomeSignal2Request) workflow.Future {
	return workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeSignal2Name, req)
}

func SomeCall1External(ctx workflow.Context, workflowID, runID string, req *SomeCall1Request) (SomeCall1ResponseExternal, error) {
	// Require request ID
	var resp SomeCall1ResponseExternal
	if req.Id == "" {
		return resp, fmt.Errorf("missing request ID")
	} else if req.ResponseTaskQueue != "" {
		return resp, fmt.Errorf("cannot have task queue for external")
	}
	req.ResponseWorkflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
	resp.Channel = workflow.GetSignalChannel(ctx, SomeCall1ResponseName+"-"+req.Id)
	resp.Future = workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeCall1SignalName, req)
	return resp, nil
}

type SomeCall1ResponseExternal struct {
	Future  workflow.Future
	Channel workflow.ReceiveChannel
}

func (f SomeCall1ResponseExternal) WaitSent(ctx workflow.Context) error {
	return f.Future.Get(ctx, nil)
}

// Func can be nil
func (f SomeCall1ResponseExternal) SelectSent(sel workflow.Selector, fn func(SomeCall1ResponseExternal)) workflow.Selector {
	return sel.AddFuture(f.Future, func(workflow.Future) {
		if fn != nil {
			fn(f)
		}
	})
}

func (s *SomeCall1ResponseExternal) Receive(ctx workflow.Context) *SomeCall1Response {
	var resp SomeCall1Response
	if !s.Channel.Receive(ctx, &resp) {
		return nil
	}
	return &resp
}

func (s *SomeCall1ResponseExternal) ReceiveAsync() (resp *SomeCall1Response, stillOpen bool) {
	resp = &SomeCall1Response{}
	if ok, more := s.Channel.ReceiveAsyncWithMoreFlag(resp); !ok || !more {
		return nil, more
	}
	return resp, true
}

func (s *SomeCall1ResponseExternal) Select(sel workflow.Selector, fn func(req *SomeCall1Response)) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		req, _ := s.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

type ActivitiesImpl interface {
	SomeActivity1(context.Context, *SomeActivity1Request) (*SomeActivity1Response, error)
}

func RegisterActivities(r worker.ActivityRegistry, a ActivitiesImpl) {
	RegisterSomeActivity1(r, a.SomeActivity1)
}

func RegisterSomeActivity1(r worker.ActivityRegistry, impl func(context.Context, *SomeActivity1Request) (*SomeActivity1Response, error)) {
	r.RegisterActivityWithOptions(impl, activity.RegisterOptions{Name: SomeActivity1Name})
}

// Nil options uses ones in context
func SomeActivity1(ctx workflow.Context, opts *workflow.ActivityOptions, req *SomeActivity1Request) SomeActivity1Future {
	if opts == nil {
		ctxOpts := workflow.GetActivityOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithActivityOptions(ctx, *opts)
	return SomeActivity1Future{workflow.ExecuteActivity(ctx, SomeActivity1Name, req)}
}

// Nil options uses ones in context
func SomeActivity1Local(
	ctx workflow.Context,
	opts *workflow.LocalActivityOptions,
	fn func(context.Context, *SomeActivity1Request) (*SomeActivity1Response, error),
	req *SomeActivity1Request,
) SomeActivity1Future {
	if opts == nil {
		ctxOpts := workflow.GetLocalActivityOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithLocalActivityOptions(ctx, *opts)
	return SomeActivity1Future{workflow.ExecuteLocalActivity(ctx, fn, req)}
}

type SomeActivity1Future struct{ Future workflow.Future }

func (f SomeActivity1Future) Get(ctx workflow.Context) (*SomeActivity1Response, error) {
	var resp SomeActivity1Response
	if err := f.Future.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (f SomeActivity1Future) Select(sel workflow.Selector, fn func(SomeActivity1Future)) workflow.Selector {
	return sel.AddFuture(f.Future, func(workflow.Future) {
		if fn != nil {
			fn(f)
		}
	})
}

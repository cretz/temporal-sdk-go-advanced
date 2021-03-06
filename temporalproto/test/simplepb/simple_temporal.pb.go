// Code generated by protoc-gen-go_temporal. DO NOT EDIT.

package simplepb

import (
	context "context"
	fmt "fmt"
	activity "go.temporal.io/sdk/activity"
	client "go.temporal.io/sdk/client"
	worker "go.temporal.io/sdk/worker"
	workflow "go.temporal.io/sdk/workflow"
	reflect "reflect"
)

// Constants used as workflow, activity, query, and signal names.
const (
	SomeWorkflow1Name     = "mycompany.simple.Simple.SomeWorkflow1"
	SomeWorkflow2Name     = "mycompany.simple.Simple.SomeWorkflow2"
	SomeWorkflow3Name     = "mycompany.simple.Simple.SomeWorkflow3"
	SomeActivity1Name     = "mycompany.simple.Simple.SomeActivity1"
	SomeActivity2Name     = "mycompany.simple.Simple.SomeActivity2"
	SomeActivity3Name     = "mycompany.simple.Simple.SomeActivity3"
	SomeQuery1Name        = "mycompany.simple.Simple.SomeQuery1"
	SomeQuery2Name        = "mycompany.simple.Simple.SomeQuery2"
	SomeSignal1Name       = "mycompany.simple.Simple.SomeSignal1"
	SomeSignal2Name       = "mycompany.simple.Simple.SomeSignal2"
	SomeCall1SignalName   = "mycompany.simple.Simple.SomeCall1"
	SomeCall1ResponseName = SomeCall1SignalName + "-response"
)

type Client interface {

	// SomeWorkflow1 does some workflow thing.
	ExecuteSomeWorkflow1(ctx context.Context, opts *client.StartWorkflowOptions, req *SomeWorkflow1Request) (SomeWorkflow1Run, error)

	// GetSomeWorkflow1 returns an existing run started by ExecuteSomeWorkflow1.
	GetSomeWorkflow1(ctx context.Context, workflowID, runID string) (SomeWorkflow1Run, error)

	// SomeWorkflow2 does some workflow thing.
	ExecuteSomeWorkflow2(ctx context.Context, opts *client.StartWorkflowOptions, signalStart bool) (SomeWorkflow2Run, error)

	// GetSomeWorkflow2 returns an existing run started by ExecuteSomeWorkflow2.
	GetSomeWorkflow2(ctx context.Context, workflowID, runID string) (SomeWorkflow2Run, error)

	// SomeWorkflow3 does some workflow thing.
	ExecuteSomeWorkflow3(ctx context.Context, opts *client.StartWorkflowOptions, req *SomeWorkflow3Request, signalStart *SomeSignal2Request) (SomeWorkflow3Run, error)

	// GetSomeWorkflow3 returns an existing run started by ExecuteSomeWorkflow3.
	GetSomeWorkflow3(ctx context.Context, workflowID, runID string) (SomeWorkflow3Run, error)

	// SomeQuery1 queries some thing.
	SomeQuery1(ctx context.Context, workflowID, runID string) (*SomeQuery1Response, error)

	// SomeQuery2 queries some thing.
	SomeQuery2(ctx context.Context, workflowID, runID string, req *SomeQuery2Request) (*SomeQuery2Response, error)

	// SomeSignal1 is a signal.
	SomeSignal1(ctx context.Context, workflowID, runID string) error

	// SomeSignal2 is a signal.
	SomeSignal2(ctx context.Context, workflowID, runID string, req *SomeSignal2Request) error

	// SomeCall1 is a call.
	SomeCall1(ctx context.Context, workflowID, runID string, req *SomeCall1Request) (*SomeCall1Response, error)
}

// ClientOptions are used for NewClient.
type ClientOptions struct {
	// Required client.
	Client client.Client
	// Handler that must be present for client calls to succeed.
	CallResponseHandler CallResponseHandler
}

// CallResponseHandler handles activity responses.
type CallResponseHandler interface {
	// TaskQueue returns the task queue for response activities.
	TaskQueue() string

	// PrepareCall creates a new ID and channels to receive response/error.
	// Each channel only has a buffer of one and are never closed and only one is ever sent to.
	// If context is closed, the context error is returned on error channel.
	PrepareCall(ctx context.Context) (id string, chOk <-chan interface{}, chErr <-chan error)

	// AddResponseType adds an activity for the given type and ID field.
	// Does not error if activity name already exists for the same params.
	AddResponseType(activityName string, typ reflect.Type, idField string) error
}

type clientImpl struct {
	client              client.Client
	callResponseHandler CallResponseHandler
}

// NewClient creates a new Client.
func NewClient(opts ClientOptions) Client {
	if opts.Client == nil {
		panic("missing client")
	}
	c := &clientImpl{client: opts.Client, callResponseHandler: opts.CallResponseHandler}
	if opts.CallResponseHandler != nil {
		if err := opts.CallResponseHandler.AddResponseType(SomeCall1ResponseName, reflect.TypeOf((*SomeCall1Response)(nil)), "Id"); err != nil {
			panic(err)
		}
	}
	return c
}

func (c *clientImpl) ExecuteSomeWorkflow1(ctx context.Context, opts *client.StartWorkflowOptions, req *SomeWorkflow1Request) (SomeWorkflow1Run, error) {
	if opts == nil {
		opts = &client.StartWorkflowOptions{}
	}
	run, err := c.client.ExecuteWorkflow(ctx, *opts, SomeWorkflow1Name, req)
	if run == nil || err != nil {
		return nil, err
	}
	return &someWorkflow1Run{c, run}, nil
}

func (c *clientImpl) GetSomeWorkflow1(ctx context.Context, workflowID, runID string) (SomeWorkflow1Run, error) {
	return &someWorkflow1Run{c, c.client.GetWorkflow(ctx, workflowID, runID)}, nil
}

func (c *clientImpl) ExecuteSomeWorkflow2(ctx context.Context, opts *client.StartWorkflowOptions, signalStart bool) (SomeWorkflow2Run, error) {
	if opts == nil {
		opts = &client.StartWorkflowOptions{}
	}
	var run client.WorkflowRun
	var err error
	if signalStart {
		run, err = c.client.SignalWithStartWorkflow(ctx, opts.ID, SomeSignal1Name, nil, *opts, SomeWorkflow2Name)
	} else {
		run, err = c.client.ExecuteWorkflow(ctx, *opts, SomeWorkflow2Name)
	}
	if run == nil || err != nil {
		return nil, err
	}
	return &someWorkflow2Run{c, run}, nil
}

func (c *clientImpl) GetSomeWorkflow2(ctx context.Context, workflowID, runID string) (SomeWorkflow2Run, error) {
	return &someWorkflow2Run{c, c.client.GetWorkflow(ctx, workflowID, runID)}, nil
}

func (c *clientImpl) ExecuteSomeWorkflow3(ctx context.Context, opts *client.StartWorkflowOptions, req *SomeWorkflow3Request, signalStart *SomeSignal2Request) (SomeWorkflow3Run, error) {
	if opts == nil {
		opts = &client.StartWorkflowOptions{}
	}
	if opts.ID == "" && req.Id != "" {
		opts.ID = req.Id
	}
	if opts.TaskQueue == "" {
		opts.TaskQueue = "my-task-queue"
	}
	var run client.WorkflowRun
	var err error
	if signalStart != nil {
		run, err = c.client.SignalWithStartWorkflow(ctx, opts.ID, SomeSignal2Name, signalStart, *opts, SomeWorkflow3Name, req)
	} else {
		run, err = c.client.ExecuteWorkflow(ctx, *opts, SomeWorkflow3Name, req)
	}
	if run == nil || err != nil {
		return nil, err
	}
	return &someWorkflow3Run{c, run}, nil
}

func (c *clientImpl) GetSomeWorkflow3(ctx context.Context, workflowID, runID string) (SomeWorkflow3Run, error) {
	return &someWorkflow3Run{c, c.client.GetWorkflow(ctx, workflowID, runID)}, nil
}

func (c *clientImpl) SomeQuery1(ctx context.Context, workflowID, runID string) (*SomeQuery1Response, error) {
	var resp SomeQuery1Response
	if val, err := c.client.QueryWorkflow(ctx, workflowID, runID, SomeQuery1Name); err != nil {
		return nil, err
	} else if err = val.Get(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *clientImpl) SomeQuery2(ctx context.Context, workflowID, runID string, req *SomeQuery2Request) (*SomeQuery2Response, error) {
	var resp SomeQuery2Response
	if val, err := c.client.QueryWorkflow(ctx, workflowID, runID, SomeQuery2Name, req); err != nil {
		return nil, err
	} else if err = val.Get(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *clientImpl) SomeSignal1(ctx context.Context, workflowID, runID string) error {
	return c.client.SignalWorkflow(ctx, workflowID, runID, SomeSignal1Name, nil)
}

func (c *clientImpl) SomeSignal2(ctx context.Context, workflowID, runID string, req *SomeSignal2Request) error {
	return c.client.SignalWorkflow(ctx, workflowID, runID, SomeSignal2Name, req)
}

func (c *clientImpl) SomeCall1(ctx context.Context, workflowID, runID string, req *SomeCall1Request) (*SomeCall1Response, error) {
	if c.callResponseHandler == nil {
		return nil, fmt.Errorf("missing response handler")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, chOk, chErr := c.callResponseHandler.PrepareCall(ctx)
	req.Id = id
	req.ResponseTaskQueue = c.callResponseHandler.TaskQueue()
	if err := c.client.SignalWorkflow(ctx, workflowID, runID, SomeCall1SignalName, req); err != nil {
		return nil, err
	}
	select {
	case resp := <-chOk:
		return resp.(*SomeCall1Response), nil
	case err := <-chErr:
		return nil, err
	}
}

// SomeWorkflow1Run represents an execution of SomeWorkflow1.
type SomeWorkflow1Run interface {
	// ID is the workflow ID.
	ID() string

	// RunID is the workflow run ID.
	RunID() string

	// Get returns the completed workflow value, waiting if necessary.
	Get(ctx context.Context) (*SomeWorkflow1Response, error)

	// SomeQuery1 queries some thing.
	SomeQuery1(ctx context.Context) (*SomeQuery1Response, error)

	// SomeQuery2 queries some thing.
	SomeQuery2(ctx context.Context, req *SomeQuery2Request) (*SomeQuery2Response, error)

	// SomeSignal1 is a signal.
	SomeSignal1(ctx context.Context) error

	// SomeSignal2 is a signal.
	SomeSignal2(ctx context.Context, req *SomeSignal2Request) error

	// SomeCall1 is a call.
	SomeCall1(ctx context.Context, req *SomeCall1Request) (*SomeCall1Response, error)
}

type someWorkflow1Run struct {
	client *clientImpl
	run    client.WorkflowRun
}

func (r *someWorkflow1Run) ID() string { return r.run.GetID() }

func (r *someWorkflow1Run) RunID() string { return r.run.GetRunID() }

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

// SomeWorkflow2Run represents an execution of SomeWorkflow2.
type SomeWorkflow2Run interface {
	// ID is the workflow ID.
	ID() string

	// RunID is the workflow run ID.
	RunID() string

	// Get returns the completed workflow value, waiting if necessary.
	Get(ctx context.Context) error

	// SomeSignal1 is a signal.
	SomeSignal1(ctx context.Context) error
}

type someWorkflow2Run struct {
	client *clientImpl
	run    client.WorkflowRun
}

func (r *someWorkflow2Run) ID() string { return r.run.GetID() }

func (r *someWorkflow2Run) RunID() string { return r.run.GetRunID() }

func (r *someWorkflow2Run) Get(ctx context.Context) error {
	return r.run.Get(ctx, nil)
}

func (r *someWorkflow2Run) SomeSignal1(ctx context.Context) error {
	return r.client.SomeSignal1(ctx, r.ID(), "")
}

// SomeWorkflow3Run represents an execution of SomeWorkflow3.
type SomeWorkflow3Run interface {
	// ID is the workflow ID.
	ID() string

	// RunID is the workflow run ID.
	RunID() string

	// Get returns the completed workflow value, waiting if necessary.
	Get(ctx context.Context) error

	// SomeSignal2 is a signal.
	SomeSignal2(ctx context.Context, req *SomeSignal2Request) error
}

type someWorkflow3Run struct {
	client *clientImpl
	run    client.WorkflowRun
}

func (r *someWorkflow3Run) ID() string { return r.run.GetID() }

func (r *someWorkflow3Run) RunID() string { return r.run.GetRunID() }

func (r *someWorkflow3Run) Get(ctx context.Context) error {
	return r.run.Get(ctx, nil)
}

func (r *someWorkflow3Run) SomeSignal2(ctx context.Context, req *SomeSignal2Request) error {
	return r.client.SomeSignal2(ctx, r.ID(), "", req)
}

// SomeWorkflow1 does some workflow thing.
type SomeWorkflow1Impl interface {
	Run(workflow.Context) (*SomeWorkflow1Response, error)

	SomeQuery1() (*SomeQuery1Response, error)

	SomeQuery2(*SomeQuery2Request) (*SomeQuery2Response, error)
}

// SomeWorkflow1Input is input provided to SomeWorkflow1Impl.Run.
type SomeWorkflow1Input struct {
	Req         *SomeWorkflow1Request
	SomeSignal1 SomeSignal1
	SomeSignal2 SomeSignal2
	SomeCall1   SomeCall1
}

type someWorkflow1Worker struct {
	newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)
}

func (w someWorkflow1Worker) SomeWorkflow1(ctx workflow.Context, req *SomeWorkflow1Request) (*SomeWorkflow1Response, error) {
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

// BuildSomeWorkflow1 returns a function for the given impl.
func BuildSomeWorkflow1(newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)) func(ctx workflow.Context, req *SomeWorkflow1Request) (*SomeWorkflow1Response, error) {
	return someWorkflow1Worker{newImpl}.SomeWorkflow1
}

// RegisterSomeWorkflow1 registers a workflow with the given impl.
func RegisterSomeWorkflow1(r worker.WorkflowRegistry, newImpl func(workflow.Context, *SomeWorkflow1Input) (SomeWorkflow1Impl, error)) {
	r.RegisterWorkflowWithOptions(BuildSomeWorkflow1(newImpl), workflow.RegisterOptions{Name: SomeWorkflow1Name})
}

// SomeWorkflow2 does some workflow thing.
type SomeWorkflow2Impl interface {
	Run(workflow.Context) error
}

// SomeWorkflow2Input is input provided to SomeWorkflow2Impl.Run.
type SomeWorkflow2Input struct {
	SomeSignal1 SomeSignal1
}

type someWorkflow2Worker struct {
	newImpl func(workflow.Context, *SomeWorkflow2Input) (SomeWorkflow2Impl, error)
}

func (w someWorkflow2Worker) SomeWorkflow2(ctx workflow.Context) error {
	in := &SomeWorkflow2Input{}
	in.SomeSignal1.Channel = workflow.GetSignalChannel(ctx, SomeSignal1Name)
	impl, err := w.newImpl(ctx, in)
	if err != nil {
		return err
	}
	return impl.Run(ctx)
}

// BuildSomeWorkflow2 returns a function for the given impl.
func BuildSomeWorkflow2(newImpl func(workflow.Context, *SomeWorkflow2Input) (SomeWorkflow2Impl, error)) func(ctx workflow.Context) error {
	return someWorkflow2Worker{newImpl}.SomeWorkflow2
}

// RegisterSomeWorkflow2 registers a workflow with the given impl.
func RegisterSomeWorkflow2(r worker.WorkflowRegistry, newImpl func(workflow.Context, *SomeWorkflow2Input) (SomeWorkflow2Impl, error)) {
	r.RegisterWorkflowWithOptions(BuildSomeWorkflow2(newImpl), workflow.RegisterOptions{Name: SomeWorkflow2Name})
}

// SomeWorkflow3 does some workflow thing.
type SomeWorkflow3Impl interface {
	Run(workflow.Context) error
}

// SomeWorkflow3Input is input provided to SomeWorkflow3Impl.Run.
type SomeWorkflow3Input struct {
	Req         *SomeWorkflow3Request
	SomeSignal2 SomeSignal2
}

type someWorkflow3Worker struct {
	newImpl func(workflow.Context, *SomeWorkflow3Input) (SomeWorkflow3Impl, error)
}

func (w someWorkflow3Worker) SomeWorkflow3(ctx workflow.Context, req *SomeWorkflow3Request) error {
	in := &SomeWorkflow3Input{Req: req}
	in.SomeSignal2.Channel = workflow.GetSignalChannel(ctx, SomeSignal2Name)
	impl, err := w.newImpl(ctx, in)
	if err != nil {
		return err
	}
	return impl.Run(ctx)
}

// BuildSomeWorkflow3 returns a function for the given impl.
func BuildSomeWorkflow3(newImpl func(workflow.Context, *SomeWorkflow3Input) (SomeWorkflow3Impl, error)) func(ctx workflow.Context, req *SomeWorkflow3Request) error {
	return someWorkflow3Worker{newImpl}.SomeWorkflow3
}

// RegisterSomeWorkflow3 registers a workflow with the given impl.
func RegisterSomeWorkflow3(r worker.WorkflowRegistry, newImpl func(workflow.Context, *SomeWorkflow3Input) (SomeWorkflow3Impl, error)) {
	r.RegisterWorkflowWithOptions(BuildSomeWorkflow3(newImpl), workflow.RegisterOptions{Name: SomeWorkflow3Name})
}

// SomeSignal1 is a signal.
type SomeSignal1 struct{ Channel workflow.ReceiveChannel }

// Receive blocks until signal is received.
func (s SomeSignal1) Receive(ctx workflow.Context) {
	s.Channel.Receive(ctx, nil)
}

// ReceiveAsync returns true if signal received or false if not.
func (s SomeSignal1) ReceiveAsync() (received bool) {
	return s.Channel.ReceiveAsync(nil)
}

// Select adds the callback to the selector to be invoked when signal received. Callback can be nil.
func (s SomeSignal1) Select(sel workflow.Selector, fn func()) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		s.ReceiveAsync()
		if fn != nil {
			fn()
		}
	})
}

// SomeSignal2 is a signal.
type SomeSignal2 struct{ Channel workflow.ReceiveChannel }

// Receive blocks until signal is received.
func (s SomeSignal2) Receive(ctx workflow.Context) *SomeSignal2Request {
	var resp SomeSignal2Request
	s.Channel.Receive(ctx, &resp)
	return &resp
}

// ReceiveAsync returns received signal or nil if none.
func (s SomeSignal2) ReceiveAsync() *SomeSignal2Request {
	var resp SomeSignal2Request
	if !s.Channel.ReceiveAsync(&resp) {
		return nil
	}
	return &resp
}

// Select adds the callback to the selector to be invoked when signal received. Callback can be nil.
func (s SomeSignal2) Select(sel workflow.Selector, fn func(*SomeSignal2Request)) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		req := s.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

// SomeCall1 is a call.
type SomeCall1 struct{ Channel workflow.ReceiveChannel }

// Receive blocks until call is received.
func (s SomeCall1) Receive(ctx workflow.Context) *SomeCall1Request {
	var resp SomeCall1Request
	s.Channel.Receive(ctx, &resp)
	return &resp
}

// ReceiveAsync returns received signal or nil if none.
func (s SomeCall1) ReceiveAsync() *SomeCall1Request {
	var resp SomeCall1Request
	if !s.Channel.ReceiveAsync(&resp) {
		return nil
	}
	return &resp
}

// Select adds the callback to the selector to be invoked when signal received. Callback can be nil
func (s SomeCall1) Select(sel workflow.Selector, fn func(*SomeCall1Request)) workflow.Selector {
	return sel.AddReceive(s.Channel, func(workflow.ReceiveChannel, bool) {
		req := s.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

// Respond sends a response. Activity options not used if request received via
// another workflow. If activity options needed and not present, they are taken
// from the context.
func (s SomeCall1) Respond(ctx workflow.Context, opts *workflow.ActivityOptions, req *SomeCall1Request, resp *SomeCall1Response) workflow.Future {
	resp.Id = req.Id
	if req.ResponseWorkflowId != "" {
		return workflow.SignalExternalWorkflow(ctx, req.ResponseWorkflowId, "", SomeCall1ResponseName+"-"+req.Id, resp)
	}
	newOpts := workflow.GetActivityOptions(ctx)
	if opts != nil {
		newOpts = *opts
	}
	newOpts.TaskQueue = req.ResponseTaskQueue
	ctx = workflow.WithActivityOptions(ctx, newOpts)
	return workflow.ExecuteActivity(ctx, SomeCall1ResponseName, resp)
}

// SomeWorkflow1Child executes a child workflow.
// If options not present, they are taken from the context.
func SomeWorkflow1Child(ctx workflow.Context, opts *workflow.ChildWorkflowOptions, req *SomeWorkflow1Request) SomeWorkflow1ChildRun {
	if opts == nil {
		ctxOpts := workflow.GetChildWorkflowOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithChildOptions(ctx, *opts)
	return SomeWorkflow1ChildRun{workflow.ExecuteChildWorkflow(ctx, SomeWorkflow1Name, req)}
}

// SomeWorkflow1ChildRun is a future for the child workflow.
type SomeWorkflow1ChildRun struct{ Future workflow.ChildWorkflowFuture }

// WaitStart waits for the child workflow to start.
func (r SomeWorkflow1ChildRun) WaitStart(ctx workflow.Context) (*workflow.Execution, error) {
	var exec workflow.Execution
	if err := r.Future.GetChildWorkflowExecution().Get(ctx, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}

// SelectStart adds waiting for start to the selector. Callback can be nil.
func (r SomeWorkflow1ChildRun) SelectStart(sel workflow.Selector, fn func(SomeWorkflow1ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future.GetChildWorkflowExecution(), func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// Get returns the completed workflow value, waiting if necessary.
func (r SomeWorkflow1ChildRun) Get(ctx workflow.Context) (*SomeWorkflow1Response, error) {
	var resp SomeWorkflow1Response
	if err := r.Future.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Select adds this completion to the selector. Callback can be nil.
func (r SomeWorkflow1ChildRun) Select(sel workflow.Selector, fn func(SomeWorkflow1ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future, func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// SomeSignal1 is a signal.
func (r SomeWorkflow1ChildRun) SomeSignal1(ctx workflow.Context) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal1Name, nil)
}

// SomeSignal2 is a signal.
func (r SomeWorkflow1ChildRun) SomeSignal2(ctx workflow.Context, req *SomeSignal2Request) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal2Name, req)
}

// SomeCall1 is a call.
func (r SomeWorkflow1ChildRun) SomeCall1(ctx workflow.Context, req *SomeCall1Request) (SomeCall1ResponseExternal, error) {
	var resp SomeCall1ResponseExternal
	if req.Id == "" {
		return resp, fmt.Errorf("missing request ID")
	}
	if req.ResponseTaskQueue != "" {
		return resp, fmt.Errorf("cannot have task queue for child")
	}
	req.ResponseWorkflowId = workflow.GetInfo(ctx).WorkflowExecution.ID
	resp.Channel = workflow.GetSignalChannel(ctx, SomeCall1ResponseName+"-"+req.Id)
	resp.Future = r.Future.SignalChildWorkflow(ctx, SomeCall1SignalName, req)
	return resp, nil
}

// SomeWorkflow2Child executes a child workflow.
// If options not present, they are taken from the context.
func SomeWorkflow2Child(ctx workflow.Context, opts *workflow.ChildWorkflowOptions) SomeWorkflow2ChildRun {
	if opts == nil {
		ctxOpts := workflow.GetChildWorkflowOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithChildOptions(ctx, *opts)
	return SomeWorkflow2ChildRun{workflow.ExecuteChildWorkflow(ctx, SomeWorkflow2Name)}
}

// SomeWorkflow2ChildRun is a future for the child workflow.
type SomeWorkflow2ChildRun struct{ Future workflow.ChildWorkflowFuture }

// WaitStart waits for the child workflow to start.
func (r SomeWorkflow2ChildRun) WaitStart(ctx workflow.Context) (*workflow.Execution, error) {
	var exec workflow.Execution
	if err := r.Future.GetChildWorkflowExecution().Get(ctx, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}

// SelectStart adds waiting for start to the selector. Callback can be nil.
func (r SomeWorkflow2ChildRun) SelectStart(sel workflow.Selector, fn func(SomeWorkflow2ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future.GetChildWorkflowExecution(), func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// Get returns the completed workflow value, waiting if necessary.
func (r SomeWorkflow2ChildRun) Get(ctx workflow.Context) error {
	return r.Future.Get(ctx, nil)
}

// Select adds this completion to the selector. Callback can be nil.
func (r SomeWorkflow2ChildRun) Select(sel workflow.Selector, fn func(SomeWorkflow2ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future, func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// SomeSignal1 is a signal.
func (r SomeWorkflow2ChildRun) SomeSignal1(ctx workflow.Context) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal1Name, nil)
}

// SomeWorkflow3Child executes a child workflow.
// If options not present, they are taken from the context.
func SomeWorkflow3Child(ctx workflow.Context, opts *workflow.ChildWorkflowOptions, req *SomeWorkflow3Request) SomeWorkflow3ChildRun {
	if opts == nil {
		ctxOpts := workflow.GetChildWorkflowOptions(ctx)
		opts = &ctxOpts
	}
	if opts.WorkflowID == "" && req.Id != "" {
		opts.WorkflowID = req.Id
	}
	ctx = workflow.WithChildOptions(ctx, *opts)
	return SomeWorkflow3ChildRun{workflow.ExecuteChildWorkflow(ctx, SomeWorkflow3Name, req)}
}

// SomeWorkflow3ChildRun is a future for the child workflow.
type SomeWorkflow3ChildRun struct{ Future workflow.ChildWorkflowFuture }

// WaitStart waits for the child workflow to start.
func (r SomeWorkflow3ChildRun) WaitStart(ctx workflow.Context) (*workflow.Execution, error) {
	var exec workflow.Execution
	if err := r.Future.GetChildWorkflowExecution().Get(ctx, &exec); err != nil {
		return nil, err
	}
	return &exec, nil
}

// SelectStart adds waiting for start to the selector. Callback can be nil.
func (r SomeWorkflow3ChildRun) SelectStart(sel workflow.Selector, fn func(SomeWorkflow3ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future.GetChildWorkflowExecution(), func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// Get returns the completed workflow value, waiting if necessary.
func (r SomeWorkflow3ChildRun) Get(ctx workflow.Context) error {
	return r.Future.Get(ctx, nil)
}

// Select adds this completion to the selector. Callback can be nil.
func (r SomeWorkflow3ChildRun) Select(sel workflow.Selector, fn func(SomeWorkflow3ChildRun)) workflow.Selector {
	return sel.AddFuture(r.Future, func(workflow.Future) {
		if fn != nil {
			fn(r)
		}
	})
}

// SomeSignal2 is a signal.
func (r SomeWorkflow3ChildRun) SomeSignal2(ctx workflow.Context, req *SomeSignal2Request) workflow.Future {
	return r.Future.SignalChildWorkflow(ctx, SomeSignal2Name, req)
}

// SomeSignal1 is a signal.
func SomeSignal1External(ctx workflow.Context, workflowID, runID string) workflow.Future {
	return workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeSignal1Name, nil)
}

// SomeSignal2 is a signal.
func SomeSignal2External(ctx workflow.Context, workflowID, runID string, req *SomeSignal2Request) workflow.Future {
	return workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeSignal2Name, req)
}

// SomeCall1 is a call.
func SomeCall1External(ctx workflow.Context, workflowID, runID string, req *SomeCall1Request) (SomeCall1ResponseExternal, error) {
	var resp SomeCall1ResponseExternal
	if req.Id == "" {
		return resp, fmt.Errorf("missing request ID")
	}
	if req.ResponseTaskQueue != "" {
		return resp, fmt.Errorf("cannot have task queue for child")
	}
	req.ResponseWorkflowId = workflow.GetInfo(ctx).WorkflowExecution.ID
	resp.Channel = workflow.GetSignalChannel(ctx, SomeCall1ResponseName+"-"+req.Id)
	resp.Future = workflow.SignalExternalWorkflow(ctx, workflowID, runID, SomeCall1SignalName, req)
	return resp, nil
}

// SomeCall1ResponseExternal represents a call response.
type SomeCall1ResponseExternal struct {
	Future  workflow.Future
	Channel workflow.ReceiveChannel
}

// WaitSent blocks until the request is sent.
func (e SomeCall1ResponseExternal) WaitSent(ctx workflow.Context) error {
	return e.Future.Get(ctx, nil)
}

// SelectSent adds when a request is sent to the selector. Callback can be nil.
func (e SomeCall1ResponseExternal) SelectSent(sel workflow.Selector, fn func(SomeCall1ResponseExternal)) workflow.Selector {
	return sel.AddFuture(e.Future, func(workflow.Future) {
		if fn != nil {
			fn(e)
		}
	})
}

// Receive blocks until response is received.
func (e SomeCall1ResponseExternal) Receive(ctx workflow.Context) *SomeCall1Response {
	var resp SomeCall1Response
	e.Channel.Receive(ctx, &resp)
	return &resp
}

// ReceiveAsync returns response or nil if none.
func (e SomeCall1ResponseExternal) ReceiveAsync() *SomeCall1Response {
	var resp SomeCall1Response
	if !e.Channel.ReceiveAsync(&resp) {
		return nil
	}
	return &resp
}

// Select adds the callback to the selector to be invoked when response received. Callback can be nil
func (e SomeCall1ResponseExternal) Select(sel workflow.Selector, fn func(*SomeCall1Response)) workflow.Selector {
	return sel.AddReceive(e.Channel, func(workflow.ReceiveChannel, bool) {
		req := e.ReceiveAsync()
		if fn != nil {
			fn(req)
		}
	})
}

// ActivitiesImpl is an interface for activity implementations.
type ActivitiesImpl interface {

	// SomeActivity1 does some activity thing.
	SomeActivity1(context.Context) error

	// SomeActivity2 does some activity thing.
	SomeActivity2(context.Context, *SomeActivity2Request) error

	// SomeActivity3 does some activity thing.
	SomeActivity3(context.Context, *SomeActivity3Request) (*SomeActivity3Response, error)
}

// RegisterActivities registers all activities in the interface.
func RegisterActivities(r worker.ActivityRegistry, a ActivitiesImpl) {
	RegisterSomeActivity1(r, a.SomeActivity1)
	RegisterSomeActivity2(r, a.SomeActivity2)
	RegisterSomeActivity3(r, a.SomeActivity3)
}

// RegisterSomeActivity1 registers the single activity.
func RegisterSomeActivity1(r worker.ActivityRegistry, impl func(context.Context) error) {
	r.RegisterActivityWithOptions(impl, activity.RegisterOptions{Name: SomeActivity1Name})
}

// SomeActivity1 does some activity thing.
func SomeActivity1(ctx workflow.Context, opts *workflow.ActivityOptions) SomeActivity1Future {
	if opts == nil {
		ctxOpts := workflow.GetActivityOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithActivityOptions(ctx, *opts)
	return SomeActivity1Future{workflow.ExecuteActivity(ctx, SomeActivity1Name)}
}

// SomeActivity1 does some activity thing.
func SomeActivity1Local(ctx workflow.Context, opts *workflow.LocalActivityOptions, fn func(context.Context) error) SomeActivity1Future {
	if opts == nil {
		ctxOpts := workflow.GetLocalActivityOptions(ctx)
		opts = &ctxOpts
	}
	ctx = workflow.WithLocalActivityOptions(ctx, *opts)
	return SomeActivity1Future{workflow.ExecuteLocalActivity(ctx, fn)}
}

// SomeActivity1Future represents completion of the activity.
type SomeActivity1Future struct{ Future workflow.Future }

// Get waits for completion.
func (f SomeActivity1Future) Get(ctx workflow.Context) error {
	return f.Future.Get(ctx, nil)
}

// Select adds the completion to the selector. Callback can be nil.
func (f SomeActivity1Future) Select(sel workflow.Selector, fn func(SomeActivity1Future)) workflow.Selector {
	return sel.AddFuture(f.Future, func(workflow.Future) {
		if fn != nil {
			fn(f)
		}
	})
}

// RegisterSomeActivity2 registers the single activity.
func RegisterSomeActivity2(r worker.ActivityRegistry, impl func(context.Context, *SomeActivity2Request) error) {
	r.RegisterActivityWithOptions(impl, activity.RegisterOptions{Name: SomeActivity2Name})
}

// SomeActivity2 does some activity thing.
func SomeActivity2(ctx workflow.Context, opts *workflow.ActivityOptions, req *SomeActivity2Request) SomeActivity2Future {
	if opts == nil {
		ctxOpts := workflow.GetActivityOptions(ctx)
		opts = &ctxOpts
	}
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = 10000000000 // 10s
	}
	ctx = workflow.WithActivityOptions(ctx, *opts)
	return SomeActivity2Future{workflow.ExecuteActivity(ctx, SomeActivity2Name, req)}
}

// SomeActivity2 does some activity thing.
func SomeActivity2Local(ctx workflow.Context, opts *workflow.LocalActivityOptions, fn func(context.Context, *SomeActivity2Request) error, req *SomeActivity2Request) SomeActivity2Future {
	if opts == nil {
		ctxOpts := workflow.GetLocalActivityOptions(ctx)
		opts = &ctxOpts
	}
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = 10000000000 // 10s
	}
	ctx = workflow.WithLocalActivityOptions(ctx, *opts)
	return SomeActivity2Future{workflow.ExecuteLocalActivity(ctx, fn, req)}
}

// SomeActivity2Future represents completion of the activity.
type SomeActivity2Future struct{ Future workflow.Future }

// Get waits for completion.
func (f SomeActivity2Future) Get(ctx workflow.Context) error {
	return f.Future.Get(ctx, nil)
}

// Select adds the completion to the selector. Callback can be nil.
func (f SomeActivity2Future) Select(sel workflow.Selector, fn func(SomeActivity2Future)) workflow.Selector {
	return sel.AddFuture(f.Future, func(workflow.Future) {
		if fn != nil {
			fn(f)
		}
	})
}

// RegisterSomeActivity3 registers the single activity.
func RegisterSomeActivity3(r worker.ActivityRegistry, impl func(context.Context, *SomeActivity3Request) (*SomeActivity3Response, error)) {
	r.RegisterActivityWithOptions(impl, activity.RegisterOptions{Name: SomeActivity3Name})
}

// SomeActivity3 does some activity thing.
func SomeActivity3(ctx workflow.Context, opts *workflow.ActivityOptions, req *SomeActivity3Request) SomeActivity3Future {
	if opts == nil {
		ctxOpts := workflow.GetActivityOptions(ctx)
		opts = &ctxOpts
	}
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = 10000000000 // 10s
	}
	ctx = workflow.WithActivityOptions(ctx, *opts)
	return SomeActivity3Future{workflow.ExecuteActivity(ctx, SomeActivity3Name, req)}
}

// SomeActivity3 does some activity thing.
func SomeActivity3Local(ctx workflow.Context, opts *workflow.LocalActivityOptions, fn func(context.Context, *SomeActivity3Request) (*SomeActivity3Response, error), req *SomeActivity3Request) SomeActivity3Future {
	if opts == nil {
		ctxOpts := workflow.GetLocalActivityOptions(ctx)
		opts = &ctxOpts
	}
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = 10000000000 // 10s
	}
	ctx = workflow.WithLocalActivityOptions(ctx, *opts)
	return SomeActivity3Future{workflow.ExecuteLocalActivity(ctx, fn, req)}
}

// SomeActivity3Future represents completion of the activity.
type SomeActivity3Future struct{ Future workflow.Future }

// Get waits for completion.
func (f SomeActivity3Future) Get(ctx workflow.Context) (*SomeActivity3Response, error) {
	var resp SomeActivity3Response
	if err := f.Future.Get(ctx, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Select adds the completion to the selector. Callback can be nil.
func (f SomeActivity3Future) Select(sel workflow.Selector, fn func(SomeActivity3Future)) workflow.Selector {
	return sel.AddFuture(f.Future, func(workflow.Future) {
		if fn != nil {
			fn(f)
		}
	})
}

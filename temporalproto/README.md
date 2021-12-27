# Temporal Protobuf Generator

This tool generates code for calling and hosting workflows and activities based on annotated
[Protobuf RPC services](https://developers.google.com/protocol-buffers/docs/proto3#services). See
[simple.proto](test/simplepb/simple.proto) for an example proto and
[simple_temporal.pb.go](test/simplepb/simple_temporal.pb.go) for its generated code.

## Usage

### Generating Code

This is similar to the [gRPC quick start](https://grpc.io/docs/languages/go/quickstart/).

This requires [Go](https://golang.org/) installed, and [protoc](https://developers.google.com/protocol-buffers)
installed and on the `PATH`.

If not already installed, the protobuf Go code generator should be installed with:

    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

Similarly, this code generator should also be installed with:

    go install github.com/cretz/temporal-sdk-go-advanced/temporalproto/cmd/protoc-gen-go_temporal@latest

The Go `bin` also needs to be on the `PATH` for those two protoc-gen binaries to be seen by `protoc`. For example, on
Linux:

    export PATH="$PATH:$(go env GOPATH)/bin"

Now `protoc` can be used with both `--go_out` and `--go_temporal_out` and Temporal code will be written. For example,
to generate the code in [test/simplepb/simple.proto](test/simplepb/simple.proto) from this directory, run:

    protoc --go_out=paths=source_relative:. --go_temporal_out=paths=source_relative:. ./test/simplepb/simple.proto

If executing from another directory, a `-I` may need to be set to the absolute path of this directory so the
`import "temporalpb/sdk.proto"` resolves properly.

### Using Client-Side Generated Code

Generated code contains a `NewClient` call that accepts a `ClientOptions`. The client options has a required 
[Client](https://pkg.go.dev/go.temporal.io/sdk/client#Client) and an optional `CallResponseHandler`. The latter must be
set to make any request/response calls since it receives the response activity on its worker. A common implementation is
at `github.com/cretz/temporal-sdk-go-advanced/temporalutil/clientutil`.

Below is an example to start a workflow, make a call, and then wait for the workflow to complete:

```go
// Create client with call responder
callResponseHandler, err := clientutil.NewCallResponseHandler(clientutil.CallResponseHandlerOptions{
  TaskQueue: "my-client-task-queue",
  Worker:    myClientSideWorker,
})
if err != nil {
  panic(err)
}
c := genpkg.NewClient(genpkg.ClientOptions{
  Client: myclient,
  CallResponseHandler: callResponseHandler,
})

// Start a workflow
run, err := c.ExecuteMyWorkflow(
  ctx,
  &client.StartWorkflowOptions{TaskQueue: "my-workflow-task-queue"},
  &genpkg.MyWorkflowRequest{},
)
if err != nil {
  panic(err)
}

// Make call
callResp, err := run.MyCall(ctx, &genpkg.MyCallRequest{})
if err != nil {
  panic(err)
}
log.Printf("Call response: %v", callResp)

// Wait for workflow to complete
workflowResp, err := run.Get(ctx)
if err != nil {
  panic(err)
}
log.Printf("Workflow response: %v", workflowResp)
```

#### Using Worker-Side Generated Code

Generated code contains interfaces for workflows and activities. A registration function is made for workflows that
accepts a factory function for creating instances of the workflow interface. Similarly a registration function is made
for each individual activity function or the collection of them as an interface.

For example, to implement a workflow:

```go
type myWorkflowImpl struct { *genpkg.MyWorkflowInput }

func newMyWorkflowImpl(ctx workflow.Context, in *genpkg.MyWorkflowInput) (genpkg.MyWorkflowImpl, error) {
  return &myWorkflowImpl{in}, nil
}

func (m *myWorkflowImpl) Run(ctx workflow.Context) (*genpkg.MyWorkflowResponse, error) {
  // Do stuff
}
```

And to register it:

```go
genpkg.RegisterMyWorkflow(myWorker, newMyWorkflowImpl)
```

To implement an activity:

```go
func myActivity(ctx context.Context, req *genpkg.MyActivityRequest) (*genpkg.MyActivityResponse, error) {
  // Do stuff
}
```

And to register it:

```go
genpkg.RegisterMyActivity(myWorker, myActivity)
```

Or you can implement the entire `ActivitiesImpl` generated interface and call `RegisterActivities`.

There are also helpers to call an activity from a workflow, for example:

```go
resp := genpkg.MyActivity(
  ctx,
  &workflow.ActivityOptions{ScheduleToCloseTimeout: 1 * time.Minute},
  &genpkg.MyActivityResponse{},
).Get(ctx)
if err != nil {
  panic(err)
}
log.Printf("Activity response: %v", resp)
```

A similar `MyActivityLocal` is also generated for executing local activities. Calls for starting child workflows are
also available.

## Options and Generated Code Guide

Temporal constructs are defined as `rpc` calls on a `service`. A `service` can be considered a "bundle" of the
following items:

* [Workflows](https://docs.temporal.io/docs/go/workflows)
* [Activities](https://docs.temporal.io/docs/go/activities)
* [Queries](https://docs.temporal.io/docs/go/queries)
* [Signals](https://docs.temporal.io/docs/go/signals)
* Calls - combines requests via signals with responses via activities (or other signals if called from another workflow)

In order to use the options, `import "temporalpb/sdk.proto";` will have to be added and when using `protoc`, a `-I`
include will need to be set to this directory so that import can be resolved.

In addition to the defails below, reference the [temporalpb/sdk.proto](temporalpb/sdk.proto) file for which options can
be set.

### Workflows

Workflows are `rpc` calls with the `temporal.sdk.workflow` option set. For example:

```proto
  rpc SomeWorkflow(SomeWorkflowRequest) returns (SomeWorkflowResponse) {
    option (temporal.sdk.workflow) = {
      query: { ref: 'SomeQuery' }
      signal: { ref: 'SomeSignal' }
      call: { ref: 'SomeCall' }
      workflow_id_field: 'id'
      default_options {
        task_queue: 'my-task-queue'
      }
    };
  }
```

The request and/or response values can be set to `google.protobuf.Empty` to signify no request and/or response.

* `query` can have a `ref` to a query in the same service for accepting queries
* `signal` can have a `ref` to a signal in the same service for accepting signals
* `signal_start` can have a `ref` to a signal in the same service for accepting a start signal
* `call` can have a `ref` to a call in the same service for accepting calls
* `workflow_id_field` can be set to a field in the request to be treated as the ID
* `default_options` has some options values that can be set for `client.StartWorkflowOptions` if they aren't otherwise
  set

The following client-side items may be generated for such a workflow:

```go
type Client interface {
  ExecuteSomeWorkflow(ctx context.Context, opts *client.StartWorkflowOptions, req *SomeWorkflowRequest) (SomeWorkflowRun, error)

  GetSomeWorkflow(ctx context.Context, workflowID, runID string) (SomeWorkflowRun, error)
}

type SomeWorkflowRun interface {
  // ID is the workflow ID.
  ID() string

  // RunID is the workflow run ID.
  RunID() string

  // Get returns the completed workflow value, waiting if necessary.
  Get(ctx context.Context) (*SomeWorkflowResponse, error)

  SomeQuery(ctx context.Context, req *SomeQueryRequest) (*SomeQueryResponse, error)

  SomeSignal(ctx context.Context, req *SomeSignalRequest) error

  SomeCall(ctx context.Context, req *SomeCallRequest) (*SomeCallResponse, error)
}
```

The following worker-side items may be generated for such a workflow:

```go
type SomeWorkflowImpl interface {
  Run(workflow.Context) (*SomeWorkflowResponse, error)

  SomeQuery(*SomeQueryRequest) (*SomeQueryResponse, error)
}

func RegisterSomeWorkflow1(
  r worker.WorkflowRegistry,
  newImpl func(workflow.Context, *SomeWorkflowInput) (SomeWorkflowImpl, error),
)

type SomeWorkflow1Input struct {
  Req        *SomeWorkflowRequest
  SomeSignal SomeSignal
  SomeCall   SomeCall
}

func SomeWorkflowChild(ctx workflow.Context, opts *workflow.ChildWorkflowOptions, req *SomeWorkflowRequest) SomeWorkflowChildRun

type SomeWorkflowChildRun struct{ Future workflow.ChildWorkflowFuture }
func (SomeWorkflowChildRun) WaitStart(ctx workflow.Context) (*workflow.Execution, error)
func (SomeWorkflowChildRun) SelectStart(sel workflow.Selector, fn func(SomeWorkflowChildRun)) workflow.Selector
func (SomeWorkflowChildRun) Get(ctx workflow.Context) (*SomeWorkflowResponse, error)
func (SomeWorkflowChildRun) Select(sel workflow.Selector, fn func(SomeWorkflowChildRun)) workflow.Selector
func (SomeWorkflowChildRun) SomeSignal(ctx workflow.Context, req *SomeSignalRequest) workflow.Future
func (SomeWorkflowChildRun) SomeCall(ctx workflow.Context, req *SomeCallRequest) (SomeCallResponseExternal, error)
```

### Activities

Activities are `rpc` calls with the `temporal.sdk.activity` option set. For example:

```proto
  rpc SomeActivity(SomeActivityRequest) returns (SomeActivityResponse) {
    option (temporal.sdk.activity) = {
      default_options {
        start_to_close_timeout: { seconds: 10 }
      }
    };
  }
```

The request and/or response values can be set to `google.protobuf.Empty` to signify no request and/or response.

* `default_options` has some options values that can be set to replace `workflow.ActivityOptions` if they aren't otherwise
  set

The following worker-side items may be generated for such an activity:

```go
type ActivitiesImpl interface {
  SomeActivity(context.Context, *SomeActivityRequest) (*SomeActivityResponse, error)
}

func RegisterActivities(r worker.ActivityRegistry, a ActivitiesImpl)

func RegisterSomeActivity(
  r worker.ActivityRegistry,
  impl func(context.Context, *SomeActivity3Request) (*SomeActivity3Response, error),
)

func SomeActivity(
  ctx workflow.Context,
  opts *workflow.ActivityOptions,
  req *SomeActivityRequest,
) SomeActivityFuture

func SomeActivityLocal(
  ctx workflow.Context,
  opts *workflow.LocalActivityOptions,
  fn func(context.Context, *SomeActivityRequest) (*SomeActivityResponse, error),
  req *SomeActivityRequest,
) SomeActivityFuture

type SomeActivityFuture struct{ Future workflow.Future }
func (SomeActivityFuture) Get(ctx workflow.Context) (*SomeActivityResponse, error)
func (SomeActivityFuture) Select(sel workflow.Selector, fn func(SomeActivityFuture)) workflow.Selector
```

### Queries

Queries are `rpc` calls with the `temporal.sdk.query` options. For example:

```proto
  rpc SomeQuery(SomeQueryRequest) returns (SomeQueryResponse) {
    option (temporal.sdk.query) = { };
  }
```

The request can be set to `google.protobuf.Empty` to signify no request, but a response must always be set.

The following client-side items may be generated for such a query:

```go
type Client interface {
  SomeQuery(ctx context.Context, workflowID, runID string, req *SomeQueryRequest) (*SomeQueryResponse, error)
}
```

### Signals

Signals are `rpc` calls with the `temporal.sdk.signal` options. For example:

```proto
  rpc SomeSignal(SomeSignalRequest) returns (google.protobuf.Empty) {
    option (temporal.sdk.signal) = { };
  }
```

The request can be set to `google.protobuf.Empty` to signify no request, but a response must always be
`google.protobuf.Empty` since signals cannot send responses.

The following client-side items may be generated for such a signal:

```go
type Client interface {
  SomeSignal(ctx context.Context, workflowID, runID string, req *SomeSignalRequest) error
}
```

The following worker-side items may be generated for such a signal:

```go
type SomeSignal struct{ Channel workflow.ReceiveChannel }

func (SomeSignal) Receive(ctx workflow.Context) *SomeSignalRequest
func (SomeSignal) ReceiveAsync() *SomeSignalRequest
func (SomeSignal) Select(sel workflow.Selector, fn func(*SomeSignalRequest)) workflow.Selector

func SomeSignalExternal(ctx workflow.Context, workflowID, runID string, req *SomeSignalRequest) workflow.Future
```

### Calls

Queries are `rpc` calls with the `temporal.sdk.query` options. For example:

```proto
  rpc SomeCall(SomeCallRequest) returns (SomeCallResponse) {
    option (temporal.sdk.call) = { };
  }
```

The request and response are required since they contain information to tie to each other.

Every call request must have an `id` field. If a call has a `response_task_queue` field, it can be called from the
client side since it will use that task queue to send a response activity on. If a call has a `response_workflow_id`, it
can be called from another workflow since it will use that ID to send the response signal on.

Every call response must have an `id` field which is used to set the ID of the request.

The following client-side items may be generated for such a call:

```go
type Client interface {
  SomeCall(ctx context.Context, workflowID, runID string, req *SomeCallRequest) (*SomeCallResponse, error)
}
```

The following worker-side items may be generated for such a call:

```go
type SomeCall struct{ Channel workflow.ReceiveChannel }

func (SomeCall) Receive(ctx workflow.Context) *SomeCallRequest
func (SomeCall) ReceiveAsync() *SomeCallRequest
func (SomeCall) Select(sel workflow.Selector, fn func(*SomeCallRequest)) workflow.Selector
func (SomeCall1) Respond(
  ctx workflow.Context,
  opts *workflow.ActivityOptions,
  req *SomeCallRequest,
  resp *SomeCallResponse,
) workflow.Future

func SomeCallExternal(ctx workflow.Context, workflowID, runID string, req *SomeCallRequest) (SomeCallResponseExternal, error)

type SomeCallResponseExternal struct {
	Future  workflow.Future
	Channel workflow.ReceiveChannel
}

func (SomeCallResponseExternal) WaitSent(ctx workflow.Context) error
func (SomeCallResponseExternal) SelectSent(sel workflow.Selector, fn func(SomeCallResponseExternal)) workflow.Selector
func (SomeCallResponseExternal) Receive(ctx workflow.Context) *SomeCallResponse
func (SomeCallResponseExternal) ReceiveAsync() *SomeCallResponse
func (SomeCallResponseExternal) Select(sel workflow.Selector, fn func(*SomeCallResponse)) workflow.Selector
```

## TODO

* Better documentation
* A real example
* More tests
* More workflow and activity options
* Option to set the generated code prefix
* Code cleanup to support more reuse (lots of copy/pasted strings everywhere)
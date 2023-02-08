package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cretz/temporal-sdk-go-advanced/temporalmodern/temporalng4"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Notes:
// * Approach - Similar to temporalng3 but the spec is a var for reference
//   instead of client calls. Still contains client calls on the fields, though
//   more explicit and does not include impl calls on fields.
// * Pro - Easy to have custom names and have struct spec
// * Con - Have to reference this spec every time you want something typed
// * Con - Bit of a strange syntax (though could be broken out, doesn't have to
//   be an anon struct)

var MyWorkflow1 = temporalng4.WorkflowSpec[struct {
	// These name tags are implied with defaults, this is just for demo purposes
	temporalng4.Workflow[string, string] `name:"MyWorkflow1"`

	MySignal1 temporalng4.WorkflowSignal[string] `name:"MySignal1"`

	MyQuery1 temporalng4.WorkflowQuery[string, string] `name:"MyQuery1"`

	// Nothing besides workflow, signals, and queries allowed in this struct
}]()

func MyActivity1(ctx context.Context, param string) (string, error) {
	return param, nil
}

// I can't think of a better way to get the spec to this
func MyWorkflow1Run(ctx temporalng4.Context, param string) (string, error) {
	temporalng4.SetWorkflowQueryHandler(ctx, MyWorkflow1.MyQuery1, func(s string) (string, error) {
		return "Queried with param: " + s, nil
	})
	signalVal := temporalng4.GetWorkflowSignalChannel(ctx, MyWorkflow1.MySignal1).Receive()
	// Thoughts on execute vs start?
	actVal, err := temporalng4.StartActivity(ctx, MyActivity1, "some act arg", temporalng4.StartActivityOptions{}).Get()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("param: %v, signal: %v, activity: %v", param, signalVal, actVal), nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.TODO()

	client, err := client.Dial(client.Options{})
	if err != nil {
		return err
	}
	defer client.Close()

	worker := worker.New(client, "my-task-queue", worker.Options{})
	// Cannot infer types here if we go the other other direction of top-level
	// RegisterWorkflow
	MyWorkflow1.Register(worker, MyWorkflow1Run)
	temporalng4.RegisterActivity(worker, MyActivity1)

	if err := worker.Start(); err != nil {
		return err
	}
	defer worker.Stop()

	handle, err := MyWorkflow1.Start(ctx, client, "some arg", temporalng4.WorkflowStartOptions{
		ID:        "workflow-id",
		TaskQueue: "my-task-queue",
	})
	if err != nil {
		return err
	}

	// Just demonstrates two ways to re-obtain the handle
	handle = MyWorkflow1.GetHandle(client, handle.ID)

	// Putting the handle on the options instead of the params here seems
	// more reasonable
	err = MyWorkflow1.MySignal1.Send(ctx, client, "some signal arg", temporalng4.WorkflowSignalOptions{
		Workflow: handle,
	})
	if err != nil {
		return err
	}

	queryResult, err := MyWorkflow1.MyQuery1.Invoke(ctx, client, "some query arg", temporalng4.WorkflowQueryOptions{
		Workflow: handle,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Query result: %v\n", queryResult)

	result, err := handle.Get(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("workflow result: %v\n", result)

	return nil
}

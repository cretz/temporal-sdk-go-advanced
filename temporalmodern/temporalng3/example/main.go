package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cretz/temporal-sdk-go-advanced/temporalmodern/temporalng3"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Notes:
// * Approach - Defined as a struct of client calls but still using activity
//   functions
// * Pro - Easy to have custom names and have struct spec
// * Con - A bit confusing to have the fields be calls?
// * Con - Having a separate handle from client is a bit confusing
// * Con - Having the spec provided to the workflow is a bit confusing

type MyWorkflow1 struct {
	// These name tags are implied with defaults, this is just for demo purposes
	temporalng3.Workflow[string, string] `name:"MyWorkflow1"`

	MySignal1 temporalng3.WorkflowSignal[string] `name:"MySignal1"`

	MyQuery1 temporalng3.WorkflowQuery[string, string] `name:"MyQuery1"`

	// Nothing besides workflow, signals, and queries allowed in this struct
}

func MyActivity1(ctx context.Context, param string) (string, error) {
	return param, nil
}

// I can't think of a better way to get the spec to this
func MyWorkflow1Run(ctx temporalng3.Context, spec MyWorkflow1, param string) (string, error) {
	spec.MyQuery1.SetHandler(ctx, func(s string) (string, error) {
		return "Queried with param: " + s, nil
	})
	signalVal := spec.MySignal1.GetChannel(ctx).Receive()
	// Thoughts on execute vs start?
	actVal, err := temporalng3.StartActivity(ctx, MyActivity1, "some act arg", temporalng3.StartActivityOptions{}).Get()
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
	temporalng3.RegisterWorkflow(worker, MyWorkflow1Run)
	temporalng3.RegisterActivity(worker, MyActivity1)

	if err := worker.Start(); err != nil {
		return err
	}
	defer worker.Stop()

	wfClient := temporalng3.NewWorkflowClient[MyWorkflow1](client)
	handle, err := wfClient.Start(ctx, "some arg", temporalng3.WorkflowStartOptions{
		ID:        "workflow-id",
		TaskQueue: "my-task-queue",
	})
	if err != nil {
		return err
	}

	// Just demonstrates two ways to re-obtain the handle
	handle = wfClient.GetHandle(handle.ID)

	// Putting the handle on the options instead of the params here seems
	// more reasonable
	err = wfClient.MySignal1(ctx, "some signal arg", temporalng3.WorkflowSignalOptions{
		Workflow: handle,
	})
	if err != nil {
		return err
	}

	queryResult, err := wfClient.MyQuery1(ctx, "some query arg", temporalng3.WorkflowQueryOptions{
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

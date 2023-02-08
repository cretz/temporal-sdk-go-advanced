package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cretz/temporal-sdk-go-advanced/temporalmodern/temporalng1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Notes:
// * Approach - Each defined thing is a separate definition
// * Pro - All things are typed
// * Pro - Simplicity of definition and clarity of providing name
// * Con - Cannot associate signals/queries to workflows
// * Con - Have to explicitly provide name every time

var MyWorkflow1 = temporalng1.Workflow[string, string]{Name: "MyWorkflow1"}
var MySignal1 = temporalng1.Signal[string]{Name: "MySignal1"}
var MyQuery1 = temporalng1.Query[string, string]{Name: "MyQuery1"}
var MyActivity1 = temporalng1.Activity[string, string]{Name: "MyActivity1"}

func MyWorkflow1Run(ctx temporalng1.Context, param string) (string, error) {
	MyQuery1.SetHandler(ctx, func(s string) (string, error) {
		return "Queried with param: " + s, nil
	})
	signalVal := MySignal1.GetChannel(ctx).Receive()
	actVal, err := MyActivity1.Execute(ctx, "some act arg", temporalng1.ActivityExecuteOptions{}).Get()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("param: %v, signal: %v, activity: %v", param, signalVal, actVal), nil
}

func MyActivity1Run(ctx context.Context, param string) (string, error) {
	return param, nil
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
	MyWorkflow1.Register(worker, MyWorkflow1Run)
	MyActivity1.Register(worker, MyActivity1Run)

	if err := worker.Start(); err != nil {
		return err
	}
	defer worker.Stop()

	handle, err := MyWorkflow1.Start(ctx, client, "some wf arg", temporalng1.WorkflowStartOptions{
		ID:        "workflow-id",
		TaskQueue: "my-task-queue",
	})
	if err != nil {
		return err
	}

	// Just demonstrates how to re-obtain the handle
	handle = MyWorkflow1.GetWorkflowHandle(client, handle.ID)

	err = MySignal1.Send(ctx, handle, "some signal arg", temporalng1.SignalSendOptions{})
	if err != nil {
		return err
	}

	queryResult, err := MyQuery1.Invoke(ctx, handle, "some query arg", temporalng1.QueryInvokeOptions{})
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

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cretz/temporal-sdk-go-advanced/temporalmodern/temporalng2"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Notes:
// * Approach - Signals and queries as interface methods, activities as
//   interface
// * Pro - Can tie signals and queries to workflows
// * Pro - Stateful struct instantiated each workflow run, can help users keep
//   state better than callbacks
// * Pro - Stateful activities built in since they are iface impls
// * Con - Using signal handlers can be a problem because to build this in user
//   land, we'd wrap run with our own that have a signal selector in a coroutine
//   but Go doesn't abide by the signals-first doctrine of other SDKs.
// * Con - Not easy to set custom names
// * Con - Registration is a bit ugly
// * Con - Can be non-obvious that you need pointers and state sharing
// * Con - Forces people to make interfaces and impls like Java - more lines of
//   code than we might want
// * Con - Makes handle getting ugly because we can't infer

type MyWorkflow1 interface {
	temporalng2.Workflow[string, string]

	MySignal1(temporalng2.Context, string)

	MyQuery1(string) (string, error)

	// Names are defaulted, this just demonstrates how one might override
	temporalng2.CustomNaming[struct {
		MyWorkflow1 `name:"MyWorkflow1"`
		MySignal1   string `name:"MySignal1"`
	}]
}

type myWorkflow1 struct {
	// Needed for communicating signal to run
	mySignalChan *temporalng2.Channel[string]
}

func (m *myWorkflow1) Run(ctx temporalng2.Context, param string) (string, error) {
	signalVal := m.getMySignalChan().Receive()
	actVal, err := temporalng2.ExecuteActivity(ctx, Activities.MyActivity1,
		"some act arg", temporalng2.ExecuteActivityOptions{}).Get()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("param: %v, signal: %v, activity: %v", param, signalVal, actVal), nil
}

func (m *myWorkflow1) MySignal1(ctx temporalng2.Context, param string) {
	m.mySignalChan.Send(param)
}

func (*myWorkflow1) MyQuery1(param string) (string, error) {
	return "Queried with param: " + param, nil
}

// Has to be in separate function, because what if MySignal1 runs before Run
func (m *myWorkflow1) getMySignalChan() *temporalng2.Channel[string] {
	if m.mySignalChan == nil {
		m.mySignalChan = temporalng2.NewChannel[string](1)
	}
	return m.mySignalChan
}

type Activities interface {
	MyActivity1(context.Context, string) (string, error)

	// Names are defaulted, this just demonstrates how one might override
	temporalng2.CustomNaming[struct {
		MyActivity1 string `name:"MyActivity1"`
	}]
}

type activities struct {
}

func (activities) MyActivity1(ctx context.Context, param string) (string, error) {
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
	// No, those extra args cannot be easily inferred. Yes, we need an instance of
	// the interface, we cannot accept *myWorkflow1 as a type arg because there is
	// no way to ensure it implements the interface (type args can't have
	// constraints on other type args). We'll throw away the instance.
	temporalng2.RegisterWorkflow[MyWorkflow1, string, string](worker, &myWorkflow1{})
	temporalng2.RegisterActivities[Activities](worker, activities{})

	if err := worker.Start(); err != nil {
		return err
	}
	defer worker.Stop()

	// Have to use .Run instead of just MyWorkflow1 because we can't infer arg
	// types otherwise
	handle, err := temporalng2.StartWorkflow(ctx, client, MyWorkflow1.Run, "some arg", temporalng2.StartWorkflowOptions{
		ID:        "workflow-id",
		TaskQueue: "my-task-queue",
	})
	if err != nil {
		return err
	}

	// Just demonstrates how to re-obtain the handle. Have to use run reference
	// for the same reason
	handle = temporalng2.GetWorkflowHandle(client, MyWorkflow1.Run, handle.ID)

	err = temporalng2.SignalWorkflow(ctx, handle, MyWorkflow1.MySignal1, "some signal arg",
		temporalng2.SignalWorkflowOptions{})
	if err != nil {
		return err
	}

	queryResult, err := temporalng2.QueryWorkflow(ctx, handle, MyWorkflow1.MyQuery1, "some query arg",
		temporalng2.QueryWorkflowOptions{})
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

package sandboxrt

import (
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func RunMain(
	taskQueue string,
	initWorker func(*worker.Options, worker.Registry) error,
	initClient func(*client.Options) error,
) {
	if err := Run(taskQueue, initWorker, initClient); err != nil {
		log.Fatal(err)
	}
}

func Run(
	taskQueue string,
	initWorker func(*worker.Options, worker.Registry) error,
	initClient func(*client.Options) error,
) error {
	panic("TODO")
}

func DefaultInitClient(*client.Options) error { return nil }

func NewClient(initClient func(*client.Options) error) (client.Client, error) {
	var options client.Options
	if err := initClient(&options); err != nil {
		return nil, err
	}
	return client.NewClient(options)
}

// Returns nil with no error if no activities
func NewActivityWorker(
	client client.Client,
	taskQueue string,
	initWorker func(*worker.Options, worker.Registry) error,
	// For registering local activities
	ipcHost IPCHost,
) (worker.Worker, error) {
	options := worker.Options{DisableWorkflowWorker: true}
	var reg LocalRegistry
	if err := initWorker(&options, &reg); err != nil {
		return nil, err
	}
	if len(reg.Activities) == 0 {
		return nil, nil
	}

	worker := worker.New(client, taskQueue, options)
	for _, a := range reg.Activities {
		// TODO(cretz): Support struct activities over IPC
		if reflect.ValueOf(a.Activity).Kind() != reflect.Func {
			return nil, fmt.Errorf("only activity functions currently supported")
		}
		if a.Options.Name == "" {
			a.Options.Name, _ = getTemporalRegisterFunctionName(a.Activity)
		}
		worker.RegisterActivityWithOptions(a.Activity, a.Options)
		// Register the local activity for IPC
		if err := RegisterLocalActivityFunc(ipcHost, a.Options.Name, a.Activity); err != nil {
			return nil, fmt.Errorf("failed registering activity %v for local IPC: %w", a.Options.Name, err)
		}
	}

	return worker, nil
}

// Returns nil with no error if no workflows
func NewWorkflowWorker(
	client client.Client,
	taskQueue string,
	initWorker func(*worker.Options, worker.Registry) error,
	// For invoking local activities
	ipcClient IPCClient,
) (worker.Worker, error) {
	options := worker.Options{LocalActivityWorkerOnly: true}
	var reg LocalRegistry
	if err := initWorker(&options, &reg); err != nil {
		return nil, err
	}
	if len(reg.Workflows) == 0 {
		return nil, nil
	}

	worker := worker.New(client, taskQueue, options)
	for _, w := range reg.Workflows {
		worker.RegisterWorkflowWithOptions(w.Workflow, w.Options)
	}

	for _, a := range reg.Activities {
		// TODO(cretz): Support struct activities over IPC
		if reflect.ValueOf(a.Activity).Kind() != reflect.Func {
			return nil, fmt.Errorf("only activity functions currently supported")
		}
		if a.Options.Name == "" {
			a.Options.Name, _ = getTemporalRegisterFunctionName(a.Activity)
		}
		fn, err := MakeLocalActivityProxyFunc(ipcClient, a.Options.Name, a.Activity)
		if err != nil {
			return nil, fmt.Errorf("failed making local activity IPC for activity %v: %w", a.Options.Name, err)
		}
		worker.RegisterActivityWithOptions(fn, a.Options)
	}

	return worker, nil
}

func getTemporalRegisterFunctionName(i interface{}) (name string, isMethod bool) {
	if fullName, ok := i.(string); ok {
		return fullName, false
	}
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	// Full function name that has a struct pointer receiver has the following format
	// <prefix>.(*<type>).<function>
	isMethod = strings.ContainsAny(fullName, "*")
	elements := strings.Split(fullName, ".")
	shortName := elements[len(elements)-1]
	// This allows to call activities by method pointer
	// Compiler adds -fm suffix to a function name which has a receiver
	// Note that this works even if struct pointer used to get the function is nil
	// It is possible because nil receivers are allowed.
	// For example:
	// var a *Activities
	// ExecuteActivity(ctx, a.Foo)
	// will call this function which is going to return "Foo"
	return strings.TrimSuffix(shortName, "-fm"), isMethod
}

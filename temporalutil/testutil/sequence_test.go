package testutil_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/cretz/temporal-sdk-go-advanced/temporalutil/testutil"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func ExampleAddSequence() {
	var suite testsuite.WorkflowTestSuite
	suite.SetLogger(simpleLogger)
	env := suite.NewTestWorkflowEnvironment()

	// Add sequence of stuff
	testutil.AddSequence(env, func(seq testutil.Sequencer) {
		fmt.Println("Started")
		env.SignalWorkflow("signal", "signal1")
		env.SignalWorkflow("signal", "signal2")
		seq.Tick()
		env.SignalWorkflow("signal", "signal3")
		env.SignalWorkflow("signal", "signal4")
		seq.Sleep(10 * time.Hour)
		env.SignalWorkflow("signal", "signal5")
		env.SignalWorkflow("signal", "signal6")
		seq.Sleep(0)
		env.SignalWorkflow("signal", "finish")
		fmt.Println("Finished")
	})

	// Run workflow that captures signals
	env.ExecuteWorkflow(func(ctx workflow.Context) (signalsReceived []string, err error) {
		sig := workflow.GetSignalChannel(ctx, "signal")
		var sigVal string
		for sigVal != "finish" {
			sig.Receive(ctx, &sigVal)
			signalsReceived = append(signalsReceived, sigVal)
		}
		return
	})

	// Dump signals received
	if env.GetWorkflowError() != nil {
		panic(env.GetWorkflowError())
	}
	var signalsReceived []string
	env.GetWorkflowResult(&signalsReceived)
	fmt.Println("Signals Received: " + strings.Join(signalsReceived, ", "))
	// Output:
	// Started
	// DEBUG Auto fire timer TimerID 1 TimerDuration 1ns TimeSkipped 1ns
	// DEBUG Auto fire timer TimerID 2 TimerDuration 10h0m0s TimeSkipped 10h0m0s
	// Finished
	// Signals Received: signal1, signal2, signal3, signal4, signal5, signal6, finish
}

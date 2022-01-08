package sandboxrt

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type LocalRegistry struct {
	Workflows  []*WorkflowRegistration
	Activities []*ActivityRegistration
}

type WorkflowRegistration struct {
	Workflow interface{}
	Options  workflow.RegisterOptions
}

type ActivityRegistration struct {
	Activity interface{}
	Options  activity.RegisterOptions
}

func (l *LocalRegistry) RegisterWorkflow(w interface{}) {
	l.RegisterWorkflowWithOptions(w, workflow.RegisterOptions{})
}

func (l *LocalRegistry) RegisterWorkflowWithOptions(w interface{}, options workflow.RegisterOptions) {
	l.Workflows = append(l.Workflows, &WorkflowRegistration{Workflow: w, Options: options})
}

func (l *LocalRegistry) RegisterActivity(a interface{}) {
	l.RegisterActivityWithOptions(a, activity.RegisterOptions{})
}

func (l *LocalRegistry) RegisterActivityWithOptions(a interface{}, options activity.RegisterOptions) {
	l.Activities = append(l.Activities, &ActivityRegistration{Activity: a, Options: options})
}

package testutil

import (
	"time"

	"go.temporal.io/sdk/testsuite"
)

// Sequencer allows you to wait during sequences
type Sequencer interface {
	// Tick yields to workflow. This is just Sleep(1) here as a convenience.
	Tick()

	// Sleep skips the given duration before executing the next code
	Sleep(time.Duration)
}

// AddSequence invokes RegisterDelayedCallback that will start the given
// function and continually register new callbacks as sleeps are needed.
func AddSequence(env *testsuite.TestWorkflowEnvironment, fn func(seq Sequencer)) {
	env.RegisterDelayedCallback(func() {
		endCh := make(chan struct{})
		seq := &sequencer{env, endCh}
		go func() {
			fn(seq)
			close(seq.lastCallbackEndCh)
		}()
		<-endCh
	}, 0)
}

type sequencer struct {
	env               *testsuite.TestWorkflowEnvironment
	lastCallbackEndCh chan struct{}
}

func (s *sequencer) Tick() { s.Sleep(1) }

func (s *sequencer) Sleep(d time.Duration) {
	beginCh, endCh := make(chan struct{}), make(chan struct{})
	s.env.RegisterDelayedCallback(func() {
		close(beginCh)
		<-endCh
	}, d)
	if s.lastCallbackEndCh != nil {
		close(s.lastCallbackEndCh)
	}
	s.lastCallbackEndCh = endCh
	<-beginCh
}

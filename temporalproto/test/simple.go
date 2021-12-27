package test

import (
	"context"
	"strings"
	"time"

	"github.com/cretz/temporal-sdk-go-advanced/temporalproto/test/simplepb"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type someWorkflow1 struct {
	*simplepb.SomeWorkflow1Input
	sel    workflow.Selector
	events []string
}

func Register(r worker.Registry) {
	simplepb.RegisterSomeWorkflow1(r, newSomeWorkflow1)
	simplepb.RegisterActivities(r, Activities)
}

func newSomeWorkflow1(ctx workflow.Context, in *simplepb.SomeWorkflow1Input) (simplepb.SomeWorkflow1Impl, error) {
	return &someWorkflow1{SomeWorkflow1Input: in, sel: workflow.NewSelector(ctx)}, nil
}

func (s *someWorkflow1) Run(ctx workflow.Context) (*simplepb.SomeWorkflow1Response, error) {
	s.events = append(s.events, "started with param "+s.Req.RequestVal)

	// Call regular activity
	resp, err := simplepb.SomeActivity3(ctx, nil,
		&simplepb.SomeActivity3Request{RequestVal: "some activity param"}).Get(ctx)
	if err != nil {
		return nil, err
	}
	s.events = append(s.events, "some activity 3 with response "+resp.ResponseVal)

	// Call local activity
	resp, err = simplepb.SomeActivity3Local(ctx, nil, Activities.SomeActivity3,
		&simplepb.SomeActivity3Request{RequestVal: "some local activity param"}).Get(ctx)
	if err != nil {
		return nil, err
	}
	s.events = append(s.events, "some local activity 3 with response "+resp.ResponseVal)

	// Handle input
	s.SomeSignal1.Select(s.sel, func() {
		s.events = append(s.events, "some signal 1")
	})
	s.SomeSignal2.Select(s.sel, func(req *simplepb.SomeSignal2Request) {
		s.events = append(s.events, "some signal 2 with param "+req.RequestVal)
	})
	s.SomeCall1.Select(s.sel, func(req *simplepb.SomeCall1Request) {
		s.events = append(s.events, "some call 1 with param "+req.RequestVal)
		respFut := s.SomeCall1.Respond(
			ctx,
			// Never retry
			&workflow.ActivityOptions{
				ScheduleToCloseTimeout: 1 * time.Minute,
				RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
			},
			req, &simplepb.SomeCall1Response{ResponseVal: "some response"},
		)
		s.sel.AddFuture(respFut, func(workflow.Future) {})
	})
	s.sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {})

	// Run until done
	for ctx.Err() == nil {
		s.sel.Select(ctx)
	}
	return &simplepb.SomeWorkflow1Response{ResponseVal: strings.Join(s.events, "\n")}, nil
}

func (s *someWorkflow1) SomeQuery1() (*simplepb.SomeQuery1Response, error) {
	return &simplepb.SomeQuery1Response{
		ResponseVal: strings.Join(s.events, "\n") + "\nsome query 1",
	}, nil
}

func (s *someWorkflow1) SomeQuery2(req *simplepb.SomeQuery2Request) (*simplepb.SomeQuery2Response, error) {
	return &simplepb.SomeQuery2Response{
		ResponseVal: strings.Join(s.events, "\n") + "\nsome query 2 with param " + req.RequestVal,
	}, nil
}

type activities struct{}

var Activities simplepb.ActivitiesImpl = activities{}
var ActivityEvents []string

func (activities) SomeActivity1(context.Context) error {
	ActivityEvents = append(ActivityEvents, "some activity 1")
	return nil
}

func (activities) SomeActivity2(ctx context.Context, req *simplepb.SomeActivity2Request) error {
	ActivityEvents = append(ActivityEvents, "some activity 2 with param "+req.RequestVal)
	return nil
}

func (activities) SomeActivity3(ctx context.Context, req *simplepb.SomeActivity3Request) (*simplepb.SomeActivity3Response, error) {
	ActivityEvents = append(ActivityEvents, "some activity 3 with param "+req.RequestVal)
	return &simplepb.SomeActivity3Response{ResponseVal: "some response"}, nil
}

package test_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DataDog/temporalite/temporaltest"
	"github.com/cretz/temporal-sdk-go-advanced/temporalproto/test"
	"github.com/cretz/temporal-sdk-go-advanced/temporalproto/test/simplepb"
	"github.com/cretz/temporal-sdk-go-advanced/temporalutil/clientutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

func TestSomeWorkflow1(t *testing.T) {
	require := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server and worker w/ workflow registered
	srv := temporaltest.NewServer(temporaltest.WithT(t))
	taskQueue := uuid.NewString()
	w := srv.Worker(taskQueue, test.Register)

	// Create the client
	callResponseHandler, err := clientutil.NewCallResponseHandler(clientutil.CallResponseHandlerOptions{
		TaskQueue: taskQueue,
		Worker:    w,
	})
	require.NoError(err)
	c := simplepb.NewClient(simplepb.ClientOptions{
		Client:              srv.Client(),
		CallResponseHandler: callResponseHandler,
	})

	// Start the workflow
	run, err := c.ExecuteSomeWorkflow1(
		ctx,
		&client.StartWorkflowOptions{TaskQueue: taskQueue},
		&simplepb.SomeWorkflow1Request{RequestVal: "some request"},
	)
	require.NoError(err)

	// Query until we get the right events
	require.Eventually(func() bool {
		resp, err := run.SomeQuery1(ctx)
		require.NoError(err)
		return reflect.DeepEqual([]string{
			"started with param some request",
			"some activity 3 with response some response",
			"some local activity 3 with response some response",
			"some query 1",
		}, strings.Split(resp.ResponseVal, "\n"))
	}, 2*time.Second, 200*time.Millisecond)

	// Check the activity events
	require.Equal([]string{
		"some activity 3 with param some activity param",
		"some activity 3 with param some local activity param",
	}, test.ActivityEvents)

	// Try a req/resp call
	resp, err := run.SomeCall1(ctx, &simplepb.SomeCall1Request{RequestVal: "some call request"})
	require.NoError(err)
	require.Equal("some response", resp.ResponseVal)
}

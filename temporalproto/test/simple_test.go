package test_test

import (
	"testing"

	"github.com/DataDog/temporalite/temporaltest"
	"github.com/cretz/temporal-sdk-go-advanced/temporalproto/test"
	"github.com/google/uuid"
)

func TestSomeWorkflow1(t *testing.T) {
	srv := temporaltest.NewServer(temporaltest.WithT(t))
	defer srv.Stop()

	// Create worker and register
	taskQueue := uuid.NewString()
	w := srv.Worker(taskQueue, test.Register)
	defer w.Stop()

	// Start the workflow then check query

}

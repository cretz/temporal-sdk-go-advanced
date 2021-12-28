package temporalsqlite_test

import (
	"testing"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"go.temporal.io/sdk/testsuite"
)

func TestSimpleQueries(t *testing.T) {

	// Register workflow
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	temporalsqlite.RegisterSqliteWorker(env, temporalsqlite.SqliteWorkerOptions{})

	// Add some queries
	// TODO(cretz): The rest
}

package test_test

import (
	"context"
	"testing"

	"github.com/DataDog/temporalite/temporaltest"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func (s *Suite) TestSimpleQueries() {
	db := s.connectDB()

	// Create a table via signal
	s.NoError(db.UpdateSimple(s.ctx, "CREATE TABLE mytable1 (v1 PRIMARY KEY, v2, v3)"))

	// Insert into table via signal
	s.NoError(db.UpdateSimple(
		s.ctx,
		"INSERT INTO mytable1 (v1, v2, v3) VALUES (?, ?, ?), (?, ?, ?)",
		"foo1", 1.23, []byte("bar1"),
		567, true, nil,
	))

	// Query table and check result
	resp1, err := db.QuerySimple(s.ctx, "SELECT * FROM mytable1")
	s.NoError(err)
	s.Equal([]string{"v1", "v2", "v3"}, resp1.ColumnNames)
	s.Equal([][]interface{}{
		{"foo1", 1.23, []byte("bar1")},
		{int64(567), int64(1), nil},
	}, resp1.Rows)

	// Insert via call
	resp2, err := db.ExecSimple(
		s.ctx,
		"INSERT INTO mytable1 VALUES (?, ?, ?) RETURNING v2 + 12 AS someval",
		1, 2, 3,
	)
	s.NoError(err)
	s.Equal([]string{"someval"}, resp2.ColumnNames)
	s.Equal([][]interface{}{{int64(14)}}, resp2.Rows)
}

func (s *Suite) TestFailedQuery() {
	db := s.connectDB()

	_, err := db.QuerySimple(s.ctx, "SELECT * FROM notexist")
	s.Error(err)
	s.Contains(err.Error(), "no such table: notexist")
}

func (s *Suite) TestMutationDuringQuery() {
	db := s.connectDB()
	s.NoError(db.UpdateSimple(s.ctx, "CREATE TABLE mytable1 (v1 PRIMARY KEY, v2, v3)"))

	// This should fail
	_, err := db.QuerySimple(s.ctx, "INSERT INTO mytable1 VALUES (1, 2, 3)")
	s.Error(err)
	s.Contains(err.Error(), "statement expected to be read only")
}

func (s *Suite) TestNonDeterministicFunc() {
	s.T().Skip("TODO")
}

func (s *Suite) TestNamedParams() {
	s.T().Skip("TODO")
}

func (s *Suite) TestSerialize() {
	s.T().Skip("TODO")
}

func (s *Suite) TestContinueAsNew() {
	s.T().Skip("TODO")
}

type Suite struct {
	suite.Suite
	*require.Assertions
	taskQueue string
	server    *temporaltest.TestServer
	ctx       context.Context
	cancel    context.CancelFunc
}

func TestSuite(t *testing.T) { suite.Run(t, new(Suite)) }

func (s *Suite) SetupSuite() {
	s.Assertions = require.New(s.T())

	// Start server and register worker
	s.taskQueue = "sqlite-test-" + uuid.NewString()
	s.server = temporaltest.NewServer(temporaltest.WithT(s.T()))
	s.server.Worker(s.taskQueue, func(r worker.Registry) {
		temporalsqlite.RegisterSqliteWorker(r, temporalsqlite.SqliteWorkerOptions{})
	})
}

func (s *Suite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *Suite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Suite) connectDB() *temporalsqlite.Client {
	client, err := temporalsqlite.ConnectDB(context.Background(), s.server.Client(), temporalsqlite.ConnectDBOptions{
		StartWorkflow: client.StartWorkflowOptions{
			ID:        "mytestdb-" + uuid.NewString(),
			TaskQueue: s.taskQueue,
		},
	})
	s.NoError(err)
	s.T().Cleanup(client.Close)
	return client
}

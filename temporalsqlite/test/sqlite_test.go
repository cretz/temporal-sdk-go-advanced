package test_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/DataDog/temporalite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/log"
)

const showServerLogs = false
const showQueries = false

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

	// Multi-query
	resp3, err := db.QueryMulti(s.ctx, "SELECT v2 FROM mytable1 WHERE v1 = 'foo1'; SELECT v2 FROM mytable1 WHERE v1 = 1")
	s.NoError(err)
	s.Equal([][]interface{}{{1.23}}, resp3[0].Rows)
	s.Equal([][]interface{}{{int64(2)}}, resp3[1].Rows)
}

func (s *Suite) TestFailedQuery() {
	db := s.connectDB()
	_, err := db.QuerySimple(s.ctx, "SELECT * FROM notexist")
	s.Error(err)
	s.Contains(err.Error(), "no such table: notexist")
}

func (s *Suite) TestFailedUpdate() {
	db := s.connectDB()
	err := db.UpdateSimple(s.ctx, "SELECT * FROM notexist")
	s.NoError(err)

	// Check that it fails
	run, err := db.GetRun(s.ctx)
	s.NoError(err)
	err = run.Get(s.ctx)
	s.Error(err)
	s.Contains(err.Error(), "no such table: notexist")

	// Check that a reconnect has the same issue
	db = s.connectDBWithOptions(temporalsqlite.ConnectDBOptions{
		StartWorkflow: client.StartWorkflowOptions{ID: run.ID()},
	})
	run2, err := db.GetRun(s.ctx)
	s.NoError(err)
	s.Equal(run.RunID(), run2.RunID())
	err = run.Get(s.ctx)
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
	db := s.connectDB()
	_, err := db.QuerySimple(s.ctx, "SELECT RANDOM()")
	s.Error(err)
	s.Contains(err.Error(), "called non-deterministic function")

	// We don't yet support preventing this kind of non-determinism
	_, err = db.QuerySimple(s.ctx, "SELECT DATETIME('now')")
	s.NoError(err)
}

func (s *Suite) TestNamedParams() {
	db := s.connectDB()
	stmt := temporalsqlite.NewSingleStmt("SELECT $foo + 1, $foo + 2, $foo + $bar")
	stmt.NamedParams = map[string]interface{}{"$foo": 30, "$bar": 10}
	res, err := db.Query(s.ctx, stmt)
	s.NoError(err)
	s.Nil(res[0].Error)
	s.Equal([]interface{}{int64(31), int64(32), int64(40)}, res[0].Successes[0].Rows[0])
}

func (s *Suite) TestSerialize() {
	db := s.connectDB()
	// Create a table and insert 1000 rows
	s.NoError(db.UpdateSimple(s.ctx, "CREATE TABLE mytable1 (v1 PRIMARY KEY, v2, v3)"))
	sql := ""
	for i := 0; i < 1000; i++ {
		sql += fmt.Sprintf("INSERT INTO mytable1 VALUES ('foo%v', 'bar%v', 'baz%v'); ", i, i, i)
	}
	s.NoError(db.UpdateMulti(s.ctx, sql))

	// Serialize
	b, err := db.Serialize(s.ctx)
	s.NoError(err)

	// Create new DB w/ the bytes and use
	conn, err := sqlite.OpenConn(":memory:")
	s.NoError(err)
	defer conn.Close()
	s.NoError(conn.Deserialize(sqlite.NewSerialized("", b, false)))
	var rows [][]string
	s.NoError(sqlitex.ExecTransient(conn, "SELECT * FROM mytable1 WHERE v1 = 'foo437'", func(stmt *sqlite.Stmt) error {
		rows = append(rows, []string{stmt.ColumnText(0), stmt.ColumnText(1), stmt.ColumnText(2)})
		return nil
	}))
	s.Equal([][]string{{"foo437", "bar437", "baz437"}}, rows)
}

func (s *Suite) TestContinueAsNew() {
	// Let's connect with a DB that we want continue-as-new after 10 requests
	db := s.connectDBWithOptions(temporalsqlite.ConnectDBOptions{
		Sqlite: &sqlitepb.SqliteOptions{RequestsUntilContinueAsNew: 10},
	})

	// Get run
	run1, err := db.GetRun(s.ctx)
	s.NoError(err)
	origRunID := run1.RunID()

	// Create a table and do 9 updates
	// TODO(cretz): There is currently a bug where, every so often, when you send
	// _over_ the amount of signals, it takes 10 seconds to continue from where it
	// left off. In talks with server team about this.
	s.NoError(db.UpdateSimple(s.ctx, "CREATE TABLE mytable1 (v1 PRIMARY KEY, v2, v3)"))
	for i := 0; i < 9; i++ {
		s.NoError(db.UpdateSimple(s.ctx, fmt.Sprintf("INSERT INTO mytable1 VALUES ('foo%v', 'bar%v', 'baz%v')", i, i, i)))
	}

	// Wait until a new run occurs
	s.Eventually(func() bool {
		run2, err := db.GetRun(s.ctx)
		s.NoError(err)
		return origRunID != run2.RunID()
	}, 10*time.Second, 100*time.Millisecond)

	// Now issue another insert and select from before and after
	res, err := db.ExecMulti(s.ctx,
		"INSERT INTO mytable1 VALUES ('new1', 'new2', 'new3');"+
			"SELECT * FROM mytable1 WHERE v1 = 'foo8';"+
			"SELECT * FROM mytable1 WHERE v1 = 'new1';")
	s.NoError(err)
	s.Len(res, 3)
	s.Equal([][]interface{}{{"foo8", "bar8", "baz8"}}, res[1].Rows)
	s.Equal([][]interface{}{{"new1", "new2", "new3"}}, res[2].Rows)
}

type Suite struct {
	suite.Suite
	*require.Assertions
	taskQueue string
	// server    *temporaltest.TestServer
	server *temporalite.Server
	client client.Client
	ctx    context.Context
	cancel context.CancelFunc
}

func TestSuite(t *testing.T) { suite.Run(t, new(Suite)) }

func (s *Suite) SetupSuite() {
	s.Assertions = require.New(s.T())

	// Start server and register worker
	s.taskQueue = "sqlite-test-" + uuid.NewString()
	namespace := "test-ns-" + uuid.NewString()
	opts := []temporalite.ServerOption{
		temporalite.WithNamespaces(namespace),
		temporalite.WithPersistenceDisabled(),
		temporalite.WithDynamicPorts(),
	}
	if !showServerLogs {
		opts = append(opts, temporalite.WithLogger(log.NewNoopLogger()))
	}
	var err error
	s.server, err = temporalite.NewServer(opts...)
	s.NoError(err)
	s.NoError(s.server.Start())
	s.T().Cleanup(s.server.Stop)
	s.client, err = s.server.NewClient(context.Background(), namespace)
	s.NoError(err)
	s.T().Cleanup(s.client.Close)
	wrk := worker.New(s.client, s.taskQueue, worker.Options{
		WorkflowPanicPolicy: worker.FailWorkflow,
	})
	temporalsqlite.RegisterSqliteWorker(wrk, temporalsqlite.SqliteWorkerOptions{LogQueries: showQueries})
	s.NoError(wrk.Start())
	s.T().Cleanup(wrk.Stop)
}

func (s *Suite) SetupTest() {
	s.ctx, s.cancel = context.WithTimeout(context.Background(), 20*time.Second)
}

func (s *Suite) TearDownTest() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Suite) connectDB() *temporalsqlite.Client {
	return s.connectDBWithOptions(temporalsqlite.ConnectDBOptions{})
}

func (s *Suite) connectDBWithOptions(opts temporalsqlite.ConnectDBOptions) *temporalsqlite.Client {
	if opts.StartWorkflow.ID == "" {
		opts.StartWorkflow.ID = "mytestdb-" + uuid.NewString()
	}
	if opts.StartWorkflow.TaskQueue == "" {
		opts.StartWorkflow.TaskQueue = s.taskQueue
	}
	client, err := temporalsqlite.ConnectDB(s.ctx, s.client, opts)
	s.NoError(err)
	s.T().Cleanup(client.Close)
	return client
}

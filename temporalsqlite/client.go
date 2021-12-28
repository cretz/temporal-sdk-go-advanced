package temporalsqlite

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"github.com/cretz/temporal-sdk-go-advanced/temporalutil/clientutil"
	"github.com/google/uuid"
	"github.com/pierrec/lz4/v4"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type ConnectDBOptions struct {
	StartWorkflow client.StartWorkflowOptions
	Sqlite        *sqlitepb.SqliteOptions
}

func ConnectDB(ctx context.Context, c client.Client, opts ConnectDBOptions) (*Client, error) {
	// Require the workflow ID
	if opts.StartWorkflow.ID == "" {
		return nil, fmt.Errorf("workflow ID required")
	}
	// Execute/start it
	if opts.Sqlite == nil {
		opts.Sqlite = &sqlitepb.SqliteOptions{}
	}
	run, err := sqlitepb.NewClient(sqlitepb.ClientOptions{Client: c}).ExecuteSqlite(ctx, &opts.StartWorkflow, opts.Sqlite)
	if err != nil {
		return nil, err
	}
	return NewClient(c, run.ID())
}

type Client struct {
	temporalClient client.Client
	sqliteClient   sqlitepb.Client
	responseWorker worker.Worker
	workflowID     string
}

func NewClient(c client.Client, workflowID string) (*Client, error) {
	// Start worker for responses
	taskQueue := "sqlite-response-handler-" + uuid.NewString()
	w := worker.New(c, taskQueue, worker.Options{})
	if err := w.Start(); err != nil {
		return nil, fmt.Errorf("failed starting worker: %w", err)
	}
	callRespHandler, err := clientutil.NewCallResponseHandler(clientutil.CallResponseHandlerOptions{
		TaskQueue: taskQueue,
		Worker:    w,
	})
	if err != nil {
		return nil, fmt.Errorf("failed creating call response handler: %w", err)
	}
	return &Client{
		temporalClient: c,
		sqliteClient:   sqlitepb.NewClient(sqlitepb.ClientOptions{Client: c, CallResponseHandler: callRespHandler}),
		responseWorker: w,
		workflowID:     workflowID,
	}, nil
}

func (c *Client) GetRun(ctx context.Context) (sqlitepb.SqliteRun, error) {
	return c.sqliteClient.GetSqlite(ctx, c.workflowID, "")
}

func (c *Client) StopDB(ctx context.Context) error {
	return c.temporalClient.CancelWorkflow(ctx, c.workflowID, "")
}

func (c *Client) Close() {
	c.responseWorker.Stop()
}

func (c *Client) Update(ctx context.Context, stmt *Stmt) error {
	protoStmt, err := stmt.ToProto()
	if err != nil {
		return fmt.Errorf("invalid statement: %w", err)
	}
	return c.sqliteClient.Update(ctx, c.workflowID, "", &sqlitepb.UpdateRequest{Stmt: protoStmt})
}

func (c *Client) UpdateSimple(ctx context.Context, sql string, args ...interface{}) error {
	return c.Update(ctx, NewSimpleStmt(sql, args...))
}

func (c *Client) Exec(ctx context.Context, stmt *Stmt) (*StmtResult, error) {
	protoStmt, err := stmt.ToProto()
	if err != nil {
		return nil, fmt.Errorf("invalid statement: %w", err)
	}
	resp, err := c.sqliteClient.Exec(ctx, c.workflowID, "", &sqlitepb.ExecRequest{Stmt: protoStmt})
	if err != nil {
		return nil, err
	} else if respErr := resp.GetError(); respErr != nil {
		return nil, fmt.Errorf("exec failed with error code %v: %v", respErr.Code, respErr.Message)
	}
	var res StmtResult
	res.FromProto(resp.GetResult())
	return &res, nil
}

func (c *Client) ExecSimple(ctx context.Context, sql string, args ...interface{}) (*StmtResult, error) {
	return c.Exec(ctx, NewSimpleStmt(sql, args...))
}

// Queries can fail because of https://github.com/temporalio/temporal/issues/2300
func (c *Client) Query(ctx context.Context, stmt *Stmt) (*StmtResult, error) {
	protoStmt, err := stmt.ToProto()
	if err != nil {
		return nil, fmt.Errorf("invalid statement: %w", err)
	}
	resp, err := c.sqliteClient.Query(ctx, c.workflowID, "", &sqlitepb.QueryRequest{Stmt: protoStmt})
	if err != nil {
		return nil, err
	} else if respErr := resp.GetError(); respErr != nil {
		return nil, fmt.Errorf("query failed with error code %v: %v", respErr.Code, respErr.Message)
	}
	var res StmtResult
	res.FromProto(resp.GetResult())
	return &res, nil
}

func (c *Client) QuerySimple(ctx context.Context, sql string, args ...interface{}) (*StmtResult, error) {
	return c.Query(ctx, NewSimpleStmt(sql, args...))
}

func (c *Client) Serialize(ctx context.Context) ([]byte, error) {
	resp, err := c.sqliteClient.Serialize(ctx, c.workflowID, "")
	if err != nil {
		return nil, err
	}
	// Uncompress
	// TODO(cretz): Compression options?
	return io.ReadAll(lz4.NewReader(bytes.NewReader(resp.Serialized)))
}

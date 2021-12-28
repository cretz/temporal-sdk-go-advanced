package temporalsqlite

import (
	"context"
	"errors"
	"fmt"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"github.com/cretz/temporal-sdk-go-advanced/temporalutil/clientutil"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
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
	// Default to reject duplicate so if a workflow failed due to bad query, it
	// remains failed
	if opts.StartWorkflow.WorkflowIDReusePolicy == enums.WORKFLOW_ID_REUSE_POLICY_UNSPECIFIED {
		opts.StartWorkflow.WorkflowIDReusePolicy = enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
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

// Statement errors not returned and fail the workflow by default
func (c *Client) Update(ctx context.Context, stmts ...*Stmt) error {
	req, err := StmtsToProto(stmts)
	if err != nil {
		return err
	}
	return c.sqliteClient.Update(ctx, c.workflowID, "", &sqlitepb.UpdateRequest{Request: req})
}

// Wraps Update
func (c *Client) UpdateSimple(ctx context.Context, sql string, args ...interface{}) error {
	return c.Update(ctx, NewSingleStmt(sql, args...))
}

// Wraps Update
func (c *Client) UpdateMulti(ctx context.Context, multiSQL string) error {
	return c.Update(ctx, NewMultiStmt(multiSQL))
}

func (c *Client) Exec(ctx context.Context, stmts ...*Stmt) ([]*StmtResult, error) {
	req, err := StmtsToProto(stmts)
	if err != nil {
		return nil, err
	}
	resp, err := c.sqliteClient.Exec(ctx, c.workflowID, "", &sqlitepb.ExecRequest{Request: req})
	if err != nil {
		return nil, err
	}
	return StmtResultsFromProto(resp.Response), nil
}

func (c *Client) ExecSimple(ctx context.Context, sql string, args ...interface{}) (*StmtResultSuccess, error) {
	res, err := c.Exec(ctx, NewSingleStmt(sql, args...))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return nil, res[0].Error
	}
	return res[0].Successes[0], nil
}

func (c *Client) ExecMulti(ctx context.Context, multiSQL string) ([]*StmtResultSuccess, error) {
	res, err := c.Exec(ctx, NewMultiStmt(multiSQL))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return res[0].Successes, res[0].Error
	}
	return res[0].Successes, nil
}

// Queries can fail during workflow continue-as-new because of https://github.com/temporalio/temporal/issues/2300
func (c *Client) Query(ctx context.Context, stmts ...*Stmt) ([]*StmtResult, error) {
	req, err := StmtsToProto(stmts)
	if err != nil {
		return nil, err
	}
	resp, err := c.sqliteClient.Query(ctx, c.workflowID, "", &sqlitepb.QueryRequest{Request: req})
	if err != nil {
		return nil, err
	}
	return StmtResultsFromProto(resp.Response), nil
}

func (c *Client) QuerySimple(ctx context.Context, sql string, args ...interface{}) (*StmtResultSuccess, error) {
	res, err := c.Query(ctx, NewSingleStmt(sql, args...))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return nil, res[0].Error
	}
	return res[0].Successes[0], nil
}

func (c *Client) QueryMulti(ctx context.Context, multiSQL string) ([]*StmtResultSuccess, error) {
	res, err := c.Query(ctx, NewMultiStmt(multiSQL))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return res[0].Successes, res[0].Error
	}
	return res[0].Successes, nil
}

func (c *Client) Serialize(ctx context.Context) ([]byte, error) {
	resp, err := c.sqliteClient.Serialize(ctx, c.workflowID, "")
	if err != nil {
		return nil, err
	} else if err := resp.GetError(); err != "" {
		return nil, errors.New(err)
	}
	// Uncompress
	// TODO(cretz): Compression options?
	return uncompress(resp.GetSerialized())
}

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

// ConnectDBOptions are options for ConnectDB.
type ConnectDBOptions struct {
	// Workflow start options. This should have ID and TaskQueue always set.
	StartWorkflow client.StartWorkflowOptions
	// Optional SQLite workflow configuration.
	Sqlite *sqlitepb.SqliteOptions
}

// ConnectDB starts the DB workflow with the given opts.StartWorkflow.ID unless
// it already exists and then uses NewClient to return a client. This means the
// default opts.StartWorkflow.WorkflowIDReusePolicy is
// WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE if otherwise unset. Callers may
// want to change this to WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY
// to support restarting a failed DB workflow.
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

// Client is a client for a DB workflow.
type Client struct {
	temporalClient client.Client
	sqliteClient   sqlitepb.Client
	responseWorker worker.Worker
	workflowID     string
}

// NewClient references an existing DB by its workflow ID. Use ConnectDB to
// potentially start the DB if it doesn't already exist. This also internally
// starts a worker for handling "Exec" responses and therefore Close should be
// called by the caller.
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

// GetRun returns the current workflow run for this DB.
func (c *Client) GetRun(ctx context.Context) (sqlitepb.SqliteRun, error) {
	return c.sqliteClient.GetSqlite(ctx, c.workflowID, "")
}

// StopDB cancels the DB workflow.
func (c *Client) StopDB(ctx context.Context) error {
	return c.temporalClient.CancelWorkflow(ctx, c.workflowID, "")
}

// Close closes this client.
func (c *Client) Close() {
	c.responseWorker.Stop()
}

// Update sends an update signal to the workflow. This returns no response and
// there is generally no way to know if it succeeded. By default, if this query
// fails, the entire workflow fails. Most callers may prefer Exec or Query
// instead.
//
// Since this uses a signal, it cannot be executed on a stopped database.
func (c *Client) Update(ctx context.Context, stmts ...*Stmt) error {
	req, err := StmtsToProto(stmts)
	if err != nil {
		return err
	}
	return c.sqliteClient.Update(ctx, c.workflowID, "", &sqlitepb.UpdateRequest{Request: req})
}

// UpdateSimple is a convenience shortcut for Update.
func (c *Client) UpdateSimple(ctx context.Context, sql string, args ...interface{}) error {
	return c.Update(ctx, NewSingleStmt(sql, args...))
}

// UpdateMulti is a convenience shortcut for Update.
func (c *Client) UpdateMulti(ctx context.Context, multiSQL string) error {
	return c.Update(ctx, NewMultiStmt(multiSQL))
}

// Exec sends an exec signal to the workflow and waits for the response to be
// reported as an activity.
//
// Since this uses a signal, it cannot be executed on a stopped database.
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

// ExecSimple is a convenience shortcut for Exec.
func (c *Client) ExecSimple(ctx context.Context, sql string, args ...interface{}) (*StmtResultSuccess, error) {
	res, err := c.Exec(ctx, NewSingleStmt(sql, args...))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return nil, res[0].Error
	}
	return res[0].Successes[0], nil
}

// ExecMulti is a convenience shortcut for Exec.
func (c *Client) ExecMulti(ctx context.Context, multiSQL string) ([]*StmtResultSuccess, error) {
	res, err := c.Exec(ctx, NewMultiStmt(multiSQL))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return res[0].Successes, res[0].Error
	}
	return res[0].Successes, nil
}

// Query sends a read-only query to the workflow. This query must be read-only
// or it will fail.
//
// Since this uses a query, it can be executed even if the database is stopped.
//
// NOTE: Currently there are rare racy cases where a query can fail when sent
// while it is restarting (i.e. performing continue-as-new). See
// https://github.com/temporalio/sdk-go/issues/475 which is waiting on
// https://github.com/temporalio/temporal/issues/2300.
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

// QuerySimple is a convenience shortcut for Query.
func (c *Client) QuerySimple(ctx context.Context, sql string, args ...interface{}) (*StmtResultSuccess, error) {
	res, err := c.Query(ctx, NewSingleStmt(sql, args...))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return nil, res[0].Error
	}
	return res[0].Successes[0], nil
}

// QueryMulti is a convenience shortcut for Query.
func (c *Client) QueryMulti(ctx context.Context, multiSQL string) ([]*StmtResultSuccess, error) {
	res, err := c.Query(ctx, NewMultiStmt(multiSQL))
	if err != nil {
		return nil, err
	} else if res[0].Error != nil {
		return res[0].Successes, res[0].Error
	}
	return res[0].Successes, nil
}

// Serialize sends a query to return a LZ4-compressed set of bytes generated via
// https://www.sqlite.org/c3ref/serialize.html.
//
// Since this uses a query, it can be executed even if the database is stopped.
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

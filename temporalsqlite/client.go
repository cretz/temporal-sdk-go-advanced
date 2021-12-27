package temporalsqlite

import (
	"context"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"go.temporal.io/sdk/client"
)

type ConnectDBOptions struct {
	client.StartWorkflowOptions
	sqlitepb.SqliteOptions
}

func ConnectDB(ctx context.Context, c client.Client, opts ConnectDBOptions) (*Client, error) {
	panic("TODO")
}

type Client struct {
}

func NewClient(c *client.Client, workflowID string) *Client {
	panic("TODO")
}

func (c *Client) GetRun(context.Context) (sqlitepb.SqliteRun, error) {
	panic("TODO")
}

func (c *Client) StopDB(context.Context) error {
	panic("TODO")
}

func (c *Client) Close() error {
	panic("TODO")
}

type Stmt struct {
	Query     string
	Args      []interface{}
	NamedArgs map[string]interface{}
}

type StmtResult struct {
	ColumnNames []string
	Rows        [][]interface{}
}

func (c *Client) Update(context.Context, *Stmt) error {
	panic("TODO")
}

func (c *Client) Exec(context.Context, *Stmt) (*StmtResult, error) {
	panic("TODO")
}

func (c *Client) Query(context.Context, *Stmt) (*StmtResult, error) {
	panic("TODO")
}

func (c *Client) Serialize(context.Context) ([]byte, error) {
	panic("TODO")
}

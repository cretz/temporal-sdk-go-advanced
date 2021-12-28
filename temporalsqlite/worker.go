package temporalsqlite

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"github.com/pierrec/lz4/v4"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const defaultStatementsUntilContinueAsNew = 5000

type SqliteWorkerOptions struct {
	CompressionOptions []lz4.Option
}

func RegisterSqliteWorker(r worker.WorkflowRegistry, opts SqliteWorkerOptions) {
	sqlitepb.RegisterSqlite(r, opts.newSqliteImpl)
}

type sqliteImpl struct {
	SqliteWorkerOptions
	*sqlitepb.SqliteInput
	log            log.Logger
	sel            workflow.Selector
	db             *db
	statementCount uint32
}

func (s *SqliteWorkerOptions) newSqliteImpl(
	ctx workflow.Context,
	in *sqlitepb.SqliteInput,
) (sqlitepb.SqliteImpl, error) {
	if in.Req == nil {
		in.Req = &sqlitepb.SqliteOptions{}
	}
	if in.Req.StatementsUntilContinueAsNew == 0 {
		in.Req.StatementsUntilContinueAsNew = defaultStatementsUntilContinueAsNew
	}
	return &sqliteImpl{
		SqliteWorkerOptions: *s,
		SqliteInput:         in,
		log:                 workflow.GetLogger(ctx),
		sel:                 workflow.NewSelector(ctx),
	}, nil
}

func (s *sqliteImpl) Run(ctx workflow.Context) error {
	// Start DB with previous serialized state if present
	var err error
	s.log.Debug("Opening database")
	if len(s.Req.Serialized) > 0 {
		if s.Req.Serialized, err = s.uncompress(s.Req.Serialized); err != nil {
			return err
		}
	}
	s.db, err = openDB(s.Req.Serialized)
	s.Req.Serialized = nil
	if err != nil {
		return err
	}
	defer s.db.close()

	// Add select handlers for update and exec
	s.Exec.Select(s.sel, func(req *sqlitepb.ExecRequest) { s.exec(ctx, req) })
	s.Update.Select(s.sel, func(req *sqlitepb.UpdateRequest) { s.update(req) })
	// Add handler for context done
	s.sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {})

	// Continually handle selects until context done or continue-as-new limit
	// reached with no more selects pending
	for ctx.Err() == nil && (s.statementCount < s.Req.StatementsUntilContinueAsNew || s.sel.HasPending()) {
		s.sel.Select(ctx)
	}
	// Return error if cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Do a continue-as-new by serializing then compressing the data
	if s.Req.Serialized, err = s.db.serialize(); err != nil {
		return err
	} else if s.Req.Serialized, err = s.compress(s.Req.Serialized); err != nil {
		return err
	}
	s.log.Debug("Serialized DB and returning continue-as-new", "serializedSize", len(s.Req.Serialized))
	return workflow.NewContinueAsNewError(ctx, sqlitepb.SqliteName, s.Req)
}

func (s *sqliteImpl) Query(req *sqlitepb.QueryRequest) (*sqlitepb.QueryResponse, error) {
	// Does not increment statement count of course since queries do not have any
	// side effects
	s.log.Debug("Query requested", "sql", req.Stmt.GetSql())
	res, err := s.db.exec(req.Stmt, true, true)
	var resp sqlitepb.QueryResponse
	if err != nil {
		s.log.Warn("Query failed", "error", err)
		resp.Response = &sqlitepb.QueryResponse_Error{Error: err}
	} else {
		s.log.Debug("Query succeeded", "rowCount", len(res.Rows))
		resp.Response = &sqlitepb.QueryResponse_Result{Result: res}
	}
	return &resp, nil
}

func (s *sqliteImpl) update(req *sqlitepb.UpdateRequest) {
	s.statementCount++
	s.log.Debug("Update requested", "sql", req.Stmt.GetSql(), "statementCount", s.statementCount)
	if _, err := s.db.exec(req.Stmt, false, false); err != nil {
		s.log.Warn("Update failed", "error", err)
	} else {
		s.log.Debug("Update succeeded")
	}
}

// Only give 10 seconds for response, no retry
var execResponseOpts = &workflow.ActivityOptions{
	ScheduleToCloseTimeout: 10 * time.Second,
	RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
}

func (s *sqliteImpl) exec(ctx workflow.Context, req *sqlitepb.ExecRequest) {
	s.statementCount++
	s.log.Debug("Exec requested", "sql", req.Stmt.GetSql(), "statementCount", s.statementCount)
	res, err := s.db.exec(req.Stmt, false, true)
	var resp sqlitepb.ExecResponse
	if err != nil {
		s.log.Warn("Exec failed", "error", err)
		resp.Response = &sqlitepb.ExecResponse_Error{Error: err}
	} else {
		s.log.Debug("Exec succeeded", "rowCount", len(res.Rows))
		resp.Response = &sqlitepb.ExecResponse_Result{Result: res}
	}
	// Send response with no retry
	s.sel.AddFuture(s.Exec.Respond(ctx, execResponseOpts, req, &resp), func(workflow.Future) {})
}

func (s *sqliteImpl) Serialize() (*sqlitepb.SerializeResponse, error) {
	var resp sqlitepb.SerializeResponse
	var err error
	if resp.Serialized, err = s.db.serialize(); err != nil {
		return nil, err
	} else if s.Req.Serialized, err = s.compress(resp.Serialized); err != nil {
		return nil, err
	}
	s.log.Debug("Serialized DB", "serializedSize", len(resp.Serialized))
	return &resp, nil
}

func (s *sqliteImpl) compress(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if err := w.Apply(s.CompressionOptions...); err != nil {
		w.Close()
		return nil, fmt.Errorf("bad compression options: %w", err)
	} else if _, err := w.Write(b); err != nil {
		w.Close()
		return nil, fmt.Errorf("compression failed: %w", err)
	} else if err = w.Close(); err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *sqliteImpl) uncompress(b []byte) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(b))
	if err := r.Apply(s.CompressionOptions...); err != nil {
		return nil, fmt.Errorf("bad compression options: %w", err)
	}
	return io.ReadAll(r)
}

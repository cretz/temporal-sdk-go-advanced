package temporalsqlite

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"time"

	"crawshaw.io/sqlite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"github.com/pierrec/lz4/v4"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const DefaultRequestsUntilContinueAsNew = 5000

type SqliteWorkerOptions struct {
	CompressionOptions []lz4.Option

	// Used if not in workflow params. Default is
	// DefaultRequestsUntilContinueAsNew.
	DefaultRequestsUntilContinueAsNew int

	// If true, all queries and their successes are logged at debug level
	LogQueries bool

	// If true, all actions of each query are logged at debug level
	LogActions bool

	InitializeSqlite func(*sqlite.Conn) error

	// Should be all lowercase. If empty, default is DefaultNonDeterministicFuncs.
	NonDeterministicFuncs map[string]bool

	// If true, errors from signal-based updates do not fail the workflow.
	IgnoreUpdateErrors bool
}

func RegisterSqliteWorker(r worker.WorkflowRegistry, opts SqliteWorkerOptions) {
	if opts.DefaultRequestsUntilContinueAsNew == 0 {
		opts.DefaultRequestsUntilContinueAsNew = DefaultRequestsUntilContinueAsNew
	}
	if len(opts.NonDeterministicFuncs) == 0 {
		opts.NonDeterministicFuncs = DefaultNonDeterministicFuncs
	}
	sqlitepb.RegisterSqlite(r, opts.newSqliteImpl)
}

type sqliteImpl struct {
	SqliteWorkerOptions
	*sqlitepb.SqliteInput
	log          log.Logger
	sel          workflow.Selector
	db           *db
	requestCount uint32
}

func (s *SqliteWorkerOptions) newSqliteImpl(
	ctx workflow.Context,
	in *sqlitepb.SqliteInput,
) (sqlitepb.SqliteImpl, error) {
	if in.Req == nil {
		in.Req = &sqlitepb.SqliteOptions{}
	}
	if in.Req.RequestsUntilContinueAsNew == 0 {
		in.Req.RequestsUntilContinueAsNew = uint32(s.DefaultRequestsUntilContinueAsNew)
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
		if s.Req.Serialized, err = uncompress(s.Req.Serialized, s.CompressionOptions...); err != nil {
			return err
		}
	}
	s.db, err = openDB(dbOptions{
		log:                   s.log,
		logQueries:            s.LogQueries,
		logActions:            s.LogActions,
		prevSerialized:        s.Req.Serialized,
		nonDeterministicFuncs: s.NonDeterministicFuncs,
	})
	s.Req.Serialized = nil
	if err != nil {
		return err
	}
	// We set the DB to close on GC collection, not on completion of this
	// function. If we close the DB on function completion, then queries cannot
	// run after workflow complete.
	runtime.SetFinalizer(s.db, func(d *db) { d.close() })
	if s.InitializeSqlite != nil {
		if err := s.InitializeSqlite(s.db.conn); err != nil {
			return err
		}
	}

	// Add select handlers for update and exec
	s.Exec.Select(s.sel, func(req *sqlitepb.ExecRequest) { s.exec(ctx, req) })
	var lastUpdateErr error
	s.Update.Select(s.sel, func(req *sqlitepb.UpdateRequest) { lastUpdateErr = s.update(req) })
	// Add handler for context done
	s.sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) {})

	// Continually handle selects until context done or continue-as-new limit
	// reached with no more selects pending
	for ctx.Err() == nil && (s.requestCount < s.Req.RequestsUntilContinueAsNew || s.sel.HasPending()) {
		s.sel.Select(ctx)
		// If there is an update error and we're not ignoring, it fails the workflow
		if lastUpdateErr != nil && !s.IgnoreUpdateErrors {
			s.log.Error("Failing workflow because update failed", "error", lastUpdateErr)
			return fmt.Errorf("failing workflow because update failed with: %w", lastUpdateErr)
		}
	}
	// Return error if cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Do a continue-as-new by serializing then compressing the data
	if s.Req.Serialized, err = s.db.serialize(); err != nil {
		return err
	} else if s.Req.Serialized, err = compress(s.Req.Serialized, s.CompressionOptions...); err != nil {
		return err
	}
	s.log.Debug("Serialized DB and returning continue-as-new", "serializedSize", len(s.Req.Serialized))
	return workflow.NewContinueAsNewError(ctx, sqlitepb.SqliteName, s.Req)
}

func (s *sqliteImpl) Query(req *sqlitepb.QueryRequest) (*sqlitepb.QueryResponse, error) {
	// Does not increment request count of course since queries do not have any
	// side effects
	return &sqlitepb.QueryResponse{Response: s.db.execAll(req.Request, true, true)}, nil
}

func (s *sqliteImpl) update(req *sqlitepb.UpdateRequest) error {
	s.requestCount++
	// Ignore response
	resp := s.db.execAll(req.Request, false, false)
	// Return the first error found
	for _, res := range resp.Results {
		if res.Error != nil {
			return &StmtResultError{Code: int(res.Error.Code), Message: res.Error.Message}
		}
	}
	return nil
}

// Only give 10 seconds for response, no retry
var execResponseOpts = &workflow.ActivityOptions{
	ScheduleToCloseTimeout: 10 * time.Second,
	RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
}

func (s *sqliteImpl) exec(ctx workflow.Context, req *sqlitepb.ExecRequest) {
	s.requestCount++
	res := s.db.execAll(req.Request, false, true)
	// Send response with no retry
	s.sel.AddFuture(
		s.Exec.Respond(ctx, execResponseOpts, req, &sqlitepb.ExecResponse{Response: res}),
		func(workflow.Future) {},
	)
}

func (s *sqliteImpl) Serialize() (*sqlitepb.SerializeResponse, error) {
	b, err := s.db.serialize()
	if err == nil {
		b, err = compress(b, s.CompressionOptions...)
	}
	if err != nil {
		return &sqlitepb.SerializeResponse{Result: &sqlitepb.SerializeResponse_Error{Error: err.Error()}}, nil
	}
	s.log.Debug("Serialized DB", "serializedSize", len(b))
	return &sqlitepb.SerializeResponse{Result: &sqlitepb.SerializeResponse_Serialized{Serialized: b}}, nil
}

func compress(b []byte, opts ...lz4.Option) ([]byte, error) {
	var buf bytes.Buffer
	w := lz4.NewWriter(&buf)
	if err := w.Apply(opts...); err != nil {
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

func uncompress(b []byte, opts ...lz4.Option) ([]byte, error) {
	r := lz4.NewReader(bytes.NewReader(b))
	if err := r.Apply(opts...); err != nil {
		return nil, fmt.Errorf("bad compression options: %w", err)
	}
	return io.ReadAll(r)
}

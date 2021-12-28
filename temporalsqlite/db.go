package temporalsqlite

import (
	"fmt"
	"strings"

	"crawshaw.io/sqlite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
	"go.temporal.io/sdk/log"
)

var DefaultNonDeterministicFuncs = map[string]bool{
	"current_date":      true,
	"current_time":      true,
	"current_timestamp": true,
	"random":            true,
	"random_blob":       true,
	// TODO(cretz): For datetime('now') and the like, we could replace the
	// function with our own impl that forwarded to strftime _after_ confirming
	// that "now" isn't used.
}

var allowedReadOnlyActions = map[sqlite.OpType]bool{
	sqlite.SQLITE_READ:     true,
	sqlite.SQLITE_FUNCTION: true,
	sqlite.SQLITE_SELECT:   true,
}

type db struct {
	dbOptions
	conn                          *sqlite.Conn
	prevSer                       *sqlite.Serialized
	stmtFirstMutateOp             sqlite.OpType
	stmtFirstNonDeterministicFunc string
}

type dbOptions struct {
	log                   log.Logger
	logQueries            bool
	logActions            bool
	prevSerialized        []byte
	nonDeterministicFuncs map[string]bool
}

func openDB(opts dbOptions) (*db, error) {
	if opts.log == nil {
		return nil, fmt.Errorf("missing log")
	}

	// TODO(cretz): sqlite3_config(SQLITE_CONFIG_SINGLETHREAD)?
	// TODO(cretz): Replace/disable non-deterministic functions
	conn, err := sqlite.OpenConn(":memory:", 0)
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			conn.Close()
		}
	}()

	db := &db{dbOptions: opts, conn: conn}

	// If there is a previously serialized DB, use it, making sure to copy over to
	// sqlite so it can resize as necessary
	if len(opts.prevSerialized) > 0 {
		// We must store this on the DB since we need a reference to it while we're
		// using the DB
		db.prevSer = sqlite.NewSerialized("", opts.prevSerialized, true)
		err = db.conn.Deserialize(db.prevSer, sqlite.SQLITE_DESERIALIZE_FREEONCLOSE|sqlite.SQLITE_DESERIALIZE_RESIZEABLE)
		if err != nil {
			return nil, fmt.Errorf("unable to deserialize DB: %w", err)
		}
		opts.prevSerialized = nil
	}

	// Set the authorize func
	db.conn.SetAuthorizer(sqlite.AuthorizeFunc(func(info sqlite.ActionInfo) sqlite.AuthResult {
		if db.logActions {
			db.log.Debug("Query action", "action", info)
		}

		// Check if action is not read-only
		if db.stmtFirstMutateOp == 0 && !allowedReadOnlyActions[info.Action] {
			db.stmtFirstMutateOp = info.Action
		}
		updateNonDet := info.Action == sqlite.SQLITE_FUNCTION &&
			db.stmtFirstNonDeterministicFunc == "" &&
			db.nonDeterministicFuncs[strings.ToLower(info.Function)]
		if updateNonDet {
			db.stmtFirstNonDeterministicFunc = info.Function
		}
		return 0
	}))

	success = true
	return db, nil
}

func (d *db) close() {
	if d.conn != nil {
		d.conn.Close()
		d.conn = nil
	}
	d.prevSer = nil
}

func (d *db) execAll(req *sqlitepb.StmtRequest, readOnly bool, includeResult bool) *sqlitepb.StmtResponse {
	res := &sqlitepb.StmtResponse{}
	for _, s := range req.Stmts {
		res.Results = append(res.Results, d.exec(s, readOnly, includeResult))
		if res.Results[len(res.Results)-1].Error == nil {
			break
		}
	}
	return res
}

func (d *db) exec(s *sqlitepb.Stmt, readOnly bool, includeResult bool) (res *sqlitepb.StmtResult) {
	res = &sqlitepb.StmtResult{}
	if s == nil || s.Sql == "" {
		res.Error = &sqlitepb.StmtResult_Error{Message: "missing sql"}
		return
	} else if s.MultiQuery && (len(s.IndexedParams) != 0 || len(s.NamedParams) != 0) {
		res.Error = &sqlitepb.StmtResult_Error{Message: "multi-query cannot have params"}
		return
	}

	// We have to loop in case it's a multi-query
	sql := strings.TrimSpace(s.Sql)
	for sql != "" {
		if d.logQueries {
			d.log.Debug("Exec SQL", "sql", sql, "readOnly", readOnly, "includeResult", includeResult)
		}
		var ok *sqlitepb.StmtResult_Success
		ok, sql, res.Error = d.execSingle(s, sql, readOnly, includeResult)
		if res.Error != nil {
			d.log.Warn("Exec failed", "error", res.Error)
			break
		}
		if d.logQueries {
			d.log.Debug("Exec succeeded", "rowCount", len(ok.Rows))
		}
		res.Successes = append(res.Successes, ok)
	}
	return
}

func (d *db) execSingle(
	s *sqlitepb.Stmt,
	sql string,
	readOnly bool,
	includeResult bool,
) (ok *sqlitepb.StmtResult_Success, remainingSQL string, resErr *sqlitepb.StmtResult_Error) {
	// Reset values so they can be fetched during prepare
	d.stmtFirstMutateOp = 0
	d.stmtFirstNonDeterministicFunc = ""

	// Prepare statement
	stmt, trailingBytes, err := d.conn.PrepareTransient(sql)
	if err != nil {
		return nil, "", convertStmtError(err)
	}
	defer func() {
		err := stmt.Finalize()
		if err != nil && resErr == nil {
			resErr = convertStmtError(err)
		}
	}()

	// Check read only
	// TODO(cretz): Could just expose https://www.sqlite.org/c3ref/stmt_readonly.html
	if readOnly && d.stmtFirstMutateOp > 0 {
		return nil, "", &sqlitepb.StmtResult_Error{
			Message: fmt.Sprintf("statement expected to be read only but got action %v", d.stmtFirstMutateOp),
		}
	}

	// Check non-deterministic function
	if d.stmtFirstNonDeterministicFunc != "" {
		return nil, "", &sqlitepb.StmtResult_Error{
			Message: fmt.Sprintf("statement called non-deterministic function %v", d.stmtFirstNonDeterministicFunc),
		}
	}

	// Check trailing bytes
	if trailingBytes > 0 {
		if !s.MultiQuery {
			return nil, "", &sqlitepb.StmtResult_Error{
				Message: "expected single statement since multi-query not set, but got multiple",
			}
		}
		remainingSQL = strings.TrimSpace(sql[len(sql)-trailingBytes:])
	}

	// Bind params
	if err := bindParams(stmt, s); err != nil {
		return nil, "", convertStmtError(err)
	}

	// Prepare result
	ok = &sqlitepb.StmtResult_Success{}
	if includeResult {
		ok.Columns = make([]*sqlitepb.StmtResult_Column, stmt.ColumnCount())
		for i := range ok.Columns {
			ok.Columns[i] = &sqlitepb.StmtResult_Column{Name: stmt.ColumnName(i)}
		}
	}

	// Step over all rows
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, "", convertStmtError(err)
		} else if !hasRow {
			return
		}
		// Only populate results if wanted
		if includeResult {
			row, err := stmtRow(stmt, ok.Columns)
			if err != nil {
				return nil, "", convertStmtError(err)
			}
			ok.Rows = append(ok.Rows, row)
		}
	}
}

func (d *db) serialize() ([]byte, error) {
	var flags sqlite.SerializeFlags
	// We can use no-copy if we came from a previous serialized value
	if d.prevSer != nil {
		flags |= sqlite.SQLITE_SERIALIZE_NOCOPY
	}
	ser := d.conn.Serialize("", flags)
	if ser == nil {
		return nil, fmt.Errorf("failure serializing")
	}
	// We choose to copy the bytes here instead of keeping track of when they'll
	// be needed later
	res := make([]byte, len(ser.Bytes()))
	copy(res, ser.Bytes())
	return res, nil
}

func convertStmtError(err error) *sqlitepb.StmtResult_Error {
	return &sqlitepb.StmtResult_Error{Code: int64(sqlite.ErrCode(err)), Message: err.Error()}
}

func bindParams(stmt *sqlite.Stmt, s *sqlitepb.Stmt) error {
	for k, v := range s.IndexedParams {
		k := int(k)
		switch v := v.Value.(type) {
		case nil:
		case *sqlitepb.Value_NullValue:
			stmt.BindNull(k)
		case *sqlitepb.Value_IntValue:
			stmt.BindInt64(k, v.IntValue)
		case *sqlitepb.Value_FloatValue:
			stmt.BindFloat(k, v.FloatValue)
		case *sqlitepb.Value_StringValue:
			stmt.BindText(k, v.StringValue)
		case *sqlitepb.Value_BytesValue:
			stmt.BindBytes(k, v.BytesValue)
		default:
			return fmt.Errorf("unrecognized param type %T", v)
		}
	}
	for k, v := range s.NamedParams {
		switch v := v.Value.(type) {
		case nil:
		case *sqlitepb.Value_NullValue:
			stmt.SetNull(k)
		case *sqlitepb.Value_IntValue:
			stmt.SetInt64(k, v.IntValue)
		case *sqlitepb.Value_FloatValue:
			stmt.SetFloat(k, v.FloatValue)
		case *sqlitepb.Value_StringValue:
			stmt.SetText(k, v.StringValue)
		case *sqlitepb.Value_BytesValue:
			stmt.SetBytes(k, v.BytesValue)
		default:
			return fmt.Errorf("unrecognized param type %T", v)
		}
	}
	return nil
}

func stmtRow(stmt *sqlite.Stmt, cols []*sqlitepb.StmtResult_Column) (*sqlitepb.StmtResult_Row, error) {
	row := &sqlitepb.StmtResult_Row{Values: make([]*sqlitepb.Value, len(cols))}
	for i := range row.Values {
		switch t := stmt.ColumnType(i); t {
		case sqlite.SQLITE_NULL:
			row.Values[i] = &sqlitepb.Value{Value: &sqlitepb.Value_NullValue{NullValue: true}}
		case sqlite.SQLITE_INTEGER:
			row.Values[i] = &sqlitepb.Value{Value: &sqlitepb.Value_IntValue{IntValue: stmt.ColumnInt64(i)}}
		case sqlite.SQLITE_FLOAT:
			row.Values[i] = &sqlitepb.Value{Value: &sqlitepb.Value_FloatValue{FloatValue: stmt.ColumnFloat(i)}}
		case sqlite.SQLITE_TEXT:
			row.Values[i] = &sqlitepb.Value{Value: &sqlitepb.Value_StringValue{StringValue: stmt.ColumnText(i)}}
		case sqlite.SQLITE_BLOB:
			b := make([]byte, stmt.ColumnLen(i))
			stmt.ColumnBytes(i, b)
			row.Values[i] = &sqlitepb.Value{Value: &sqlitepb.Value_BytesValue{BytesValue: b}}
		default:
			return nil, fmt.Errorf("unrecognized column type %v", t)
		}
	}
	return row, nil
}

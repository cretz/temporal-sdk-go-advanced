package temporalsqlite

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
)

type db struct {
	conn              *sqlite.Conn
	prevSer           *sqlite.Serialized
	stmtFirstMutateOp sqlite.OpType
}

func openDB(prevSerialized []byte) (*db, error) {
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

	db := &db{conn: conn}

	// If there is a previously serialized DB, use it, making sure to copy over to
	// sqlite so it can resize as necessary
	if len(prevSerialized) > 0 {
		// We must store this on the DB since we need a reference to it while we're
		// using the DB
		db.prevSer = sqlite.NewSerialized("", prevSerialized, true)
		err = db.conn.Deserialize(db.prevSer, sqlite.SQLITE_DESERIALIZE_FREEONCLOSE|sqlite.SQLITE_DESERIALIZE_RESIZEABLE)
		if err != nil {
			return nil, fmt.Errorf("unable to deserialize DB: %w", err)
		}
	}

	// Set the authorize func
	db.conn.SetAuthorizer(sqlite.AuthorizeFunc(func(info sqlite.ActionInfo) sqlite.AuthResult {
		if db.stmtFirstMutateOp == 0 && info.Action != sqlite.SQLITE_READ && info.Action != sqlite.SQLITE_SELECT {
			db.stmtFirstMutateOp = info.Action
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

func (d *db) exec(
	s *sqlitepb.Stmt,
	readOnly bool,
	includeResult bool,
) (stmtRes *sqlitepb.StmtResult, stmtErr *sqlitepb.StmtError) {
	if s == nil || s.Sql == "" {
		return nil, stmtErrorf("missing sql")
	}
	// Reset the mutate op so it can be fetched during prepare
	d.stmtFirstMutateOp = 0

	// Prepare statement
	stmt, _, err := d.conn.PrepareTransient(s.Sql)
	if err != nil {
		return nil, convertStmtError(err)
	}
	defer func() {
		err := stmt.Finalize()
		if err != nil && stmtErr == nil {
			stmtErr = convertStmtError(err)
		}
	}()

	// Check read only
	if readOnly && d.stmtFirstMutateOp > 0 {
		return nil, stmtErrorf("statement expected to be read only but got action %v", d.stmtFirstMutateOp)
	}

	// Bind parameters
	if err := bindParams(stmt, s); err != nil {
		return nil, err
	}

	// Prepare result
	if includeResult {
		stmtRes = &sqlitepb.StmtResult{Columns: make([]*sqlitepb.StmtResult_Column, stmt.ColumnCount())}
		for i := range stmtRes.Columns {
			stmtRes.Columns[i] = &sqlitepb.StmtResult_Column{Name: stmt.ColumnName(i)}
		}
	}

	// Step over all rows
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, convertStmtError(err)
		} else if !hasRow {
			return stmtRes, nil
		}
		// Only populate results if wanted
		if stmtRes != nil {
			row, err := stmtRow(stmt, stmtRes.Columns)
			if err != nil {
				return nil, err
			}
			stmtRes.Rows = append(stmtRes.Rows, row)
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

func convertStmtError(err error) *sqlitepb.StmtError {
	return &sqlitepb.StmtError{Code: int64(sqlite.ErrCode(err)), Message: err.Error()}
}

func stmtErrorf(f string, v ...interface{}) *sqlitepb.StmtError {
	return &sqlitepb.StmtError{Message: fmt.Sprintf(f, v...)}
}

func bindParams(stmt *sqlite.Stmt, s *sqlitepb.Stmt) *sqlitepb.StmtError {
	for k, v := range s.IndexedParams {
		k := int(k)
		switch v := v.Kind.(type) {
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
			return stmtErrorf("unrecognized param type %T", v)
		}
	}
	for k, v := range s.NamedParams {
		switch v := v.Kind.(type) {
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
			return stmtErrorf("unrecognized param type %T", v)
		}
	}
	return nil
}

func stmtRow(stmt *sqlite.Stmt, cols []*sqlitepb.StmtResult_Column) (*sqlitepb.StmtResult_Row, *sqlitepb.StmtError) {
	row := &sqlitepb.StmtResult_Row{Values: make([]*sqlitepb.Value, len(cols))}
	for i := range row.Values {
		switch t := stmt.ColumnType(i); t {
		case sqlite.SQLITE_NULL:
			row.Values[i] = &sqlitepb.Value{Kind: &sqlitepb.Value_NullValue{NullValue: true}}
		case sqlite.SQLITE_INTEGER:
			row.Values[i] = &sqlitepb.Value{Kind: &sqlitepb.Value_IntValue{IntValue: stmt.ColumnInt64(i)}}
		case sqlite.SQLITE_FLOAT:
			row.Values[i] = &sqlitepb.Value{Kind: &sqlitepb.Value_FloatValue{FloatValue: stmt.ColumnFloat(i)}}
		case sqlite.SQLITE_TEXT:
			row.Values[i] = &sqlitepb.Value{Kind: &sqlitepb.Value_StringValue{StringValue: stmt.ColumnText(i)}}
		case sqlite.SQLITE_BLOB:
			b := make([]byte, stmt.ColumnLen(i))
			stmt.ColumnBytes(i, b)
			row.Values[i] = &sqlitepb.Value{Kind: &sqlitepb.Value_BytesValue{BytesValue: b}}
		default:
			return nil, stmtErrorf("unrecognized column type %v", t)
		}
	}
	return row, nil
}

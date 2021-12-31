package temporalsqlite

import (
	"fmt"
	"math"
	"reflect"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
)

// Stmt represents a statement used by clients.
type Stmt struct {
	Query string

	// Param indexes start at 1. Values can be null, bool, integer, float, string,
	// byte slice, or pointer to any of those.
	IndexedParams map[int]interface{}
	NamedParams   map[string]interface{}

	// If true, Query can contain multiple semicolon-delimited queries. The
	// statement cannot have any parameters set if this value true. If false,
	// Query can only contain a single query.
	Multi bool
}

// NewSingleStmt creates a new single-query statement with the given indexed
// params.
func NewSingleStmt(query string, params ...interface{}) *Stmt {
	s := &Stmt{Query: query, IndexedParams: make(map[int]interface{}, len(params))}
	for i, v := range params {
		s.IndexedParams[i+1] = v
	}
	return s
}

// NewMultiStmt creates a new multi-query statement.
func NewMultiStmt(query string) *Stmt {
	return &Stmt{Query: query, Multi: true}
}

// ToProto converts this statement to its protobuf equivalent.
func (s *Stmt) ToProto() (*sqlitepb.Stmt, error) {
	stmt := &sqlitepb.Stmt{
		Sql:           s.Query,
		IndexedParams: make(map[uint32]*sqlitepb.Value, len(s.IndexedParams)),
		NamedParams:   make(map[string]*sqlitepb.Value, len(s.NamedParams)),
		MultiQuery:    s.Multi,
	}
	for k, v := range s.IndexedParams {
		v, err := ToProtoValue(v)
		if err != nil {
			return nil, fmt.Errorf("invalid param #%v: %w", k, err)
		}
		stmt.IndexedParams[uint32(k)] = v
	}
	for k, v := range s.NamedParams {
		v, err := ToProtoValue(v)
		if err != nil {
			return nil, fmt.Errorf("invalid param %q: %w", k, err)
		}
		stmt.NamedParams[k] = v
	}
	return stmt, nil
}

// StmtsToProto converts these statements to a protobuf request.
func StmtsToProto(stmts []*Stmt) (*sqlitepb.StmtRequest, error) {
	if len(stmts) == 0 {
		return nil, fmt.Errorf("no statements")
	}
	req := &sqlitepb.StmtRequest{Stmts: make([]*sqlitepb.Stmt, len(stmts))}
	for i, stmt := range stmts {
		var err error
		if req.Stmts[i], err = stmt.ToProto(); err != nil {
			return nil, fmt.Errorf("statement invalid: %w", err)
		}
	}
	return req, nil
}

// StmtResult represents a result of a client statement.
type StmtResult struct {
	// All successes that occurred. This will only contain the successes up until
	// the first error. This only has multiple values if Stmt.Multi was true.
	Successes []*StmtResultSuccess

	// The first error encountered (and last error since an error stops
	// execution).
	Error *StmtResultError
}

// StmtResultSuccess represents a single query success (of which a statement
// can have multiple).
type StmtResultSuccess struct {
	// Column names for the query result.
	ColumnNames []string
	// Values can be nil, int64, float64, string, or bytes.
	Rows [][]interface{}
}

// StmtResultError represents an error from a statement.
type StmtResultError struct {
	Code    int
	Message string
}

// Error implements error.Error.
func (s *StmtResultError) Error() string {
	if s.Code == 0 {
		return s.Message
	}
	return fmt.Sprintf("%v (code %v)", s.Message, s.Code)
}

// FromProto converts the given protobuf result to this value.
func (s *StmtResult) FromProto(res *sqlitepb.StmtResult) {
	s.Successes = make([]*StmtResultSuccess, len(res.Successes))
	for i, pbSucc := range res.Successes {
		s.Successes[i] = &StmtResultSuccess{
			ColumnNames: make([]string, len(pbSucc.Columns)),
			Rows:        make([][]interface{}, len(pbSucc.Rows)),
		}
		for j, c := range pbSucc.Columns {
			s.Successes[i].ColumnNames[j] = c.Name
		}
		for j, r := range pbSucc.Rows {
			s.Successes[i].Rows[j] = make([]interface{}, len(r.Values))
			for k, v := range r.Values {
				s.Successes[i].Rows[j][k] = FromProtoValue(v)
			}
		}
	}
	if res.Error != nil {
		s.Error = &StmtResultError{Code: int(res.Error.Code), Message: res.Error.Message}
	}
}

// StmtResultsFromProto converts the given protobuf response to statement
// results.
func StmtResultsFromProto(res *sqlitepb.StmtResponse) []*StmtResult {
	results := make([]*StmtResult, len(res.Results))
	for i, result := range res.Results {
		results[i] = &StmtResult{}
		results[i].FromProto(result)
	}
	return results
}

// ToProtoValue converts a single Go value to a protobuf value. The value must
// be nil, bool, integer, float, string, byte slice, or a pointer to any of
// those.
func ToProtoValue(v interface{}) (*sqlitepb.Value, error) {
	return toProtoValue(reflect.ValueOf(v))
}

func toProtoValue(v reflect.Value) (*sqlitepb.Value, error) {
	switch v.Kind() {
	case reflect.Invalid:
		return &sqlitepb.Value{Value: &sqlitepb.Value_NullValue{NullValue: true}}, nil
	case reflect.Bool:
		if v.Bool() {
			return &sqlitepb.Value{Value: &sqlitepb.Value_IntValue{IntValue: 1}}, nil
		}
		return &sqlitepb.Value{Value: &sqlitepb.Value_IntValue{IntValue: 0}}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &sqlitepb.Value{Value: &sqlitepb.Value_IntValue{IntValue: v.Int()}}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := v.Uint()
		if val > math.MaxInt64 {
			return nil, fmt.Errorf("%v is over max int64", val)
		}
		return &sqlitepb.Value{Value: &sqlitepb.Value_IntValue{IntValue: int64(val)}}, nil
	case reflect.Float32, reflect.Float64:
		return &sqlitepb.Value{Value: &sqlitepb.Value_FloatValue{FloatValue: v.Float()}}, nil
	case reflect.Ptr:
		if v.IsNil() {
			return &sqlitepb.Value{Value: &sqlitepb.Value_NullValue{NullValue: true}}, nil
		}
		return toProtoValue(v.Elem())
	case reflect.String:
		return &sqlitepb.Value{Value: &sqlitepb.Value_StringValue{StringValue: v.String()}}, nil
	default:
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			return &sqlitepb.Value{Value: &sqlitepb.Value_BytesValue{BytesValue: v.Bytes()}}, nil
		}
		return nil, fmt.Errorf("type %v not supported", v.Type())
	}
}

// FromProtoValue converts the given protobuf value to a Go value. This will
// return nil, int64, float64, string, or bytes.
func FromProtoValue(v *sqlitepb.Value) interface{} {
	switch v := v.Value.(type) {
	case nil, *sqlitepb.Value_NullValue:
		return nil
	case *sqlitepb.Value_IntValue:
		return v.IntValue
	case *sqlitepb.Value_FloatValue:
		return v.FloatValue
	case *sqlitepb.Value_StringValue:
		return v.StringValue
	case *sqlitepb.Value_BytesValue:
		return v.BytesValue
	default:
		panic(fmt.Sprintf("unknown kind %T", v))
	}
}

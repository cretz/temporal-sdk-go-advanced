package temporalsqlite

import (
	"fmt"
	"math"
	"reflect"

	"github.com/cretz/temporal-sdk-go-advanced/temporalsqlite/sqlitepb"
)

type Stmt struct {
	Query string
	// Param indexes start at 1. Values can be null, bool, integer, float, string,
	// byte slice, or pointer to any of those.
	IndexedParams map[int]interface{}
	NamedParams   map[string]interface{}
}

func NewSimpleStmt(query string, args ...interface{}) *Stmt {
	s := &Stmt{Query: query, IndexedParams: make(map[int]interface{}, len(args))}
	for i, v := range args {
		s.IndexedParams[i+1] = v
	}
	return s
}

func (s *Stmt) ToProto() (*sqlitepb.Stmt, error) {
	stmt := &sqlitepb.Stmt{
		Sql:           s.Query,
		IndexedParams: make(map[uint32]*sqlitepb.Value, len(s.IndexedParams)),
		NamedParams:   make(map[string]*sqlitepb.Value, len(s.NamedParams)),
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

type StmtResult struct {
	ColumnNames []string
	// Values are null, int64, float64, string, or bytes
	Rows [][]interface{}
}

func (s *StmtResult) FromProto(res *sqlitepb.StmtResult) {
	s.ColumnNames = make([]string, len(res.Columns))
	for i, c := range res.Columns {
		s.ColumnNames[i] = c.Name
	}
	s.Rows = make([][]interface{}, len(res.Rows))
	for i, r := range res.Rows {
		s.Rows[i] = make([]interface{}, len(r.Values))
		for j, v := range r.Values {
			s.Rows[i][j] = FromProtoValue(v)
		}
	}
}

func ToProtoValue(v interface{}) (*sqlitepb.Value, error) {
	return toProtoValue(reflect.ValueOf(v))
}

func toProtoValue(v reflect.Value) (*sqlitepb.Value, error) {
	switch v.Kind() {
	case reflect.Invalid:
		return &sqlitepb.Value{Kind: &sqlitepb.Value_NullValue{NullValue: true}}, nil
	case reflect.Bool:
		if v.Bool() {
			return &sqlitepb.Value{Kind: &sqlitepb.Value_IntValue{IntValue: 1}}, nil
		}
		return &sqlitepb.Value{Kind: &sqlitepb.Value_IntValue{IntValue: 0}}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &sqlitepb.Value{Kind: &sqlitepb.Value_IntValue{IntValue: v.Int()}}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := v.Uint()
		if val > math.MaxInt64 {
			return nil, fmt.Errorf("%v is over max int64", val)
		}
		return &sqlitepb.Value{Kind: &sqlitepb.Value_IntValue{IntValue: int64(val)}}, nil
	case reflect.Float32, reflect.Float64:
		return &sqlitepb.Value{Kind: &sqlitepb.Value_FloatValue{FloatValue: v.Float()}}, nil
	case reflect.Ptr:
		if v.IsNil() {
			return &sqlitepb.Value{Kind: &sqlitepb.Value_NullValue{NullValue: true}}, nil
		}
		return toProtoValue(v.Elem())
	case reflect.String:
		return &sqlitepb.Value{Kind: &sqlitepb.Value_StringValue{StringValue: v.String()}}, nil
	default:
		if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
			return &sqlitepb.Value{Kind: &sqlitepb.Value_BytesValue{BytesValue: v.Bytes()}}, nil
		}
		return nil, fmt.Errorf("type %v not supported", v.Type())
	}
}

func FromProtoValue(v *sqlitepb.Value) interface{} {
	switch v := v.Kind.(type) {
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

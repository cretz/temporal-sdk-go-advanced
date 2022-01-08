package sandboxworker

import (
	"fmt"
	"go/token"
	"reflect"
	"runtime"
	"strings"
)

type funcRef struct {
	pkg  string
	name string
}

func newFuncRef(fn interface{}) (*funcRef, error) {
	rtFunc := runtime.FuncForPC(reflect.ValueOf(fn).Pointer())
	if rtFunc == nil {
		return nil, fmt.Errorf("unable to find function")
	}
	fullName := rtFunc.Name()

	// Cannot be method
	if strings.HasSuffix(fullName, "-fm") {
		return nil, fmt.Errorf("cannot use receiver method as function reference")
	}

	// Split into package and name. Since we already confirmed it's not a receiver
	// method, we can just split on the last dot. This may be an anon-func for
	// lambdas or function variables which will fail the exported check below.
	lastDot := strings.LastIndex(fullName, ".")
	pkg, name := fullName[:lastDot], fullName[lastDot+1:]

	// Must be exported
	if !token.IsExported(name) {
		return nil, fmt.Errorf("function must be top-level and exported")
	}

	return &funcRef{pkg: pkg, name: name}, nil
}

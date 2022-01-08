package sandboxworker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFuncRef(t *testing.T) {
	require := require.New(t)
	var s someStruct

	_, err := newFuncRef(s.FuncOnPtr)
	require.Error(err)
	require.Contains(err.Error(), "cannot use receiver method as function reference")

	_, err = newFuncRef(s.FuncOnStruct)
	require.Error(err)
	require.Contains(err.Error(), "cannot use receiver method as function reference")

	_, err = newFuncRef(funcUnexported)
	require.Error(err)
	require.Contains(err.Error(), "function must be top-level and exported")

	_, err = newFuncRef(FuncVar)
	require.Error(err)
	require.Contains(err.Error(), "function must be top-level and exported")

	_, err = newFuncRef(func() {})
	require.Error(err)
	require.Contains(err.Error(), "function must be top-level and exported")

	ref, err := newFuncRef(FuncNormal)
	require.NoError(err)
	require.Equal("github.com/cretz/temporal-sdk-go-advanced/temporaldeterminism/sandboxworker", ref.pkg)
	require.Equal("FuncNormal", ref.name)
}

type someStruct struct{}

func (*someStruct) FuncOnPtr()   {}
func (someStruct) FuncOnStruct() {}

func funcUnexported() {}

var FuncVar = func() {}

func FuncNormal() {}

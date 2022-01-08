package tracecompile

import "regexp"

var defaultFuncStrings = []string{
	"time.Now",
	"time.Sleep",
}

func DefaultFuncs() []FuncMatcher {
	panic("TODO")
}

// Names are qualified, can use ParseFuncName to get pieces
type FuncMatcher func(string) *bool

func FuncMatcherRegexp(r *regexp.Regexp, include bool) FuncMatcher {
	return func(name string) *bool {
		if r.MatchString(name) {
			return &include
		}
		return nil
	}
}

func FuncMatcherString(str string, include bool) FuncMatcher {
	return func(s string) *bool {
		if s == str {
			return &include
		}
		return nil
	}
}

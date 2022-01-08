package tracecompile

import (
	"github.com/cretz/temporal-sdk-go-advanced/temporaldeterminism/tracecompile/tracert"
)

func DefaultIncludeEvents() []tracert.EventType {
	return []tracert.EventType{
		tracert.EventNewGoroutine,
		tracert.EventFuncCall,
		tracert.EventMapIteration,
		tracert.EventChannelSend,
		tracert.EventChannelReceive,
		tracert.EventSelect,
	}
}

type OverlayOptions struct {
	// If empty, default is DefaultIncludeEvents which is all events.
	Events []tracert.EventType

	// If empty, default is DefaultRegexpFuncIn. Match attempted in order, with
	// first non-nil result
	Funcs []FuncMatcher
}

type Overlay struct {
	Files map[string]string
}

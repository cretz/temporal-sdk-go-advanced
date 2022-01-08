package tracert

import (
	"sync"
	"unsafe"

	"go.temporal.io/sdk/workflow"
)

type EventType int

const (
	EventUnknown EventType = iota
	EventFuncCall
	EventNewGoroutine
	EventMapIteration
	EventChannelSend
	EventChannelReceive
	EventSelect
)

type Event struct {
	Type    EventType
	Context workflow.Context

	// Fully qualified, only non-empty when Type is EventFuncCall.
	FuncName string
	// Number of frames to skip inside the event handler to get to where the event
	// occured (e.g. when using runtime.Caller).
	CallerSkip int
}

func ParseFuncName(funcName string) (pkg, receiver, name string) {
	panic("TODO")
}

func BuildFuncName(pkg, receiver, name string) {
	panic("TODO")
}

type eventListener func(*Event)

// Never altered, only replaced
var eventListeners []*eventListener
var eventListenersLock sync.RWMutex
var mutateEventListenersLock sync.Mutex

func AddEventListener(l func(*Event)) interface{ RemoveEventListener() } {
	// Init listeners
	initEventListeners()

	// Lock this entire call so we can create a new slice out of lock
	mutateEventListenersLock.Lock()
	defer mutateEventListenersLock.Unlock()

	// Get the slice under lock, create new with this, update slice under lock
	eventListenersLock.RLock()
	existingListeners := eventListeners
	eventListenersLock.RUnlock()
	newListeners := make([]*eventListener, len(existingListeners)+1)
	copy(newListeners, existingListeners)
	newListener := eventListener(l)
	newListeners[len(newListeners)-1] = &newListener
	eventListenersLock.Lock()
	eventListeners = newListeners
	eventListenersLock.Unlock()
	return eventListenerRemover{&newListener}
}

type eventListenerRemover struct{ l *eventListener }

func (e eventListenerRemover) RemoveEventListener() {
	// Lock this entire call so we can create a new slice out of lock
	mutateEventListenersLock.Lock()
	defer mutateEventListenersLock.Unlock()

	// Get the slice under lock, remove this listener, update slice under lock
	eventListenersLock.RLock()
	existingListeners := eventListeners
	eventListenersLock.RUnlock()
	// Get index and bail if not found
	eventListenerIndex := -1
	for i, existing := range existingListeners {
		if existing == e.l {
			eventListenerIndex = i
			break
		}
	}
	if eventListenerIndex == -1 {
		return
	}
	newListeners := make([]*eventListener, len(existingListeners)-1)
	copy(newListeners, existingListeners[:eventListenerIndex])
	copy(newListeners[eventListenerIndex:], existingListeners[eventListenerIndex+1:])
	eventListenersLock.Lock()
	eventListeners = newListeners
	eventListenersLock.Unlock()
}

// Set via go:linkname from the runtime package
func setEventListener(f func(eventType int, contextPointer unsafe.Pointer, funcName string, callerSkip int))

var initEventListenersOnce sync.Once

func initEventListeners() { initEventListenersOnce.Do(func() { setEventListener(onEvent) }) }

func onEvent(eventType int, contextPointer unsafe.Pointer, funcName string, callerSkip int) {
	// Grab listeners under lock, but can iterate not under lock because we know
	// the slice is never updated
	eventListenersLock.RLock()
	existingListeners := eventListeners
	eventListenersLock.RUnlock()
	if len(existingListeners) == 0 {
		return
	}

	// Call each event handler in order
	event := &Event{
		Type:       EventType(eventType),
		Context:    *(*workflow.Context)(contextPointer),
		FuncName:   funcName,
		CallerSkip: callerSkip,
	}
	for _, l := range existingListeners {
		(*l)(event)
	}
}

package temporalng1

type Context interface {
	Done() ReceiveChannel[struct{}]
	Err() error
	Value(any) any
}

type Channel[T any] struct {
	SendChannel[T]
	ReceiveChannel[T]
}

func NewChannel[T any](buffered int) *Channel[T] {
	panic("TODO")
}

type ReceiveChannel[T any] struct {
}

func (ReceiveChannel[T]) Receive() T {
	panic("TODO")
}

func (ReceiveChannel[T]) TryReceive() (T, bool) {
	panic("TODO")
}

type SendChannel[T any] struct {
}

func (SendChannel[T]) Send(T) {
	panic("TODO")
}

type Future[T any] struct {
}

func (Future[T]) Get() (T, error) {
	panic("TODO")
}

func NewFuture[T any]() (Future[T], FutureResolver[T]) {
	panic("TODO")
}

type FutureResolver[T any] struct {
}

func (FutureResolver[T]) Resolve(T) {
}

func (FutureResolver[T]) ResolveError(error) {
}

type SelectCase interface{ selectCase() }

func Select(Context, ...SelectCase) error {
	panic("TODO")
}

func SelectCaseReceive[T any](ReceiveChannel[T], func(Context, T) error) SelectCase {
	panic("TODO")
}

func SelectCaseTryReceive[T any](ReceiveChannel[T], func(Context, T, bool) error) SelectCase {
	panic("TODO")
}

func SelectCaseSend[T any](SendChannel[T], T, func() error) SelectCase {
	panic("TODO")
}

func SelectCaseFuture[T any](Future[T], func(T) error) SelectCase {
	panic("TODO")
}

func SelectCaseDefault(func() error) SelectCase {
	panic("TODO")
}

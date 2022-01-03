TODO(cretz): All efforts to curb non-determinism in Go workflows

**NOTE:** Under development/research

## Approaches

There are two approaches: Make non-deterministic Go work deterministically (see "Deterministic Execution" below) and
warn/prevent when using non-deterministic Go aspects (see "Non-determinism Checking" below).

### General Concepts

#### External Execution Mode

Since Go expects workflows and activities to be able to be defined side-by-side, any kind of deterministic execution and
some kinds of non-determinism checking require running the workflow in "external execution mode". This basically means
instead of being able to use state/memory of the same process, the workflow has to be run separately. This can be due to
needs to recompile, interpret, etc.

The approaches for external execution mode are not currently defined, but may incorporate any of the following:

* IPC to communicate between a workflow process and the primary process
* Inbound/outbound interceptor to proxy calls to/from the primary to the external

### Deterministic Execution

These approaches refer to using standard Go constructs (e.g. map iteration, goroutines, channels, sleep/time, etc).

All deterministic execution approaches would require "external execution mode".

One of the challenges here is that deterministic execution must run until all coroutines have yielded to the next
workflow event. This "on idle" concept is not always available for all scheduling mechanisms.

#### WASM

TODO(cretz): Demonstrate

Notes:

* Go-to-WASM is deterministic as all WASM is (but imports called by WASM may not be)
* Regular `go` compiler:
  * Makes a lot of assumptions about running in JS environment so need an adapter
    * E.g. https://github.com/mattn/gowasmer and https://github.com/go-wasm-adapter/go-wasm
  * Compilation makes a large WASM blob
  * While each individual build is deterministic, separate builds are not in a predictable way
    * Due to built in scheduling rules and yielding, one minor change from one build to the next may result in a
      non-obvious change in ordering expectations, but this may be ok with whole-workflow versioning
* TinyGo
  * Too many limitations: https://tinygo.org/docs/reference/lang-support/

#### Yaegi Interpreter

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### golang.org/x/tools/go/ssa/interp Interpreter

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Custom Interpreter

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Compile-time User Code Rewrite

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Compile-time Go Stdlib Code Rewrite

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Go Fork

TODO(cretz): Demonstrate

TODO(cretz): Notes

### Non-determinism Checking

These approaches refer to notifying or failing workflows using non-deterministic

One of the primary challenges here is that not all Go non-determinism is Temporal non-determinism. For example, just
because Go iterates maps in a non-deterministic fashion doesn't mean that such an iteration is non-deterministic with
regards to the ordering of commands sent back to Temporal (maybe it's just preparing params for an activity or
something).

#### Static Analysis

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Runtime Function Hooking + Goroutine Check

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### eBPF Function Tracing + Goroutine Check

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Compile-time Function Tracepoints + Goroutine Check

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Debugger Breakpoints + Goroutine Check

TODO(cretz): Demonstrate

TODO(cretz): Notes

#### Fuzz/Coverage Replay Tracing

TODO(cretz): Demonstrate

TODO(cretz): Notes
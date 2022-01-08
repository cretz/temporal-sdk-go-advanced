
(under research)

Attempt code rewrites to remove non-determinism. Approach:

* Probably use `-overlay` to replace code during compile time
* Add deterministic pieces to `runtime` package for map iteration, channel creation, etc
  * Use `//go:linkname` to avoid direct dependency on `sdk-go/workflow` package?
* Change out the `context` package
  * Use `//go:linkname` to avoid direct dependency on `sdk-go/workflow` package?
* Disable some calls/packages (e.g. `time.Now`, `crypto/rand`, etc)
  * Cannot remove since we have to let all code compile, just disable at runtime

Currently sucks because:

* Local activities and logging and such have to use IPC to call back to primary process and I didn't want to write all
  of that (yet)
* Not sure we can rewrite packages depended on by `sdk-go/workflow` and `sdk-go/worker` since we expect those to
  continue to work
* Have to essentially have global context so we can not require such a param be woven throughout


TODO(cretz): Make a runtime determinism checker. Idea is to possibly combine the following:

* Get access to runtime.getg (via func symbol walk), then access to goid for coroutine-local storage
* Intercept inbound ExecuteWorkflow and outbound workflow.Go to mark goroutine as a workflow goroutine
* Patch calls to, e.g. `time.Now`, random, etc to check that we're not in a workflow goroutine
  * https://github.com/undefinedlabs/go-mpatch is better than https://github.com/bouk/monkey due to license
  * Or can I patch the function table?
* See what runtime functions can also be patched for similar checks (e.g. `runtime.newProc`, `runtime.chansend`,
  `runtime.mapiter`)
* Provide way to mark a section of code unchecked (e.g.
  `temporalwfcheck.BeginIgnore(temporalwfcheck.NewGoroutineCheck))`)
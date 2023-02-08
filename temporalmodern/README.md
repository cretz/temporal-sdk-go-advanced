## Temporal Go API Modernization

Ideas for newer workflow API. Rules for each approach:

* Must have generic types for workflows, activities, signals, and queries
* Must make invoking and returning all of the above respect their types
* Must make workflow and activity registering obey types
* Can force single-arg and/or single-return functions as needed
* Do not have to implement yet, just have to demonstrate compile-time safety
* Examples must show a workflow that can be queried, waits for a signal, then executes an activity and returns

See the approaches below. The links are to the examples which each list their approaches and pros/cons within.

* [temporalng1](temporalng1/example/main.go)
* [temporalng2](temporalng2/example/main.go)
* [temporalng3](temporalng3/example/main.go)
* [temporalng4](temporalng4/example/main.go)

Notes:

* There are many even crazier ideas that were not included due to impracticality
* Activity and workflow definitions could define their own default options
  * Just sucks for the untyped users
* Could offer code gen, but that sucks for users
* Some of the ideas here can be mixed and matched
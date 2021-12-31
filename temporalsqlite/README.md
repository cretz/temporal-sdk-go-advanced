# SQLite as a Workflow [![Go Reference](https://pkg.go.dev/badge/github.com/cretz/temporal-sdk-go-advanced/temporalsqlite.svg)](https://pkg.go.dev/github.com/cretz/temporal-sdk-go-advanced/temporalsqlite)

This project implements a workflow with state backed by SQLite. Features:

* SQLite database inside a workflow which means it can survive worker crashes
* Supports common request/response executions to the DB
* Supports read-only queries even after the DB has been stopped
* Built-in continue-as-new support to prevent unchecked history growth
* Can export full serialized version of the DB even after the DB has been stopped
* Some functions disabled to ensure workflow determinism
* Full-featured CLI
* Easy-to-use library

While this project is just a proof-of-concept and is missing some possible features (see TODO), it was written to be a
robust database solution. While a Temporal workflow-based DB may not be as highly performance-optimized as a general
server-based DB, it has many novel resiliency/reliability features. Even when not used via its general purpose DB
interface, it provides a good example for using SQLite as workflow state.

The implementation leverages the [Protobuf generator](../temporalproto) to provide a strong contract.

## Concepts

### Overview

Temporal workflows are long-running, deterministic, event-sourced functions which can maintain deterministic replayable
state. Such state is often implemented via language-native memory structures, but there's no reason it can't instead use
an embedded database. That's what this project does. It embeds a SQLite database in the workflow with support for
multiple forms of statement execution.

Any number of workers can have the SQLite workflow registered. A single workflow execution represents a single database.
When a workflow is first started, an in-memory SQLite database is created, potentially restoring from a previously
serialized instance. Temporal workflow signals and queries are used to serve different types of requests described
below.

These DB workflows have all the benefits of Temporal workflows. For example, while they benefit from worker stickiness
for faster execution of frequent statements, they are also optimized for infrequent use. Instead of a traditional
database which often use CPU/mem resources even when unused, a Temporal workflow can remain unused and serialized until
needed. It is not unrealistic to have hundreds of small DB workflows "open" at a time, even if they are not accessed for
days/weeks.

### Statements

A statement is either a single query with parameters (named or indexed) or multiple queries. There are multiple forms of
statement invocation approaches described below. Each can accept multiple statements at a time.

#### Executions

Executions are statements that can mutate the database and may return a result. This is the most powerful form of
statement. Executions are implemented via a request/response mechanism whereby the client sends a signal with the
statement request and receives a completed statement response via an activity execution on a worker the client is
listening on.

Since executions rely on signals, they can only be executed while the database workflow is running.

#### Queries

Queries are statements that must be read-only and may return a result. Queries are implemented via Temporal workflow
queries. The benefit of using a query instead of an execution for a read-only statement is that it does not increase
history size.

Since queries rely on Temporal queries, they can be executed even when the database has stopped (i.e. the workflow has
been cancelled). This is a neat concept since, so long as the completed DB workflow exists on the server (subject to
workflow retention) it can be queried. A DB workflow can be created w/ a few simple mutations via executions or updates
and then immediately cancelled, and queries will continue to work.

#### Updates

Updates are statements that can mutate the database, but have no response and don't even respond back whether they
succeeded. Updates are implemented via Temporal workflow signals. Since there is no error mechanism, by default a failed
update is set to fail the entire DB workflow (but this can be configured to just log a warning). This therefore makes
updates much less robust approach than executions and queries. If a mutable statement is needed, the only benefit of
using an update over an execution is that it is cheaper on history and worker/workflow performance to not have to
respond with a result.

Since updates rely on signals, they can only be executed while the database workflow is running.

### Serialization and Continue-as-new

The DB workflow can accept a previously serialized instance of a DB (implemented as an LZ4-compressed result of
[SQLite serialization](https://www.sqlite.org/c3ref/serialize.html)). While a caller can pass in a serialized database
instance, its primary purpose is to support continue-as-new.

When a workflow history (i.e. its events) grows too large, it can become heavy to replay or otherwise manage and the
server enforces a maximum history size. The
[recommended approach](https://docs.temporal.io/docs/go/workflows/#large-event-histories) to handle this is to use
continue-as-new which atomically recreates the workflow. This implementation, after thousands executions and updates
have occurred (but not affected by queries), will serialize the database and issue a continue-as-new to restart/recreate
the database with a clean set of history. While this is an implementation detail that should not be visible to callers
in most cases, it serves as a good example of how a workflow might leverage continue-as-new to restart itself.

As an added feature, a query can be issued to request the serialized database at any point in time. Since this is a
query, it can even be executed on a stopped DB workflow. The CLI for example (described later), takes this serialized
database and saves it as a local file that can be directly used by any other SQLite tool.

### Non-determinism

Temporal workflows must be deterministic to be safe for replay. While SQLite has fewer non-determinisms than traditional
RDBMSs (e.g. an absent `ORDER BY` is still deterministic), there are still some. SQLite's
[definition of determinism](https://www.sqlite.org/deterministic.html) is a strict one that expects function purity
within the context of a statement. Temporal's definition of determinism is just that the same calls in the same order
must do the same thing and produce the same result within the context of the entire workflow.

To reduce non-determinism, this implementation disables use of `current_date()`, `random()`, and other
non-deterministic functions. However, it does not restrict all functions considered non-deterministic by SQLite such as
`last_insert_rowid()` since that is deterministic by Temporal standard.

The SQLite date/time functions like `datetime()` are only non-deterministic for some arguments (namely `'now'`). Since
this cannot be prevented via simple function definition without removing the entire date/time functionality, the current
implementation just leaves it there. See TODO for alternative approaches.

### Deadlock Detection and Serialization Size

Temporal workflow code is expected to execute extremely fast and the parameters are not meant for extremely large sizes.
Using SQLite inline with the workflow code and serializing its entire state or results can potentially cause problems
here.

The Temporal Go SDK contains a deadlock detector that ensures that no workflow code runs longer than a second. This
means SQLite calls cannot take very long or they can trigger this. In practice this has only been observed to be a
concern for large databases. This project is not recommended for large, frequently updated databases.

Temporal serializes statement request/response through the server, and in cases of executions/updates, stores them in
history too. Therefore items like entire SQL responses to executions are stored in history. This project is not
recommended for large requests/responses.

As mentioned in the "Serialization" section, this project serializes the entire database as an LZ4-compressed blob for
continue-as-new. Since that means it is used as the workflow parameter and since the serialized state is used as a
parameter for the workflow, it is also stored in history. In practice this means that while dozens of inserts may only
result in a 2K blob, many hundreds/thousands can easily exceed 20K or more. This project is not recommended for large
databases.

## Building

This project may be used as a library or a CLI. To build the CLI, with the latest `Go` on the `PATH`, from this
directory run:

    go build ./cmd/temporalsqlite

Here is the usage defa

## Usage and CLI Walkthrough

Commands here use the build `temporalsqlite` CLI binary, but that can be replaced with `go run ./cmd/temporalsqlite` to
skip the build step. Here is output of `--help`:

```
$ temporalsqlite --help
NAME:
  temporalsqlite - Temporal SQLite utilities

USAGE:
  temporalsqlite [global options] command [command options] [arguments...]

COMMANDS:
  worker         run a SQLite worker
  get-or-create  get or create the DB
  stop           stop the DB
  save           download the DB into a local SQLite file
  query          send a read-only query and get a response
  update         send an update that, by default, fails the entire DB if it does not succeed
  exec           exec a query and get a response
  help, h        Shows a list of commands or help for one command

GLOBAL OPTIONS:
  --help, -h  show help (default: false)
```

Before starting, a Temporal server must be running. This walkthrough assumes a server is running locally. See the
[quick install guide](https://docs.temporal.io/docs/server/quick-install/). The `--server` and/or `--namespace` and/or
`--task-queue` flags can be used to customize the client/worker.

At least one worker must register the workflow for the server. In the background or a separate terminal, start the
worker:

    $ temporalsqlite worker

That will remain running and serve all SQLite databases. While `get-or-create` can create the DB, it is automatically
created when running any of the statement calls. To create a table:

    $ temporalsqlite exec --db mydb1 "CREATE TABLE mytable1 (v1 PRIMARY KEY, v2)"
    Executions succeeded

This has started the database and created the table. To confirm it is running:

    $ temporalsqlite get-or-create --db mydb1
    DB mydb1 status: Running

Now we can insert records. Let's try via query:

    $ temporalsqlite query --db mydb1 "INSERT INTO mytable1 VALUES ('foo1', 'bar1')"
    statement expected to be read only but got action SQLITE_INSERT
    exit status 1

This shows an error because queries may not be used for mutations. So we use exec:

    $ temporalsqlite exec --db mydb1 "INSERT INTO mytable1 VALUES ('foo1', 'bar1')"
    Executions succeeded

We can also use parameterized executions. The parameters must be in the form of:

    --param <index|name>=<type>(<value>)

When using indexes, they start with 1. To insert another record using parameters:

    $ temporalsqlite exec --db mydb1 --param "1=string(foo2)" --param ":myparam=float(1.23)" "INSERT INTO mytable1 VALUES (?, :myparam)"
    Executions succeeded

We can query for all records:

    $ temporalsqlite query --db mydb1 "SELECT * FROM mytable1"
    +------+------+
    |  V1  |  V2  |
    +------+------+
    | foo1 | bar1 |
    | foo2 | 1.23 |
    +------+------+
    Queries succeeded

This could have been done with the exact same results using `exec` but we chose query since it's read only. We can try
multiple queries:

    $ temporalsqlite query --db mydb1 "SELECT * FROM mytable1; SELECT COUNT(1) FROM mytable1"
    expected single statement since multi-query not set, but got multiple
    exit status 1

This is because, to execute multiple statements, we have to use `--multi` (and they can't have parameters):

    $ temporalsqlite query --db mydb1 --multi "SELECT * FROM mytable1; SELECT COUNT(1) FROM mytable1"
    +------+------+
    |  V1  |  V2  |
    +------+------+
    | foo1 | bar1 |
    | foo2 | 1.23 |
    +------+------+

    +----------+
    | COUNT(1) |
    +----------+
    |        2 |
    +----------+
    Queries succeeded

What if we try an execution that doesn't work?

    $ temporalsqlite exec --db mydb1 "INSERT INTO notexists VALUES (1)"
    sqlite.Conn.Prepare: SQLITE_ERROR: no such table: notexists (INSERT INTO notexists VALUES (1)) (code 1)
    exit status 1

This is fine and the workflow is still running, but if we did this with `update` instead of `exec` the workflow would
silently fail (not shown).

Say we wanted to stop the DB, we can run:

    $ temporalsqlite stop --db mydb1
    DB stopped

Confirm it is stopped:

    $ temporalsqlite get-or-create --db mydb1
    DB mydb1 status: Canceled

So if we try to exec a statement:

    $ temporalsqlite exec --db mydb1 "SELECT * FROM mytable1"
    workflow execution already completed
    exit status 1

It fails. But queries can be executed even on completed workflows. For example:

    $ temporalsqlite query --db mydb1 "SELECT * FROM mytable1"
    +------+------+
    |  V1  |  V2  |
    +------+------+
    | foo1 | bar1 |
    | foo2 | 1.23 |
    +------+------+
    Queries succeeded

This works so long as the workflow is still retained on the server (there is a retention period after which they are
deleted).

At any time (including after complete as we are now), a serialized instance of the DB may be obtained and stored
locally using `save`:

    $ temporalsqlite save --db mydb1
    DB saved at mydb1.db

That file can be used via any SQLite tool. For example, using the standard `sqlite3` CLI:

    $ sqlite3 mydb1.db "SELECT * FROM mytable1"
    foo1|bar1
    foo2|1.23

Now that the DB is stopped, we cannot restart it. But we can create a new one with its name using `--create-if-stopped`:

    $ temporalsqlite get-or-create --db mydb1 --create-if-stopped
    DB mydb1 status: Running

Of course, the table does not exist because it is a blank DB:

    $ temporalsqlite query --db mydb1 "SELECT * FROM mytable1"
    sqlite.Conn.Prepare: SQLITE_ERROR: no such table: mytable1 (SELECT * FROM mytable1) (code 1)
    exit status 1

There are more advanced flags we did not show here.

## Development

### Generate Code from Protos

To regenerate the protos, from this dir with `protoc` and `protoc-gen-go` on the `PATH`, install
`protoc-gen-go_temporal` from this repo:

    go install ../temporalproto/cmd/protoc-gen-go_temporal

And run:

    protoc --go_out=paths=source_relative:. --go_temporal_out=paths=source_relative:. -I . -I ../temporalproto ./sqlitepb/sqlite.proto


### Integration Tests

Integration tests are present in [test](test). To run from the `test` directory:

    go test -ldflags "-extldflags=-Wl,--allow-multiple-definition" .

The `--allow-multiple-definition` flag is required because the embedded Temporalite server references a different SQLite
client, and having two SQLite clients each attempting to statically compile SQLite causes a compilation error.

## TODO

* Prevent `datetime('now')` and friends by replacing the entire impl with our own `strftime` that prevents `'now'` as an
  argument
* Metrics
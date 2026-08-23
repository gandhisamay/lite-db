# lite-db

`lite-db` is an educational key-value database written in Go. It is a small
storage-engine experiment focused on the foundations of durable writes:

- an in-memory key-value map for reads
- a checksummed write-ahead log (WAL) for recovery
- ordered, asynchronous WAL writes
- group-commit `fsync` batching

The project is intentionally a work in progress. It is currently a WAL-backed
in-memory store, not a production database and not yet a general-purpose
database server.

## How it works

The current architecture is:

```text
CLI
 |
 v
database.Database ──> in-memory map
        |
        v
  store.WriteRequest
        |
        v
      Store ──> memtable
        |
        v
   WAL writer ──> data.wal
```

On startup, `database.Start` opens `data.wal`, replays its records, validates
each checksum, and reconstructs the in-memory map. Mutations are represented as
structured `store.WriteRequest` values and sent through a single writer
pipeline. The WAL serializes each request only at the persistence boundary and
flushes batches of five records or when the short batching timer expires.

Each WAL record is encoded as:

```text
4-byte payload length (little endian)
payload bytes
4-byte CRC32 checksum (little endian)
```

The payload is a plain-text command such as `SET greeting hello` or
`DELETE greeting`.

## Current operations

The database recognizes these case-sensitive operations:

| Operation | Behavior | Arguments |
| --- | --- | --- |
| `GET` | Reads a value from the in-memory map | `GET key` |
| `SET` | Writes or overwrites a value | `SET key value` |
| `DELETE` | Removes a key | `DELETE key` |

Keys and values are currently split using whitespace, so they must be supplied
as single whitespace-free command-line arguments. `SET` inserts a new key or
updates the value when the key already exists.

## Getting started

### Requirements

- Go `1.26.5` or a compatible Go toolchain, as specified in `go.mod`

### Run the tests

```bash
go test ./...
```

The store test also verifies that five queued records are written and
`fsync`ed as one group commit:

```bash
go test ./store -run TestBatchOfFiveIsFsyncedOnce -v
```

### Build and run the CLI

Build the executable:

```bash
go build -o lite-db .
```

The command-line shape is:

```text
lite-db COMMAND KEY [VALUE]
```

Examples:

```bash
./lite-db GET greeting
./lite-db SET greeting hello
./lite-db SET language golang
./lite-db DELETE greeting
```

The WAL path is relative to the process working directory, so these commands
create or use `data.wal` in the directory from which the executable is run.
`data.wal` is ignored by Git because it is runtime data.

## Important limitations

This repository is a learning project, and several behaviors are deliberately
unfinished:

- The CLI starts a new database for every command and does not currently call
  `Database.Close` before exiting.
- `Store.Append` queues a write and returns before the WAL writer has appended
  and `fsync`ed it. As a result, a process that exits immediately after a
  mutation may lose that mutation.
- Store shutdown does not yet wait for the writer goroutines to finish, so the
  close path also needs hardening before durability can be promised.
- There is no server, interactive shell, transaction layer, locking, or
  concurrent-client protocol.
- All live data is held in memory; the WAL is the only on-disk representation.
- There are no pages, buffer pool, index, compaction, or size-management
  mechanisms yet.
- The CLI assumes the required arguments are present and currently reports
  failures with simple console messages.
- WAL records use whitespace-delimited text, so arbitrary strings and values
  containing spaces are not supported.

These limitations are useful boundaries for the next stages of the project:
first finish the WAL lifecycle and ordering guarantees, then add page storage,
a buffer pool, and an index such as a B+ tree. Transactions and MVCC can come
later.

## Repository layout

```text
.
├── main.go              # Command-line entry point
├── database/db.go       # Database API and in-memory state machine
├── store/request.go     # Structured write requests and operation types
├── store/store.go       # Batched asynchronous write pipeline
├── store/store_test.go  # Group-commit test with a fake WAL
├── memtable/memtable.go # In-memory table for staged writes
├── wal/wal.go           # WAL framing, checksums, replay, and fsync
├── go.mod               # Go module definition
└── plan.md              # Development roadmap and storage-engine notes
```

## Development notes

The package boundaries mirror the storage path:

1. `database` applies logical operations to the map and creates structured write requests.
2. `store` batches requests, writes them to the WAL, and updates the memtable.
3. `wal` serializes requests and owns the file format and recovery checks.

When changing the WAL format, update both `Append` and `Replay` together and
add coverage for truncated records and checksum failures. When changing the
writer pipeline, preserve the ordering guarantee and test shutdown as well as
batch-size and timer-triggered flushes.

## Project direction

The intended learning path is:

```text
WAL and recovery
      -> fixed-size pages
      -> page manager
      -> buffer pool
      -> B+ tree index
      -> transactions and MVCC
```

The current implementation is at the first stage: WAL and recovery
infrastructure around an in-memory map.

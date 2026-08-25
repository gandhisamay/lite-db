# lite-db

`lite-db` is an educational key-value storage engine written in Go. It
currently combines an in-memory memtable with a checksummed write-ahead log
(WAL), and exposes a small command-line interface.

The project is a work in progress. It is not yet a production database or a
database server.

## Current architecture

```text
CLI
 |
 v
database.Database
 |
 v
store.Store ──> memtable.Memtable
 |
 v
asynchronous WAL writer ──> data.wal
```

On startup, the store opens `data.wal`, replays and verifies its records, and
rebuilds the active memtable. Mutations are sent through a write channel,
appended to the WAL, `fsync`ed, and then applied to the memtable.

The memtable stores values in a map and retains delete tombstones so that an
older value cannot be resurrected during future flushing or merging. It also
provides `SortedEntries`, which returns records ordered by key for later SSTable
writing.

The `sst` package contains the initial SSTable record encoder, but
`WriteToSST` is not implemented yet.

## Supported operations

The CLI recognizes these case-sensitive commands:

| Command | Usage | Description |
| --- | --- | --- |
| `GET` | `GET key` | Reads a value |
| `SET` | `SET key value` | Inserts or replaces a value |
| `DELETE` | `DELETE key` | Records a tombstone |

Arguments are currently split by whitespace, so keys and values cannot contain
spaces.

## Getting started

### Requirements

- Go 1.26.5, or a compatible Go toolchain

### Run the tests

```bash
go test ./...
```

The store test verifies that five queued requests can be written and fsynced
as one group when the writer is configured with a batch size of five.

### Build and run

```bash
go build -o lite-db .
./lite-db SET greeting hello
./lite-db GET greeting
./lite-db DELETE greeting
```

The WAL is stored as `data.wal` in the process working directory. Runtime WAL
data is ignored by Git.

## WAL format

Each WAL record is encoded as:

```text
4-byte payload length   (little endian)
payload bytes
4-byte CRC32 checksum   (little endian)
```

The payload is a whitespace-delimited text record:

```text
1 key value\n    # SET
2 key value\n    # DELETE; value is currently empty
```

The WAL supports append, `fsync`, replay, and checksum validation. A truncated
record or checksum mismatch causes replay to return an error.

## Repository layout

```text
.
├── main.go              # CLI entry point
├── database/db.go       # User-facing database operations
├── store/request.go     # Write request and operation definitions
├── store/store.go       # Write pipeline, replay, and memtable management
├── store/store_test.go  # Group-commit test
├── memtable/memtable.go # In-memory values, tombstones, and sorted records
├── wal/wal.go           # WAL framing, checksums, replay, and fsync
├── sst/sst.go           # Initial SSTable encoding scaffold
├── go.mod               # Go module definition
└── plan.md              # Storage-engine roadmap
```

## Known limitations

- All live data is currently held in memory; the WAL is the only completed
  on-disk storage path.
- The default store batch size is currently one, although the writer supports
  configurable batch sizes and timer-triggered flushes.
- `Append` queues a mutation and returns before the asynchronous writer has
  completed its WAL append and `fsync`.
- Shutdown does not yet wait for all writer goroutines to finish, and callers
  must explicitly close the database.
- The CLI starts a new database instance for each command and does not
  currently close it before exiting.
- There is no completed SSTable writer, page manager, buffer pool, index,
  compaction, transaction layer, MVCC, locking, or concurrent-client protocol.
- The CLI assumes the required arguments are present and reports errors with
  simple console messages.
- The text WAL format does not support arbitrary keys or values containing
  whitespace.

## Roadmap

The intended learning path is:

```text
WAL and recovery
    -> SSTable flushing
    -> fixed-size pages and page manager
    -> buffer pool
    -> B+ tree or LSM index
    -> compaction
    -> transactions and MVCC
```

The current focus is completing the transition from the WAL-backed memtable to
durable sorted-table files.

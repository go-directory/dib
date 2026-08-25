![dib_logo_small](https://github.comc/user-attachments/assets/85052a0c-e1d8-4357-84d1-296e6939e7f0)

Package dib wraps [boltdb/bolt](https://github.com/boltdb/bolt) to provide a storage backend for directory data. It is designed for use by [go-directory/dsa](https://github.com/go-directory/dsa).

This package does not implement any wire protocols such as LDAP, DAP, DISP, et al.,

## Status

The current state of this package is EXPERIMENTAL. It should not be used in mission-critical or production environments, and is prone to breaking changes at any time. There are some interesting long-term plans for this package, so it should continue to evolve and grow over time.

Contributions are most welcome.

## License

The dib package is released under the terms of the MIT license. See the repository root for applicable license files.

## Database

The package supports Ephemeral and Persistent database forms.

When operating in Persistent mode, the underlying database is an instance of bolt.DB is used. The importing application(s) do not directly interact with [bolt] contexts, as these are all layered in convenient abstraction.

When operating in Ephemeral mode, a map-based "mimicry" of a bolt.DB is used and is entirely contained in memory. While data is written to an Ephemeral database instance in the same manner as its Persistent counterpart, all content within an Ephemeral database instance is LOST when the process exits UNLESS it was otherwise preserved in some manner.

This package offers two distinct database "types":

 - Persistent (traditional bolt database)
 - Ephemeral (map-based mimicry of bolt database)

[boltdb/bolt](https://github.com/boltdb/bolt)-types are purely interface oriented:

 - `Database`
 - `Bucket`
 - `Cursor`
 - `Tx`

Actual instances are private and are of (functionally) like-design. This is done to make the interactions between ephemeral and persistent databases procedurally identical to the caller.

Some methods -- such as `Database.Batch` -- are only meaningful when used upon a persistent database. When used upon an ephemeral database, `Database.Update` is called instead.

Conversely, `Bucket.Sort` is only meaningful for ephemeral use cases, and will have no effect when used upon persistent instances.

### Persistent Databases

These are file-based, and are intended for real data that should survive restarts.

Aside from a few convenience methods this package affords the user, these databases are nothing more than a boltdb wrapped inside a private struct.

### Ephemeral Databases

These are intended for session-specific data, i.e.:

  - Runtime monitoring statistics
  - Runtime config abstractions
  - Published schemata
  - The Root DSE
  - Test DITs
  - As performance optimizer
    - EQUALITY, SUBSTR, PRESENCE indices for fast reads
    - Special object caching (i.e.: values -- which may or may not require initial processing -- that are accessed by the caller in high volume)

As these reside solely in memory, they are NOT appropriate for large data sets unless sufficient system memory is available. Use wisely.

## Logging

Logging is disabled by default. To enable logging:

  - `go run` or `go build` with `-tags dib_debug`, and ...
  - Enact the desired loglevel(s) via `DirectoryInformationBase.Loglevel.Shift(level)`

<sub>NOTE: Loglevel is an instance of [`*common.BitValue`](https://github.com/go-directory/common)</sub>

Loglevels available:

  - `0` (none, default)
  - `1` (errors)
  - `2` (warnings)
  - `4` (trace)

Where DIB is used in busy production environments, logging should be disabled unless actually needed. If it absolutely must be enabled, try to limit your loglevels to errors only (1).

Callers/importers may use the logger for their own purposes. When built or run with `-tags dib_debug`, see the `EventLogger` global.

Example:

```go
  // In your package:
  dib.EventLogger.Printf("my message %q", someVar)
```

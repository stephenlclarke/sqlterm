# SQLTerm upstream PR #1 follow-up

## Summary

Fix five issues found while reviewing upstream jpxcz/sqlterm#1:

- Preserve escaped SQL strings, quoted identifiers and Unicode names.
- Return query results when SQL starts with a comment.
- Ignore a second Run request while a query is running.
- Keep Unicode text intact when clipping and aligning the display.
- Use the configured database name for DSN connections.

## Type Of Change

- [x] Bug fix
- [x] Documentation update

## Motivation And Context

These issues could change SQL values, hide results, repeat a database write or
select the wrong database. This follow-up updates the branch used by
jpxcz/sqlterm#1 after review.

## Testing

- [x] Tested locally
- [x] Added regression tests
- [x] Updated documentation

The new tests failed before the fixes and pass with them. The full Go test suite
passes with 92.0% line coverage. Build, vet and staticcheck pass. No live
database was used.

## Compatibility

No configuration keys or command-line options change. A DSN now uses the
existing `database` setting. A second query can start after the first finishes.

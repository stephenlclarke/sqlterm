# SQLTerm upstream PR #1: query safety review

## Problem

Review of [jpxcz/sqlterm#1](https://github.com/jpxcz/sqlterm/pull/1) found five
related problems in the ODBC query workspace. The formatter changed escaped
string contents and Unicode identifiers. Leading SQL comments sent row queries
through the execution path that discards rows. Repeated Ctrl+R submitted
duplicate writes while a query was running. Byte-based display clipping damaged
Unicode text. DSN connections ignored the configured database name.

## Reproduction

The added regression tests fail on the published source. They cover escaped
quotes, bracket escapes, combining characters, hash comments, SELECT with
leading comments, repeated submission, wide-character display, and DSN database
selection. Tests use a mock database and do not connect to a live server.

## Expected result

Keep SQL data unchanged, return query rows, submit one statement at a time,
preserve displayed characters, and connect to the selected database.

## Scope and ownership

This follow-up belongs to the upstream PR refresh on 24 November 2026. Keep the
review branch and worktree until the fixes merge into Stephen's protected main
branch. That merge also updates upstream PR #1. The original local checkout and
its untracked files are unchanged.

## CI follow-up

The commit-message job installed the conventional rules but did not load them.
It rejected a valid commit with an empty-rules error. The workflow now loads
those rules explicitly. The same commit passes, and actionlint passes.

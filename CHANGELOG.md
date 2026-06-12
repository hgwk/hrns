# Changelog

## v0.1.2 - 2026-06-12

### Changed

- Add side-effect-free top-level and command help output.
- Make `hrns init --help` print usage instead of creating files.
- Fail `verify-line-count` when configured roots scan zero files, so bad
  project roots cannot produce a misleading PASS.

## v0.1.1 - 2026-06-11

### Changed

- Harden Go packaging and config handling.

## v0.1.0 - 2026-06-09

Initial public release of `hrns`.

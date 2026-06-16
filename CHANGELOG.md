# Changelog

## v0.1.4 - 2026-06-16

### Changed

- Add package-level `test` and `version` scripts so npm and CLI checks use the
  same release version.

## v0.1.3 - 2026-06-12

### Changed

- Add npm postinstall binary downloader for GitHub Release assets.
- Refine audit catalog config detection so inactive audits do not look runnable.
- Keep npm package contents scoped to the CLI runtime files.

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

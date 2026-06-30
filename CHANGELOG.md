# Changelog

## v0.1.9 - 2026-06-30

### Fixed

- Preserve npm binary entries during publish by using package-relative bin
  paths.
- Keep release smoke independent of the npm `ldgr` package metadata cache by
  installing `ldgr` from source for the `audit:ldgr` check.

## v0.1.8 - 2026-06-16

### Changed

- Add `hrns init --home <path>` for isolated automation, matching the shared
  guide-pointer init flow.
- Split init-related CLI tests so repository line-count guardrails remain under
  300 lines per file.

## v0.1.7 - 2026-06-16

### Fixed

- Resolve Markdown links relative to the document that contains them.
- Ignore Markdown links inside fenced code blocks during docs symbol sync.

## v0.1.6 - 2026-06-16

### Changed

- Let `hrns init --target <path>` create the target directory when it does not
  exist, matching other init automation flows.

## v0.1.5 - 2026-06-16

### Changed

- Make `hrns init` install `AGENTS.md` / `CLAUDE.md` pointers and the
  home-local audit guide by default.
- Add `hrns init --no-instructions` for repos that intentionally skip agent
  guide pointers.

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

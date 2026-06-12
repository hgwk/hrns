# hrns

Portable repository audit and test harness CLI.

This package started as the reusable audit harness extracted from Agent-Zero.
The primary implementation is a Go CLI that runs from a consumer repository
root and executes packaged checks against that repository.

## Usage

From another project:

```sh
go install github.com/hgwk/hrns/cmd/hrns@latest
hrns version
hrns init --docs --instructions
hrns list
hrns audit
```

For local development and manual installs, use this shared convention:

```sh
install -m 0755 hrns ~/.local/bin/hrns
```

If another PATH directory must expose `hrns`, prefer a symlink back to
`~/.local/bin/hrns` instead of copying multiple binaries.

Local development in this repository:

```sh
go run ./cmd/hrns version
go run ./cmd/hrns list
go run ./cmd/hrns audit
go run ./cmd/hrns audit --all
```

## Companion Tool Roles

- `cduo doctor` checks pair-agent runtime setup and project hook readiness.
- `ldgr verify` checks ledger lifecycle, audit, worklog, and Git evidence.
- `hrns audit` checks repository structure, docs, config, and code guardrails.

## Configuration

Create `hrns.config.json` with:

```sh
hrns init
```

`hrns init` detects the current repository shape and writes roots for the
directories and files that actually exist, such as `src`, `tests`,
`migrations`, `docs`, `package.json`, and TypeScript config files.

Create a JSON document proposal template with:

```sh
hrns init --docs
```

Inject a short `AGENTS.md` / `CLAUDE.md` prelude and create the home-local
instruction body with:

```sh
hrns init --instructions
```

The CLI also reads `package.json#hrns`. `hrns.config.json` wins over
`package.json#hrns`.

Useful keys:

```json
{
  "auditSets": {
    "default": ["verify-line-count"],
    "all": ["verify-line-count", "verify-docs-duplication"]
  },
  "lineAudit": {
    "maxLines": 300,
    "roots": ["cmd", "internal", "scripts", "packages", "infra"],
    "extensions": [".go", ".ts", ".tsx", ".mjs", ".js", ".rs", ".sql"]
  },
  "env": {
    "example": ".env.example",
    "roots": ["packages", "scripts"],
    "requiredPrefixes": ["APP_", "OPENAI_"],
    "ignored": ["PATH", "NODE_ENV", "CI"]
  },
  "docsDuplication": {
    "mode": "fail",
    "roots": ["docs", "README.md"],
    "threshold": 0.72
  },
  "docsProposal": {
    "mode": "fail",
    "roots": ["docs", "README.md"],
    "proposalPath": ".hrns/doc-proposal.json",
    "threshold": 0.52,
    "titleThreshold": 0.45
  },
  "forbiddenReferences": {
    "mode": "warn",
    "roots": ["docs", "apps", "packages"],
    "rules": [{ "pattern": "legacy-api", "message": "use the current API surface" }],
    "allowPaths": ["^docs/archive/"]
  },
  "structureRatchet": {
    "mode": "fail",
    "files": [
      {
        "path": "apps/api/src/routes/users.ts",
        "maxLines": 120,
        "metrics": [{ "name": "raw fetch", "pattern": "\\bfetch\\s*\\(", "max": 0 }]
      }
    ]
  }
}
```

`mode` may be `fail`, `warn`, or `off`. hrns does not own task state, lessons,
tickets, or worklogs; use ldgr for that layer.

Instruction injection matches cduo/ldgr prelude behavior. The actual rules live
in `~/.hrns/audit-guide.md`, and root policy files get only a top-of-file
absolute `@.../.hrns/audit-guide.md` pointer. `AGENTS.md` and `CLAUDE.md` are
both ensured by default; existing body content is preserved below the pointer.
Set `HRNS_HOME` to override the home-local directory in tests or isolated
environments.

This is the shared guide-pointer convention used by `cduo`, `ldgr`, and `hrns`:
the home-local guide holds the long body, while root policy files only carry the
absolute `@...` pointer and any project-local rules below it.

## Audit Sets

Stable audits are intended to work across normal TypeScript/JavaScript
repositories with little or no configuration:

- `verify-line-count`
- `verify-docs-symbol-sync`
- `verify-sensitive-config-placeholders`
- `verify-no-orphan-fixtures`
- `verify-duplicate-helpers`
- `verify-thin-barrel-modules`

Configurable audits are copied in, but still need project-shape options before
they should be treated as universally portable:

- `verify-env-example-symbol-sync`
- `verify-agent-instruction-drift`
- `verify-docs-duplication`
- `verify-doc-proposal`
- `verify-json-duplicate-keys`
- `verify-forbidden-references`
- `verify-magic-numbers`
- `verify-structure-ratchet`
- `verify-no-placeholder-routes`
- `verify-scope-drift`
- `verify-speculative-abstractions`
- `verify-regression-evidence`
- `verify-main-diff-scope`
- `verify-stop-rule`
- `verify-elegance-review`

Ledger validation is delegated to `ldgr verify`; hrns does not duplicate ledger
state-model rules. Project-local runtime preflights, such as Agent-Zero
watchdog checks, should live in that project's own runbook rather than in hrns.
Upstream drift checks are project-specific and should be configured outside the
portable default audit set.

`verify-docs-duplication` is the guard for agents that keep creating
near-duplicate Markdown files. Set it to `fail` once a project has a settled
documentation taxonomy.

The structural guardrails are config-driven:

- `verify-json-duplicate-keys` finds duplicate keys in JSON files before the
  parser silently keeps the last value.
- `verify-forbidden-references` blocks configured legacy names, docs, imports,
  or surfaces outside allowlisted paths.
- `verify-magic-numbers` warns on inline numeric policy values that should move
  to named constants.
- `verify-structure-ratchet` enforces per-file line and regex-count budgets.
- `verify-no-placeholder-routes` catches stable route files that still return
  placeholder, coming-soon, or not-implemented responses.
- `verify-scope-drift` compares changed files with active ldgr claim paths.
- `verify-speculative-abstractions` flags new single-use Manager/Factory/etc.
  surfaces that are likely premature.
- `verify-regression-evidence` warns when a bugfix-looking diff has no changed
  regression test.

For prevention before a new Markdown file exists, use the JSON proposal gate:

```sh
hrns docs:index
hrns docs:check .hrns/doc-proposal.json
```

`docs:check` compares the proposal's `title`, `purpose`, `summary`, and
`content` against existing Markdown. If it overlaps, the proposal must switch to
`"decision": "update_existing"` and set `target` to the existing document path.
That turns "make another similar doc" into "patch the source of truth".

Pattern-only harness files are preserved under their original source paths:

- `packages/e2e/helpers/servers.ts`
- `packages/desktop/test/desktop-wire-smoke.ts`
- `packages/desktop/test/desktop-wire-smoke-matrix.ts`

## Packaging Direction

Keep runnable generic checks in the Go CLI under `cmd/hrns` and `internal/hrns`.
Checks should read the target repository from the current working directory.

Primary distribution is the Go module plus GitHub Release tarballs named
`hrns_<version>_<os>_<arch>.tar.gz`. The npm package keeps a thin compatibility
wrapper that shells to `go run`, so npm consumers need Go installed.

Project-specific defaults belong in `hrns.config.json` or `package.json#hrns`
instead of being hard-coded in each verifier.

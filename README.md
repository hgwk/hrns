# hrns

Portable repository audit and test harness scripts.

This package started as the reusable audit harness extracted from Agent-Zero.
The first packaging target is a Node CLI that runs from a consumer repository
root and executes the packaged checks against that repository.

## Usage

From another project:

```sh
pnpm add -D @hgwk/hrns
pnpm hrns init --tasks --docs --instructions
pnpm hrns list
pnpm hrns audit
```

Local development in this repository:

```sh
pnpm list
pnpm audit
pnpm audit:all
```

## Configuration

Create `hrns.config.json` with:

```sh
pnpm hrns init
```

Create JSON task files with:

```sh
pnpm hrns init --tasks
```

Create a JSON document proposal template with:

```sh
pnpm hrns init --docs
```

Inject a short `AGENTS.md` / `CLAUDE.md` include and create
`.hrns/instructions.md` with:

```sh
pnpm hrns init --instructions
```

The CLI also reads `package.json#hrns`. `hrns.config.json` wins over
`package.json#hrns`.

Useful keys:

```json
{
  "auditSets": {
    "default": ["verify-line-count.mjs"],
    "all": ["verify-line-count.mjs", "verify-operational-surface.mjs"]
  },
  "lineAudit": {
    "maxLines": 300,
    "roots": ["packages", "scripts"],
    "extensions": [".ts", ".tsx", ".mjs", ".js"]
  },
  "env": {
    "example": ".env.example",
    "roots": ["packages", "scripts"],
    "requiredPrefixes": ["APP_", "OPENAI_"],
    "ignored": ["PATH", "NODE_ENV", "CI"]
  },
  "operational": {
    "requiredRootScripts": ["ci", "audit"],
    "docsRoots": ["docs", "README.md"],
    "packageJsonRoots": ["package.json", "packages"]
  },
  "workflow": {
    "mode": "warn",
    "todoPath": "tasks/todo.json",
    "lessonsPath": "tasks/lessons.json"
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
  }
}
```

`mode` may be `fail`, `warn`, or `off`. Workflow state is JSON by default;
`tasks/todo.json` and `tasks/lessons.json` are intended to be machine-readable
agent coordination files, not prose scratchpads.

Instruction injection is intentionally short. `AGENTS.md` and `CLAUDE.md` get
only an `HRNS_START` include block; the actual rules live in
`.hrns/instructions.md`, similar to cduo/ldgr-style local instruction includes.

## Audit Sets

Stable audits are intended to work across normal TypeScript/JavaScript
repositories with little or no configuration:

- `verify-line-count.mjs`
- `verify-docs-symbol-sync.mjs`
- `verify-sensitive-config-placeholders.mjs`
- `verify-no-orphan-fixtures.mjs`
- `verify-duplicate-helpers.mjs`
- `verify-thin-barrel-modules.mjs`

Configurable audits are copied in, but still need project-shape options before
they should be treated as universally portable:

- `verify-env-example-symbol-sync.mjs`
- `verify-closed-world.mjs`
- `verify-operational-surface.mjs`
- `verify-agent-instruction-drift.mjs`
- `verify-docs-duplication.mjs`
- `verify-task-workflow.mjs`
- `verify-proof-record.mjs`
- `verify-root-cause-record.mjs`
- `verify-main-diff-scope.mjs`
- `verify-stop-rule.mjs`
- `verify-elegance-review.mjs`
- `ledger-verify.mjs`
- `upstream-drift-check.mjs`
- `dev-watchdog-preflight.mjs`

`verify-docs-duplication.mjs` is the guard for agents that keep creating
near-duplicate Markdown files. Set it to `fail` once a project has a settled
documentation taxonomy.

For prevention before a new Markdown file exists, use the JSON proposal gate:

```sh
pnpm hrns docs:index
pnpm hrns docs:check .hrns/doc-proposal.json
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

Keep runnable generic checks under `scripts/` and expose them through `bin/hrns.mjs`.
Checks should read the target repository from `process.cwd()`, while package
assets should be resolved from `import.meta.url`.

Project-specific defaults belong in `hrns.config.json` or `package.json#hrns`
instead of being hard-coded in each verifier.

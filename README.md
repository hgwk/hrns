# hrns

Portable repository audit and test harness scripts.

This package started as the reusable audit harness extracted from Agent-Zero.
The first packaging target is a Node CLI that runs from a consumer repository
root and executes the packaged checks against that repository.

## Usage

From another project:

```sh
pnpm add -D @hgwk/hrns
pnpm hrns init
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
  }
}
```

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
- `ledger-verify.mjs`
- `upstream-drift-check.mjs`
- `dev-watchdog-preflight.mjs`

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

#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { asStringArray, loadConfig } from '../scripts/audit-lib/config.mjs'

const PACKAGE_ROOT = dirname(dirname(fileURLToPath(import.meta.url)))
const config = loadConfig()

const stableAudits = [
  'verify-line-count.mjs',
  'verify-docs-symbol-sync.mjs',
  'verify-sensitive-config-placeholders.mjs',
  'verify-no-orphan-fixtures.mjs',
  'verify-duplicate-helpers.mjs',
  'verify-thin-barrel-modules.mjs',
]

const configurableAudits = [
  'verify-env-example-symbol-sync.mjs',
  'verify-closed-world.mjs',
  'verify-operational-surface.mjs',
  'verify-agent-instruction-drift.mjs',
  'verify-docs-duplication.mjs',
  'verify-doc-proposal.mjs',
  'verify-main-diff-scope.mjs',
  'verify-stop-rule.mjs',
  'verify-elegance-review.mjs',
]

const specializedAudits = [
  'ledger-verify.mjs',
  'upstream-drift-check.mjs',
  'dev-watchdog-preflight.mjs',
]

const command = process.argv[2] ?? 'audit'
const args = process.argv.slice(3)

if (command === 'list') {
  printList()
  process.exit(0)
}

if (command === 'init') {
  initConfig()
  if (args.includes('--docs')) initDocsProposal()
  if (args.includes('--instructions')) initInstructions()
  process.exit(0)
}

if (command === 'audit') {
  const includeAll = args.includes('--all')
  const auditSets = config.auditSets ?? {}
  const configuredDefault = asStringArray(auditSets.default, stableAudits)
  const configuredAll = asStringArray(auditSets.all, [...configuredDefault, ...configurableAudits])
  const audits = includeAll ? configuredAll : configuredDefault
  process.exit(runMany(audits))
}

if (command === 'run') {
  const name = args[0]
  if (!name) die('usage: hrns run <script-name>')
  process.exit(runOne(normalizeScriptName(name)))
}

if (command === 'line-audit') {
  process.exit(runOne('line-audit.mjs'))
}

if (command === 'docs:index') {
  process.exit(runOne('docs-index.mjs'))
}

if (command === 'docs:check') {
  const proposal = args[0]
  const env = proposal ? { ...process.env, HRNS_DOC_PROPOSAL: proposal } : process.env
  process.exit(runOne('verify-doc-proposal.mjs', env))
}

die(`unknown command: ${command}`)

function runMany(scripts) {
  let failures = 0
  for (const script of scripts) {
    const status = runOne(script)
    if (status !== 0) failures += 1
  }
  if (failures > 0) {
    console.error(`hrns audit: FAIL (${failures}/${scripts.length} audit(s) failed)`)
    return 1
  }
  console.log(`hrns audit: PASS (${scripts.length} audit(s) passed)`)
  return 0
}

function runOne(script, env = process.env) {
  const scriptPath = join(PACKAGE_ROOT, 'scripts', script)
  if (!existsSync(scriptPath)) die(`script not found: ${script}`)
  const result = spawnSync(process.execPath, [scriptPath], {
    cwd: process.cwd(),
    stdio: 'inherit',
    env,
  })
  return result.status ?? 1
}

function normalizeScriptName(name) {
  if (name.endsWith('.mjs')) return name
  return `${name}.mjs`
}

function printList() {
  console.log('Stable audits:')
  for (const script of stableAudits) console.log(`- ${script}`)
  console.log('\nConfigurable audits:')
  for (const script of configurableAudits) console.log(`- ${script}`)
  console.log('\nSpecialized scripts:')
  for (const script of specializedAudits) console.log(`- ${script}`)
  console.log('\nConfigured default audit set:')
  for (const script of asStringArray(config.auditSets?.default, stableAudits)) console.log(`- ${script}`)
}

function initConfig() {
  const target = join(process.cwd(), 'hrns.config.json')
  if (existsSync(target)) {
    console.log('hrns.config.json already exists')
    return
  }
  writeFileSync(target, `${JSON.stringify(defaultProjectConfig(), null, 2)}\n`)
  console.log('created hrns.config.json')
}

function initDocsProposal() {
  const hrnsDir = join(process.cwd(), '.hrns')
  mkdirSync(hrnsDir, { recursive: true })
  const proposalPath = join(hrnsDir, 'doc-proposal.json')
  if (!existsSync(proposalPath)) {
    writeFileSync(proposalPath, `${JSON.stringify(defaultDocProposal(), null, 2)}\n`)
    console.log('created .hrns/doc-proposal.json')
  }
}

function initInstructions() {
  const instructionsPath = instructionBodyPath()
  mkdirSync(dirname(instructionsPath), { recursive: true })
  writeFileSync(instructionsPath, defaultInstructions())
  console.log(`updated ${instructionsPath}`)
  for (const file of ['AGENTS.md', 'CLAUDE.md']) injectInstructionInclude(file)
}

function injectInstructionInclude(file) {
  const target = join(process.cwd(), file)
  const pointer = `@${instructionBodyPath()}`
  if (!existsSync(target)) {
    writeFileSync(target, `${pointer}\n`)
    console.log(`created ${file}`)
    return
  }
  const current = readFileSync(target, 'utf8')
  const updated = upsertInstructionPointer(current, pointer)
  if (updated === current) {
    console.log(`${file} already references hrns instructions`)
    return
  }
  writeFileSync(target, updated)
  console.log(`updated ${file}`)
}

function upsertInstructionPointer(current, pointer) {
  const cleaned = stripLeadingSeparator(removeKnownInstructionPointers(current)).trim()
  if (current.trim() === pointer) return current
  if (cleaned === '') return `${pointer}\n`
  return `${pointer}\n\n---\n\n${cleaned}\n`
}

function removeKnownInstructionPointers(content) {
  let out = content
  for (const pointer of [`@${instructionBodyPath()}`, '@.hrns/instructions.md']) {
    out = removePointerPrelude(out, pointer)
    out = stripLeadingSeparator(out)
  }
  return out
}

function removePointerPrelude(content, pointer) {
  const lines = content.split('\n')
  if (lines[0]?.trim() !== pointer) return content
  return lines.slice(1).join('\n')
}

function stripLeadingSeparator(content) {
  return content.replace(/^\s*---\s*\n+/, '')
}

function instructionBodyPath() {
  const home = process.env.HRNS_HOME ?? join(process.env.HOME ?? process.cwd(), '.hrns')
  return join(home, 'instructions.md')
}

function defaultProjectConfig() {
  return {
    scanRoots: ['packages', 'scripts', 'infra'],
    auditSets: {
      default: stableAudits,
      all: [...stableAudits, ...configurableAudits],
    },
    lineAudit: {
      maxLines: 300,
      roots: ['packages', 'scripts', 'infra'],
      extensions: ['.ts', '.tsx', '.mjs', '.js', '.rs', '.sql'],
    },
    docs: {
      roots: ['docs', 'README.md', 'AGENTS.md', 'CLAUDE.md'],
    },
    fixtures: {
      roots: ['packages'],
      testRoots: ['packages'],
    },
    env: {
      example: '.env.example',
      roots: ['packages', 'scripts', 'infra'],
      requiredPrefixes: [],
      ignored: ['PATH', 'NODE_ENV', 'CI', 'LINE_AUDIT_MAX'],
    },
    operational: {
      requiredRootScripts: ['ci', 'audit', 'line-audit'],
      docsRoots: ['docs', 'README.md', '.github'],
      packageJsonRoots: ['package.json', 'packages'],
    },
    closedWorld: {
      roots: ['packages', 'scripts'],
      extraEntrypoints: [],
      alwaysAllowedPatterns: [],
    },
    docsDuplication: {
      mode: 'warn',
      roots: ['docs', 'README.md'],
      threshold: 0.72,
      minTokens: 80,
    },
    docsProposal: {
      mode: 'fail',
      roots: ['docs', 'README.md'],
      proposalPath: '.hrns/doc-proposal.json',
      indexPath: '.hrns/docs-index.json',
      threshold: 0.52,
      titleThreshold: 0.45,
      minProposalTokens: 12,
    },
    mainDiff: {
      mode: 'warn',
      base: 'main',
      maxFiles: 40,
      maxChangedLines: 1200,
    },
    stopRule: {
      mode: 'warn',
      logPaths: ['.hrns/failures.log'],
      repeatedFailureThreshold: 2,
    },
    elegance: {
      mode: 'warn',
      base: 'main',
      maxNewFiles: 20,
      maxLargeFiles: 4,
    },
  }
}

function defaultDocProposal() {
  return {
    version: 1,
    proposals: [
      {
        path: 'docs/example.md',
        title: 'Example Document',
        purpose: 'Explain why this must be a new document instead of an update.',
        summary: 'Short summary of the planned content.',
        decision: 'new_document',
        target: '',
      },
    ],
  }
}

function defaultInstructions() {
  return `# hrns Instructions

## Audit Gates

- Run \`pnpm hrns audit\` before marking work complete.
- For broader review, run \`pnpm hrns audit --all\` and resolve fail-mode findings.
- Keep project-specific gate behavior in \`hrns.config.json\` or \`package.json#hrns\`.

## Task And Worklog Ownership

- hrns does not own tasks, lessons, tickets, or worklogs.
- Use ldgr for task state, lessons, append-only tickets, worklogs, and handoff records.
- Use hrns only for repository audit gates and document creation checks.

## Document Creation Gate

- Before creating a new Markdown document, write \`.hrns/doc-proposal.json\`.
- Run \`pnpm hrns docs:index\` and \`pnpm hrns docs:check .hrns/doc-proposal.json\`.
- If the proposal overlaps an existing document, do not create the new file. Update the reported existing document instead.
- To intentionally update an existing document, set \`"decision": "update_existing"\` and \`"target"\` to the existing document path.

## Duplicate Instruction Control

- Keep long operating instructions in one included file.
- Do not paste the same guidance into both \`AGENTS.md\`, \`CLAUDE.md\`, and tool-specific instruction files.
- If an include already points to this file, update this file rather than adding another prose copy.
`
}

function die(message) {
  console.error(`hrns: ${message}`)
  process.exit(1)
}

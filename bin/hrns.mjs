#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
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
  'verify-task-workflow.mjs',
  'verify-proof-record.mjs',
  'verify-root-cause-record.mjs',
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
  if (!args.includes('--tasks-only')) initConfig()
  if (args.includes('--tasks')) initTasks()
  if (args.includes('--docs')) initDocsProposal()
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

function initTasks() {
  const tasksDir = join(process.cwd(), 'tasks')
  mkdirSync(tasksDir, { recursive: true })
  const todoPath = join(tasksDir, 'todo.json')
  const lessonsPath = join(tasksDir, 'lessons.json')
  if (!existsSync(todoPath)) {
    writeFileSync(todoPath, `${JSON.stringify(defaultTodo(), null, 2)}\n`)
    console.log('created tasks/todo.json')
  }
  if (!existsSync(lessonsPath)) {
    writeFileSync(lessonsPath, `${JSON.stringify(defaultLessons(), null, 2)}\n`)
    console.log('created tasks/lessons.json')
  }
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
    workflow: {
      mode: 'warn',
      todoPath: 'tasks/todo.json',
      lessonsPath: 'tasks/lessons.json',
    },
    proof: {
      mode: 'warn',
      roots: ['tasks/todo.json', 'ledger/worklog.jsonl', 'README.md'],
      terms: ['verification', '검증', 'evidence', 'commands'],
    },
    rootCause: {
      mode: 'warn',
      sources: ['tasks/todo.json', 'ledger/worklog.jsonl'],
      requiredTerms: ['root cause', 'impact', 'why missed', 'verification'],
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
      logPaths: ['tasks/todo.json', 'tasks/failures.log'],
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

function defaultTodo() {
  return {
    version: 1,
    items: [
      {
        id: 'plan',
        task: 'Write the implementation plan before changing code.',
        status: 'todo',
      },
      {
        id: 'verify',
        task: 'Record verification commands and observed results.',
        status: 'todo',
      },
    ],
    verification: {
      commands: [],
      evidence: [],
      notes: '',
    },
    review: {
      summary: '',
      risks: [],
    },
  }
}

function defaultLessons() {
  return {
    version: 1,
    lessons: [],
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

function die(message) {
  console.error(`hrns: ${message}`)
  process.exit(1)
}

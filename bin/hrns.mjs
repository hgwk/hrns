#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { existsSync, writeFileSync } from 'node:fs'
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

function runOne(script) {
  const scriptPath = join(PACKAGE_ROOT, 'scripts', script)
  if (!existsSync(scriptPath)) die(`script not found: ${script}`)
  const result = spawnSync(process.execPath, [scriptPath], {
    cwd: process.cwd(),
    stdio: 'inherit',
    env: process.env,
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
  if (existsSync(target)) die('hrns.config.json already exists')
  writeFileSync(target, `${JSON.stringify(defaultProjectConfig(), null, 2)}\n`)
  console.log('created hrns.config.json')
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
  }
}

function die(message) {
  console.error(`hrns: ${message}`)
  process.exit(1)
}

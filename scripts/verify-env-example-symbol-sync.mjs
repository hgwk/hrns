#!/usr/bin/env node

import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit, unique } from './audit-lib/report.mjs'

const audit = createAudit('verify-env-example-symbol-sync')

const ENV_REF = /process\.env(?:\[['"]([A-Z0-9_]+)['"]\]|\.([A-Z0-9_]+))/g
const EXAMPLE_KEY = /^#?\s*([A-Z][A-Z0-9_]+)=/gm
const REQUIRED_PREFIXES = ['AGENTZERO_', 'AGZ_', 'OFFICENEXT_', 'OPENAI_', 'DATABASE_', 'CONTROL_']
const IGNORED = new Set([
  'PATH',
  'NODE_ENV',
  'CI',
  'TEST_DATABASE_URL',
  'UPDATE_FIXTURES',
  'LINE_AUDIT_MAX',
  'FORENSIC_INCLUDE_TS',
  'ALLOW_OFFLINE_UPSTREAM',
  'DRIFT_NOW',
])

const example = readText('.env.example')
const exampleKeys = new Set([...example.matchAll(EXAMPLE_KEY)].map((match) => match[1]))
const refs = []

for (const file of listFiles({ roots: ['packages', 'scripts', 'infra'] })) {
  const text = readText(file)
  for (const match of text.matchAll(ENV_REF)) refs.push(match[1] ?? match[2])
}

for (const key of unique(refs)) {
  if (IGNORED.has(key)) continue
  if (!REQUIRED_PREFIXES.some((prefix) => key.startsWith(prefix))) continue
  if (!exampleKeys.has(key)) audit.fail(`${key} is read from code but missing in .env.example`)
}

audit.finish()

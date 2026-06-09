#!/usr/bin/env node

import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit, unique } from './audit-lib/report.mjs'
import { asStringArray, configSection } from './audit-lib/config.mjs'

const audit = createAudit('verify-env-example-symbol-sync')
const config = configSection('env')

const ENV_REF = /process\.env(?:\[['"]([A-Z0-9_]+)['"]\]|\.([A-Z0-9_]+))/g
const EXAMPLE_KEY = /^#?\s*([A-Z][A-Z0-9_]+)=/gm
const REQUIRED_PREFIXES = asStringArray(config.requiredPrefixes, [])
const IGNORED = new Set(asStringArray(config.ignored, []))

if (REQUIRED_PREFIXES.length === 0) {
  console.log('verify-env-example-symbol-sync: PASS (no required prefixes configured)')
  process.exit(0)
}

const example = readText(config.example ?? '.env.example')
const exampleKeys = new Set([...example.matchAll(EXAMPLE_KEY)].map((match) => match[1]))
const refs = []

for (const file of listFiles({ roots: asStringArray(config.roots, ['packages', 'scripts', 'infra']) })) {
  const text = readText(file)
  for (const match of text.matchAll(ENV_REF)) refs.push(match[1] ?? match[2])
}

for (const key of unique(refs)) {
  if (IGNORED.has(key)) continue
  if (!REQUIRED_PREFIXES.some((prefix) => key.startsWith(prefix))) continue
  if (!exampleKeys.has(key)) audit.fail(`${key} is read from code but missing in ${config.example}`)
}

audit.finish()

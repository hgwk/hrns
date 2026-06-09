#!/usr/bin/env node

import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'

const audit = createAudit('verify-duplicate-helpers')

const HELPER_NAMES = [
  'asNumber',
  'clamp',
  'coerceNumber',
  'parseBoolean',
  'parseEnv',
  'sleep',
  'toNumber',
]
const DECL_RE = /\b(?:export\s+)?(?:function|const)\s+([A-Za-z0-9_]+)\b/g
const locations = new Map()

for (const file of listFiles({ roots: ['packages', 'scripts'] })) {
  if (!/\.(ts|tsx|js|mjs)$/.test(file)) continue
  if (file.endsWith('.test.ts') || file.endsWith('.test.tsx')) continue
  const text = readText(file)
  for (const match of text.matchAll(DECL_RE)) {
    const name = match[1]
    if (!HELPER_NAMES.includes(name)) continue
    const entries = locations.get(name) ?? []
    entries.push(file)
    locations.set(name, entries)
  }
}

for (const [name, files] of locations.entries()) {
  const uniqueFiles = [...new Set(files)]
  if (uniqueFiles.length > 1) audit.fail(`duplicate helper "${name}"`, uniqueFiles.join(', '))
}

audit.finish()

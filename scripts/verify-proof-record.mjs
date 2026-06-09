#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-proof-record')
const config = configSection('proof')
const mode = config.mode ?? 'warn'
const roots = asStringArray(config.roots, ['tasks/todo.json', 'ledger/worklog.jsonl', 'README.md'])
const terms = asStringArray(config.terms, ['verification', '검증', 'evidence', 'commands'])
const findings = []

let haystack = ''
for (const root of roots) {
  if (exists(root)) haystack += `\n${readText(root)}`
}

if (!haystack.trim()) {
  findings.push({ message: 'no proof record sources found', detail: roots.join(', ') })
} else if (!terms.some((term) => haystack.toLowerCase().includes(term.toLowerCase()))) {
  findings.push({
    message: 'no verification/proof marker found',
    detail: `looked for: ${terms.join(', ')}`,
  })
}

finishByMode(audit, findings, mode)

#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-root-cause-record')
const config = configSection('rootCause')
const mode = config.mode ?? 'warn'
const sources = asStringArray(config.sources, ['tasks/todo.json', 'ledger/worklog.jsonl'])
const required = asStringArray(config.requiredTerms, [
  'root cause',
  'impact',
  'why missed',
  'verification',
])
const findings = []

let haystack = ''
for (const source of sources) {
  if (exists(source)) haystack += `\n${readText(source)}`
}

if (!haystack.trim()) {
  findings.push({ message: 'no root-cause record source found', detail: sources.join(', ') })
} else {
  for (const term of required) {
    if (!haystack.toLowerCase().includes(term.toLowerCase())) {
      findings.push({ message: 'root-cause record missing term', detail: term })
    }
  }
}

finishByMode(audit, findings, mode)

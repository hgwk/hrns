#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-stop-rule')
const config = configSection('stopRule')
const mode = config.mode ?? 'warn'
const logPaths = asStringArray(config.logPaths, ['tasks/todo.json', 'tasks/failures.log'])
const repeatedFailureThreshold = Number(config.repeatedFailureThreshold ?? 2)
const findings = []

let text = ''
for (const path of logPaths) {
  if (exists(path)) text += `\n${readText(path)}`
}

const failures = [...text.matchAll(/(?:FAIL|ERROR|failed|error):?\s+(.{8,120})/gi)].map((match) =>
  normalize(match[1]),
)
const counts = new Map()
for (const failure of failures) counts.set(failure, (counts.get(failure) ?? 0) + 1)

for (const [failure, count] of counts) {
  if (count >= repeatedFailureThreshold) {
    findings.push({
      message: 'repeated failure pattern should trigger replanning',
      detail: `${count}x ${failure}`,
    })
  }
}

finishByMode(audit, findings, mode)

function normalize(value) {
  return value.toLowerCase().replace(/\s+/g, ' ').trim()
}

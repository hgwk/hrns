#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { asStringArray, configSection } from './audit-lib/config.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-main-diff-scope')
const config = configSection('mainDiff')
const mode = config.mode ?? 'warn'
const base = config.base ?? 'main'
const maxFiles = Number(config.maxFiles ?? 40)
const maxLines = Number(config.maxChangedLines ?? 1200)
const riskyPatterns = asStringArray(config.riskyPatterns, [
  'package-lock.json',
  'pnpm-lock.yaml',
  '^dist/',
  '^build/',
  '^coverage/',
])
const findings = []

const mergeBase = git(['merge-base', 'HEAD', base]).trim()
if (!mergeBase) {
  finishByMode(audit, [{ message: `cannot find merge base with ${base}` }], mode)
  process.exit(mode === 'fail' ? 1 : 0)
}

const names = git(['diff', '--name-only', mergeBase, 'HEAD']).trim().split('\n').filter(Boolean)
const stat = git(['diff', '--numstat', mergeBase, 'HEAD']).trim().split('\n').filter(Boolean)
const changedLines = stat.reduce((sum, line) => {
  const [add, del] = line.split('\t')
  return sum + numeric(add) + numeric(del)
}, 0)

if (names.length > maxFiles) findings.push({ message: 'diff touches too many files', detail: `${names.length} > ${maxFiles}` })
if (changedLines > maxLines) findings.push({ message: 'diff changes too many lines', detail: `${changedLines} > ${maxLines}` })

for (const file of names) {
  if (riskyPatterns.some((pattern) => new RegExp(pattern).test(file))) {
    findings.push({ message: 'diff touches risky/generated path', detail: file })
  }
}

finishByMode(audit, findings, mode)

function git(args) {
  const result = spawnSync('git', args, { encoding: 'utf8' })
  return result.status === 0 ? result.stdout : ''
}

function numeric(value) {
  return /^\d+$/.test(value) ? Number(value) : 0
}

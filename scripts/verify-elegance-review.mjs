#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { asStringArray, configSection } from './audit-lib/config.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-elegance-review')
const config = configSection('elegance')
const mode = config.mode ?? 'warn'
const base = config.base ?? 'main'
const maxNewFiles = Number(config.maxNewFiles ?? 20)
const maxLargeFiles = Number(config.maxLargeFiles ?? 4)
const largeFileLineThreshold = Number(config.largeFileLineThreshold ?? 250)
const smellPatterns = asStringArray(config.smellPatterns, [
  'TODO',
  'FIXME',
  'temporary',
  'workaround',
  'hack',
])
const findings = []

const mergeBase = git(['merge-base', 'HEAD', base]).trim()
if (!mergeBase) {
  finishByMode(audit, [{ message: `cannot find merge base with ${base}` }], mode)
  process.exit(mode === 'fail' ? 1 : 0)
}

const status = git(['diff', '--name-status', mergeBase, 'HEAD']).trim().split('\n').filter(Boolean)
const newFiles = status.filter((line) => line.startsWith('A\t')).map((line) => line.split('\t')[1])
if (newFiles.length > maxNewFiles) {
  findings.push({ message: 'large number of new files; consider a smaller change boundary', detail: `${newFiles.length} > ${maxNewFiles}` })
}

const numstat = git(['diff', '--numstat', mergeBase, 'HEAD']).trim().split('\n').filter(Boolean)
const largeFiles = numstat.filter((line) => {
  const [add, del] = line.split('\t')
  return numeric(add) + numeric(del) >= largeFileLineThreshold
})
if (largeFiles.length > maxLargeFiles) {
  findings.push({ message: 'many large changed files; consider splitting/refactoring', detail: `${largeFiles.length} > ${maxLargeFiles}` })
}

const patch = git(['diff', mergeBase, 'HEAD'])
for (const pattern of smellPatterns) {
  const regex = new RegExp(`^\\+.*${escapeRegex(pattern)}`, 'gim')
  const count = [...patch.matchAll(regex)].length
  if (count > 0) findings.push({ message: 'new patch contains unresolved smell marker', detail: `${pattern}: ${count}` })
}

finishByMode(audit, findings, mode)

function git(args) {
  const result = spawnSync('git', args, { encoding: 'utf8', maxBuffer: 20 * 1024 * 1024 })
  return result.status === 0 ? result.stdout : ''
}

function numeric(value) {
  return /^\d+$/.test(value) ? Number(value) : 0
}

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-agent-instruction-drift')
const config = configSection('agentInstructions')
const mode = config.mode ?? 'warn'
const files = asStringArray(config.files, ['AGENTS.md', 'CLAUDE.md', '.cduo/orchestration.md', '.ldgr/instructions.md'])
const minWords = Number(config.minRepeatedWords ?? 24)
const findings = []

const docs = files.filter(exists).map((file) => ({ file, text: readText(file) }))

for (let i = 0; i < docs.length; i += 1) {
  for (let j = i + 1; j < docs.length; j += 1) {
    const overlap = longestWordRun(words(docs[i].text), words(docs[j].text))
    if (overlap >= minWords) {
      findings.push({
        message: 'agent instruction files contain duplicated prose',
        detail: `${docs[i].file} <-> ${docs[j].file}: ${overlap} word run`,
      })
    }
  }
}

finishByMode(audit, findings, mode)

function words(text) {
  return text.toLowerCase().match(/[a-z0-9가-힣_:-]+/g) ?? []
}

function longestWordRun(left, right) {
  let best = 0
  const prev = new Array(right.length + 1).fill(0)
  const curr = new Array(right.length + 1).fill(0)
  for (let i = 1; i <= left.length; i += 1) {
    for (let j = 1; j <= right.length; j += 1) {
      curr[j] = left[i - 1] === right[j - 1] ? prev[j - 1] + 1 : 0
      if (curr[j] > best) best = curr[j]
    }
    prev.splice(0, prev.length, ...curr)
    curr.fill(0)
  }
  return best
}

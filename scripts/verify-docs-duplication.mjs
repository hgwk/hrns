#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'

const audit = createAudit('verify-docs-duplication')
const config = configSection('docsDuplication')
const mode = config.mode ?? 'warn'
const roots = asStringArray(config.roots, ['docs', 'README.md'])
const threshold = Number(config.threshold ?? 0.72)
const minTokens = Number(config.minTokens ?? 80)
const findings = []

const docs = listFiles({ roots })
  .filter((file) => file.endsWith('.md'))
  .map((file) => ({ file, tokens: tokenSet(readText(file)) }))
  .filter((doc) => doc.tokens.size >= minTokens)

for (let i = 0; i < docs.length; i += 1) {
  for (let j = i + 1; j < docs.length; j += 1) {
    const score = jaccard(docs[i].tokens, docs[j].tokens)
    if (score >= threshold) {
      findings.push({
        message: 'documents look duplicative',
        detail: `${docs[i].file} <-> ${docs[j].file}: ${score.toFixed(2)}`,
      })
    }
  }
}

finishByMode(audit, findings, mode)

function tokenSet(text) {
  const stop = new Set(['the', 'and', 'for', 'with', 'that', 'this', 'from', '으로', '에서', '하는'])
  return new Set(
    (text.toLowerCase().match(/[a-z0-9가-힣_:-]{3,}/g) ?? []).filter((token) => !stop.has(token)),
  )
}

function jaccard(left, right) {
  let intersection = 0
  for (const token of left) if (right.has(token)) intersection += 1
  return intersection / Math.max(1, left.size + right.size - intersection)
}

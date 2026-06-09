#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'
import { jaccard, tokenSet } from './audit-lib/text-similarity.mjs'

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

#!/usr/bin/env node

import { asStringArray, configSection } from './audit-lib/config.mjs'
import { exists, listFiles, readJson, readText } from './audit-lib/files.mjs'
import { createAudit, finishByMode } from './audit-lib/report.mjs'
import { firstHeading, jaccard, tokenSet } from './audit-lib/text-similarity.mjs'

const audit = createAudit('verify-doc-proposal')
const config = configSection('docsProposal')
const mode = config.mode ?? 'warn'
const proposalPath = process.env.HRNS_DOC_PROPOSAL ?? config.proposalPath ?? '.hrns/doc-proposal.json'
const roots = asStringArray(config.roots, ['docs', 'README.md'])
const threshold = Number(config.threshold ?? 0.52)
const titleThreshold = Number(config.titleThreshold ?? 0.45)
const findings = []

if (!exists(proposalPath)) {
  console.log(`verify-doc-proposal: PASS (${proposalPath} not present)`)
  process.exit(0)
}

const proposalFile = readJson(proposalPath)
const proposals = Array.isArray(proposalFile.proposals) ? proposalFile.proposals : [proposalFile]
const docs = listFiles({ roots })
  .filter((file) => file.endsWith('.md'))
  .map((file) => {
    const text = readText(file)
    return {
      path: file,
      title: firstHeading(text),
      tokens: tokenSet(text),
      titleTokens: tokenSet(firstHeading(text)),
    }
  })

for (const [index, proposal] of proposals.entries()) {
  const label = proposal.path ?? `proposals[${index}]`
  if (!proposal.path || typeof proposal.path !== 'string') {
    findings.push({ message: 'doc proposal missing path', detail: `proposals[${index}]` })
  }
  if (!proposal.purpose || typeof proposal.purpose !== 'string') {
    findings.push({ message: 'doc proposal missing purpose', detail: label })
  }

  const text = [proposal.title, proposal.purpose, proposal.summary, proposal.content]
    .filter((value) => typeof value === 'string')
    .join('\n')
  const proposedTokens = tokenSet(text)
  const proposedTitleTokens = tokenSet(proposal.title ?? '')
  if (proposedTokens.size < Number(config.minProposalTokens ?? 12)) {
    findings.push({ message: 'doc proposal is too thin to review for duplication', detail: label })
    continue
  }

  const matches = docs
    .filter((doc) => doc.path !== proposal.path)
    .map((doc) => ({
      doc,
      score: jaccard(proposedTokens, doc.tokens),
      titleScore: jaccard(proposedTitleTokens, doc.titleTokens),
    }))
    .filter((item) => item.score >= threshold || item.titleScore >= titleThreshold)
    .sort((a, b) => Math.max(b.score, b.titleScore) - Math.max(a.score, a.titleScore))

  if (matches.length === 0) continue

  const top = matches[0]
  const decision = proposal.decision ?? ''
  const target = proposal.target ?? ''
  const acceptedUpdate = decision === 'update_existing' && target === top.doc.path
  if (!acceptedUpdate) {
    findings.push({
      message: 'new doc proposal overlaps an existing document; update existing doc instead',
      detail: `${label} -> ${top.doc.path} (body ${top.score.toFixed(2)}, title ${top.titleScore.toFixed(2)})`,
    })
  }
}

finishByMode(audit, findings, mode)

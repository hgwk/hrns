#!/usr/bin/env node

import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { asStringArray, configSection } from './audit-lib/config.mjs'
import { listFiles, readText, ROOT } from './audit-lib/files.mjs'
import { firstHeading, headings, tokenSet } from './audit-lib/text-similarity.mjs'

const config = configSection('docsProposal')
const roots = asStringArray(config.roots, ['docs', 'README.md'])
const output = config.indexPath ?? '.hrns/docs-index.json'

const documents = listFiles({ roots })
  .filter((file) => file.endsWith('.md'))
  .map((file) => {
    const text = readText(file)
    const tokens = [...tokenSet(text)].sort()
    return {
      path: file,
      title: firstHeading(text),
      headings: headings(text).slice(0, 20),
      tokenCount: tokens.length,
      tokens,
    }
  })

const payload = {
  version: 1,
  generatedAt: new Date().toISOString(),
  roots,
  documents,
}

const target = join(ROOT, output)
mkdirSync(dirname(target), { recursive: true })
writeFileSync(target, `${JSON.stringify(payload, null, 2)}\n`)
console.log(`docs-index: wrote ${output} (${documents.length} document(s))`)

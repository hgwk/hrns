#!/usr/bin/env node

import { existsSync } from 'node:fs'
import { dirname, join, normalize as pathNormalize } from 'node:path'
import { listFiles, readText, ROOT } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'
import { asStringArray, configSection } from './audit-lib/config.mjs'

const audit = createAudit('verify-docs-symbol-sync')
const config = configSection('docs')

const DOCS = listFiles({ roots: asStringArray(config.roots, ['docs', 'README.md', 'AGENTS.md', 'CLAUDE.md']) }).filter((file) =>
  file.endsWith('.md'),
)
const MARKDOWN_LINK = /\[[^\]]+\]\((?!https?:|mailto:|#)([^)#]+)(?:#[^)]+)?\)/g

for (const file of DOCS) {
  const text = readText(file)
  const refs = []
  for (const match of text.matchAll(MARKDOWN_LINK)) refs.push(match[1])
  for (const ref of refs) {
    const clean = ref.replace(/^<|>$/g, '')
    if (clean.startsWith('/') || clean.includes('*')) continue
    const base = clean.startsWith('./') || clean.startsWith('../') ? dirname(file) : ''
    const target = pathNormalize(join(ROOT, base, clean))
    if (!existsSync(target)) audit.fail(`${file}: missing referenced artifact`, clean)
  }
}

audit.finish()

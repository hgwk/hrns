#!/usr/bin/env node

import { listFiles, listPackageJsons, readJson, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'

const audit = createAudit('verify-operational-surface')

const packageScripts = new Map()
for (const pkgFile of listPackageJsons()) {
  const pkg = readJson(pkgFile)
  for (const name of Object.keys(pkg.scripts ?? {})) {
    packageScripts.set(`${pkg.name ?? 'root'}:${name}`, pkgFile)
  }
}

for (const required of ['ci', 'audit', 'ledger:verify', 'line-audit']) {
  if (!packageScripts.has(`agent-zero:${required}`)) {
    audit.fail(`root package.json missing operational script "${required}"`)
  }
}

const PNPM_RUN = /pnpm(?:\s+-w|\s+-r|\s+--filter\s+[\w@./:-]+)*\s+(?:run\s+)?([A-Za-z0-9:_-]+)/g
for (const file of listFiles({ roots: ['docs', 'README.md', '.github'] })) {
  const text = readText(file)
  for (const match of text.matchAll(PNPM_RUN)) {
    const script = match[1]
    if (['install', 'exec', 'dlx', 'add', 'remove', 'run', 'tsc'].includes(script)) continue
    if (script.startsWith('-') || /^\d+$/.test(script)) continue
    const exists = [...packageScripts.keys()].some((key) => key.endsWith(`:${script}`))
    if (!exists) audit.fail(`${file}: documents missing pnpm script`, script)
  }
}

audit.finish()

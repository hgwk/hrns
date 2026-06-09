#!/usr/bin/env node

import { listFiles, listPackageJsons, readJson, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'
import { asStringArray, configSection } from './audit-lib/config.mjs'

const audit = createAudit('verify-operational-surface')
const config = configSection('operational')

const packageScripts = new Map()
for (const pkgFile of listPackageJsons(asStringArray(config.packageJsonRoots, ['package.json', 'packages']))) {
  const pkg = readJson(pkgFile)
  for (const name of Object.keys(pkg.scripts ?? {})) {
    packageScripts.set(`${pkg.name ?? 'root'}:${name}`, pkgFile)
  }
}

const rootName = readJson('package.json').name ?? 'root'
for (const required of asStringArray(config.requiredRootScripts, ['ci', 'audit', 'line-audit'])) {
  if (!packageScripts.has(`${rootName}:${required}`)) {
    audit.fail(`root package.json missing operational script "${required}"`)
  }
}

const PNPM_RUN = /pnpm(?:\s+-w|\s+-r|\s+--filter\s+[\w@./:-]+)*\s+(?:run\s+)?([A-Za-z0-9:_-]+)/g
for (const file of listFiles({ roots: asStringArray(config.docsRoots, ['docs', 'README.md', '.github']) })) {
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

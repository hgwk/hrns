#!/usr/bin/env node

import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'

const audit = createAudit('verify-thin-barrel-modules')

const barrels = listFiles({ roots: ['packages'] }).filter((file) => /\/index\.ts$/.test(file))

for (const file of barrels) {
  const text = readText(file)
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('//'))
  const exportLines = text.filter((line) => line.startsWith('export '))
  if (text.length > 0 && text.length === exportLines.length && exportLines.length <= 1) {
    audit.fail(`${file}: thin barrel has ${exportLines.length} export line(s)`)
  }
}

audit.finish()

#!/usr/bin/env node

import { readdirSync } from 'node:fs'
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'

const scriptsDir = join(process.cwd(), 'scripts')
const audits = readdirSync(scriptsDir)
  .filter((name) => /^verify-.*\.mjs$/.test(name))
  .sort()

let failures = 0
for (const audit of audits) {
  const result = spawnSync(process.execPath, [join('scripts', audit)], {
    stdio: 'inherit',
    env: process.env,
  })
  if (result.status !== 0) failures += 1
}

if (failures > 0) {
  console.error(`audit-all: FAIL (${failures}/${audits.length} audit(s) failed)`)
  process.exit(1)
}

console.log(`audit-all: PASS (${audits.length} audit(s) passed)`)

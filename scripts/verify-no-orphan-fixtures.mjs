#!/usr/bin/env node

import { basename } from 'node:path'
import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'

const audit = createAudit('verify-no-orphan-fixtures')

const fixtures = listFiles({ roots: ['packages'] }).filter((file) => file.includes('/fixtures/'))
const tests = listFiles({ roots: ['packages'] }).filter((file) =>
  /\.(test|spec)\.(ts|tsx|js|mjs)$/.test(file),
)
const haystack = tests.map((file) => readText(file)).join('\n')

for (const fixture of fixtures) {
  const parts = fixture.split('/')
  const fixtureRoot = parts.slice(0, parts.indexOf('fixtures') + 2).join('/')
  const leaf = basename(fixture)
  const dir = parts.slice(0, -1).join('/')
  const scenario = parts.at(-2) ?? ''
  if (
    !haystack.includes(fixtureRoot) &&
    !haystack.includes(dir) &&
    !haystack.includes(leaf) &&
    !haystack.includes(scenario)
  ) {
    audit.fail(`${fixture}: fixture is not referenced by active tests`)
  }
}

audit.finish()

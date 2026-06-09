#!/usr/bin/env node

import { spawnSync } from 'node:child_process'

const result = spawnSync(process.execPath, ['scripts/line-audit.mjs'], {
  stdio: 'inherit',
  env: process.env,
})

process.exit(result.status ?? 1)

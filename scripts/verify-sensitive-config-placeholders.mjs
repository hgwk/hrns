#!/usr/bin/env node

import { listFiles, readText } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'

const audit = createAudit('verify-sensitive-config-placeholders')

const ROOTS = ['.env.example', '.github', 'docs', 'infra']
const SECRET_PATTERNS = [
  { name: 'openai_key', re: /sk-[A-Za-z0-9_-]{20,}/g },
  { name: 'anthropic_key', re: /sk-ant-[A-Za-z0-9_-]{20,}/g },
  { name: 'aws_access_key', re: /AKIA[0-9A-Z]{16}/g },
  { name: 'agent_bearer', re: /agz_live_[A-Za-z0-9_-]{30,}/g },
  { name: 'private_key', re: /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/g },
]

for (const file of listFiles({ roots: ROOTS })) {
  const text = readText(file)
  for (const pattern of SECRET_PATTERNS) {
    const matches = [...text.matchAll(pattern.re)]
    if (matches.length > 0) audit.fail(`${file}: ${pattern.name}`, matches[0][0].slice(0, 80))
  }
}

audit.finish()

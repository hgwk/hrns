export function createAudit(name) {
  const findings = []
  return {
    fail(message, detail = '') {
      findings.push({ message, detail })
    },
    warn(message, detail = '') {
      console.error(`${name}: WARN - ${message}`)
      if (detail) console.error(`  ${detail}`)
    },
    finish() {
      if (findings.length === 0) {
        console.log(`${name}: PASS`)
        return
      }
      console.error(`${name}: FAIL (${findings.length} finding(s))`)
      for (const finding of findings) {
        console.error(`- ${finding.message}`)
        if (finding.detail) console.error(`  ${finding.detail}`)
      }
      process.exit(1)
    },
  }
}

export function unique(values) {
  return [...new Set(values)].sort()
}

export function finishByMode(audit, findings, mode = 'fail') {
  if (findings.length === 0) {
    audit.finish()
    return
  }
  if (mode === 'off') {
    console.log('audit disabled by config')
    return
  }
  for (const finding of findings) {
    if (mode === 'warn') audit.warn(finding.message, finding.detail)
    else audit.fail(finding.message, finding.detail)
  }
  audit.finish()
}

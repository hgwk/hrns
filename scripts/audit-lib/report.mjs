export function createAudit(name) {
  const findings = []
  return {
    fail(message, detail = '') {
      findings.push({ message, detail })
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

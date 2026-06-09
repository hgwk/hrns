#!/usr/bin/env node

import { basename, dirname, join, normalize as pathNormalize } from 'node:path'
import { existsSync } from 'node:fs'
import { listFiles, listPackageJsons, readJson, readText, ROOT } from './audit-lib/files.mjs'
import { createAudit } from './audit-lib/report.mjs'
import { asStringArray, configSection } from './audit-lib/config.mjs'

const audit = createAudit('verify-closed-world')
const config = configSection('closedWorld')

const sourceFiles = listFiles({ roots: asStringArray(config.roots, ['packages', 'scripts']) }).filter((file) =>
  /\.(ts|tsx|js|mjs)$/.test(file),
)
const entrypoints = new Set([
  'package.json',
  ...asStringArray(config.extraEntrypoints, []),
  ...sourceFiles.filter((file) => basename(file) === 'index.ts'),
  ...sourceFiles.filter((file) => file.endsWith('.test.ts') || file.endsWith('.test.tsx')),
  ...sourceFiles.filter((file) => /^scripts\/(?:audit-lib\/|verify-|ledger-|endpoint-)/.test(file)),
  ...sourceFiles.filter((file) => /\/src\/cli\/[^/]+\.ts$/.test(file)),
  // Vite/webview entries — loaded via index.html, not package.json. Walking these
  // closes the desktop main.ts → desktop-* module reachability chain.
  ...sourceFiles.filter((file) => /\/src\/main\.(ts|tsx)$/.test(file)),
])

for (const pkgFile of listPackageJsons()) addPackageEntrypoints(pkgFile)

const reachable = new Set(entrypoints)
const IMPORT_RE =
  /(?:import|export)\s+(?:[^'"]+\s+from\s+)?['"]([^'"]+)['"]|import\(['"]([^'"]+)['"]\)/g
let changed = true

while (changed) {
  changed = false
  for (const file of [...reachable]) {
    if (!sourceFiles.includes(file)) continue
    for (const target of localImports(file)) {
      if (!reachable.has(target)) {
        reachable.add(target)
        changed = true
      }
    }
  }
}

for (const file of sourceFiles) {
  if (isAlwaysAllowed(file)) continue
  if (!reachable.has(file))
    audit.fail(`${file}: not reachable from package/script/test entrypoints`)
}

audit.finish()

function localImports(file) {
  const out = []
  const text = readText(file)
  for (const match of text.matchAll(IMPORT_RE)) {
    const spec = match[1] ?? match[2]
    if (!spec.startsWith('.')) continue
    const resolved = resolveImport(file, spec)
    if (resolved) out.push(resolved)
  }
  return out
}

function resolveImport(file, spec) {
  const base = pathNormalize(join(ROOT, dirname(file), spec))
  const candidates = [
    '',
    '.ts',
    '.tsx',
    '.js',
    '.mjs',
    '/index.ts',
    '/index.tsx',
    '/index.js',
    spec.endsWith('.js') ? '' : '.js',
  ]
  for (const suffix of candidates) {
    for (const abs of equivalentSourcePaths(base + suffix)) {
      if (existsSync(abs)) return abs.slice(ROOT.length + 1)
    }
  }
  return ''
}

function equivalentSourcePaths(abs) {
  if (abs.includes('/dist/')) return [abs, abs.replace('/dist/', '/src/').replace(/\.js$/, '.ts')]
  if (abs.endsWith('.js')) return [abs, abs.slice(0, -3) + '.ts', abs.slice(0, -3) + '.tsx']
  if (abs.endsWith('.mjs')) return [abs, abs.slice(0, -4) + '.mjs']
  return [abs]
}

function addPackageEntrypoints(pkgFile) {
  const pkg = readJson(pkgFile)
  const base = dirname(pkgFile)
  const values = [pkg.main, pkg.module, pkg.types, ...Object.values(pkg.bin ?? {})]
  for (const value of Object.values(pkg.exports ?? {})) collectExportValues(value, values)
  for (const script of Object.values(pkg.scripts ?? {})) {
    for (const match of script.matchAll(/\b(?:tsx|node|vite)\s+([^\s]+)/g)) values.push(match[1])
  }
  for (const value of values) {
    if (typeof value !== 'string') continue
    const rel = pathNormalize(join(base, value))
    const resolved = resolveEntrypoint(rel)
    if (resolved) entrypoints.add(resolved)
  }
}

function collectExportValues(value, out) {
  if (typeof value === 'string') out.push(value)
  if (value && typeof value === 'object') {
    for (const nested of Object.values(value)) collectExportValues(nested, out)
  }
}

function resolveEntrypoint(rel) {
  const abs = join(ROOT, rel)
  for (const candidate of equivalentSourcePaths(abs)) {
    if (existsSync(candidate)) return candidate.slice(ROOT.length + 1)
  }
  return ''
}

function isAlwaysAllowed(file) {
  if (asStringArray(config.alwaysAllowedPatterns, []).some((pattern) => new RegExp(pattern).test(file))) {
    return true
  }
  return (
    file.endsWith('.d.ts') ||
    file.includes('/examples/') ||
    file.includes('/test/') ||
    file.includes('/tests/') ||
    file.includes('/fixtures/') ||
    file.includes('/specs/') ||
    file.endsWith('/vite.config.ts') ||
    file.endsWith('/playwright.config.ts') ||
    file.includes('/helpers/global-') ||
    file.includes('/helpers/runtime.ts') ||
    file.endsWith('/src/main.ts') ||
    file.endsWith('/src/main.tsx') ||
    file.endsWith('.config.js')
  )
}

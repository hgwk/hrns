#!/usr/bin/env node

import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { asNumber, asStringArray, configSection } from './audit-lib/config.mjs'

const ROOT = process.cwd()
const lineConfig = configSection('lineAudit')
const MAX_LINES = Number(process.env['LINE_AUDIT_MAX'] ?? asNumber(lineConfig.maxLines, 300))

const SCAN_ROOTS = asStringArray(lineConfig.roots, ['packages', 'scripts', 'infra'])
const EXTENSIONS = new Set(
  asStringArray(lineConfig.extensions, ['.ts', '.tsx', '.mjs', '.js', '.rs', '.sql']),
)
const EXCLUDED_PARTS = new Set([
  '.git',
  'node_modules',
  'target',
  'dist',
  'build',
  'coverage',
  '.next',
  'playwright-report',
])
const EXCLUDED_SUFFIXES = ['.tsbuildinfo']

const guidance = [
  `- Keep normal source files at <= ${MAX_LINES} lines.`,
  '- Split by responsibility: route registration vs handlers, transport vs parser, UI shell vs leaf components, fixtures vs assertions.',
  '- Prefer extracting named modules over hiding complexity in anonymous helpers.',
  '- Tests may exceed the limit only when a matrix is clearer in one file; otherwise split by behavior.',
  '- Generated/build output must stay outside tracked source or be excluded by path, not baseline.',
  '- There is no baseline exception list. If a file must exceed the limit temporarily, issue a refactor ticket and keep the build red until it is split.',
]

const files = []
for (const root of SCAN_ROOTS) walk(join(ROOT, root), files)

const overLimit = []

for (const file of files) {
  const rel = normalize(relative(ROOT, file))
  const lines = countLines(file)
  if (lines > MAX_LINES) overLimit.push({ rel, lines })
}

if (overLimit.length > 0) {
  console.error(`line-audit: FAIL (max ${MAX_LINES} lines)`)
  console.error('\nViolations:')
  for (const item of overLimit.sort((a, b) => b.lines - a.lines)) {
    console.error(`- ${item.rel}: ${item.lines} lines`)
  }
  printGuidance('Refactor guidance')
  process.exit(1)
}

console.log(`line-audit: PASS (max ${MAX_LINES} lines, ${files.length} files scanned)`)

function walk(dir, out) {
  let entries
  try {
    entries = readdirSync(dir, { withFileTypes: true })
  } catch {
    return
  }
  for (const entry of entries) {
    const path = join(dir, entry.name)
    const rel = normalize(relative(ROOT, path))
    if (shouldExclude(rel)) continue
    if (entry.isDirectory()) {
      walk(path, out)
      continue
    }
    if (!entry.isFile()) continue
    if (!EXTENSIONS.has(extension(entry.name))) continue
    out.push(path)
  }
}

function shouldExclude(rel) {
  const parts = rel.split('/')
  if (parts.some((part) => EXCLUDED_PARTS.has(part))) return true
  return EXCLUDED_SUFFIXES.some((suffix) => rel.endsWith(suffix))
}

function extension(name) {
  if (name.endsWith('.tsx')) return '.tsx'
  if (name.endsWith('.ts')) return '.ts'
  const idx = name.lastIndexOf('.')
  return idx === -1 ? '' : name.slice(idx)
}

function countLines(file) {
  const text = readFileSync(file, 'utf8')
  if (text.length === 0) return 0
  return text.endsWith('\n') ? text.split('\n').length - 1 : text.split('\n').length
}

function normalize(path) {
  return path.split(sep).join('/')
}

function printGuidance(title) {
  const log = console.error
  log(`\n${title}:`)
  for (const line of guidance) log(line)
}

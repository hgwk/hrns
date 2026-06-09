import { readdirSync, readFileSync, statSync } from 'node:fs'
import { extname, join, relative, sep } from 'node:path'

export const ROOT = process.cwd()

export const ACTIVE_ROOTS = [
  '.env.example',
  '.github',
  'docs',
  'infra',
  'package.json',
  'packages',
  'pnpm-workspace.yaml',
  'runtime',
  'scripts',
  'tsconfig.base.json',
]

export const TEXT_EXTENSIONS = new Set([
  '',
  '.css',
  '.cjs',
  '.js',
  '.json',
  '.md',
  '.mjs',
  '.rs',
  '.sql',
  '.ts',
  '.tsx',
  '.yaml',
  '.yml',
])

const EXCLUDED_PARTS = new Set([
  '.git',
  '.next',
  'archived',
  'build',
  'coverage',
  'dist',
  'node_modules',
  'playwright-report',
  'reference',
  'target',
])

export function normalize(path) {
  return path.split(sep).join('/')
}

export function readText(rel) {
  return readFileSync(join(ROOT, rel), 'utf8')
}

export function readJson(rel) {
  return JSON.parse(readText(rel))
}

export function listFiles(options = {}) {
  const roots = options.roots ?? ACTIVE_ROOTS
  const extensions = options.extensions ?? TEXT_EXTENSIONS
  const files = []
  for (const root of roots) walk(root, files, extensions)
  return [...new Set(files)].sort()
}

export function listPackageJsons(roots = ['package.json', 'packages']) {
  return listFiles({ roots, extensions: new Set(['.json']) }).filter(
    (file) => file === 'package.json' || file.endsWith('/package.json'),
  )
}

export function isTextFile(file) {
  return TEXT_EXTENSIONS.has(extname(file))
}

function walk(rel, out, extensions) {
  if (shouldExclude(rel)) return
  let stats
  try {
    stats = statSync(join(ROOT, rel))
  } catch {
    return
  }
  if (stats.isFile()) {
    if (extensions.has(extname(rel))) out.push(normalize(rel))
    return
  }
  if (!stats.isDirectory()) return
  for (const entry of readdirSync(join(ROOT, rel), { withFileTypes: true })) {
    walk(normalize(join(rel, entry.name)), out, extensions)
  }
}

function shouldExclude(rel) {
  return normalize(rel)
    .split('/')
    .some((part) => EXCLUDED_PARTS.has(part))
}

export function relativeToRoot(path) {
  return normalize(relative(ROOT, path))
}

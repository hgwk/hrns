import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { ROOT } from './files.mjs'

const DEFAULT_CONFIG = {
  scanRoots: ['packages', 'scripts', 'infra'],
  auditSets: {},
  lineAudit: {
    maxLines: 300,
    roots: ['packages', 'scripts', 'infra'],
    extensions: ['.ts', '.tsx', '.mjs', '.js', '.rs', '.sql'],
  },
  docs: {
    roots: ['docs', 'README.md', 'AGENTS.md', 'CLAUDE.md'],
  },
  fixtures: {
    roots: ['packages'],
    testRoots: ['packages'],
  },
  env: {
    example: '.env.example',
    roots: ['packages', 'scripts', 'infra'],
    requiredPrefixes: [],
    ignored: [
      'PATH',
      'NODE_ENV',
      'CI',
      'LINE_AUDIT_MAX',
    ],
  },
  operational: {
    requiredRootScripts: ['ci', 'audit', 'line-audit'],
    docsRoots: ['docs', 'README.md', '.github'],
    packageJsonRoots: ['package.json', 'packages'],
  },
  closedWorld: {
    roots: ['packages', 'scripts'],
    extraEntrypoints: [],
    alwaysAllowedPatterns: [],
  },
}

let cachedConfig

export function loadConfig() {
  if (cachedConfig) return cachedConfig
  const fileConfig = readJsonIfExists(process.env.HRNS_CONFIG ?? 'hrns.config.json')
  const packageConfig = readPackageConfig()
  cachedConfig = mergeConfig(DEFAULT_CONFIG, packageConfig, fileConfig)
  return cachedConfig
}

export function configSection(name) {
  return loadConfig()[name] ?? {}
}

export function asStringArray(value, fallback = []) {
  if (!Array.isArray(value)) return fallback
  return value.filter((item) => typeof item === 'string')
}

export function asNumber(value, fallback) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function readPackageConfig() {
  const pkg = readJsonIfExists('package.json')
  return pkg && typeof pkg.hrns === 'object' ? pkg.hrns : {}
}

function readJsonIfExists(rel) {
  const abs = join(ROOT, rel)
  if (!existsSync(abs)) return {}
  return JSON.parse(readFileSync(abs, 'utf8'))
}

function mergeConfig(...configs) {
  const out = {}
  for (const config of configs) mergeInto(out, config)
  return out
}

function mergeInto(target, source) {
  if (!source || typeof source !== 'object') return target
  for (const [key, value] of Object.entries(source)) {
    if (Array.isArray(value)) {
      target[key] = [...value]
      continue
    }
    if (value && typeof value === 'object') {
      target[key] = mergeInto({ ...(target[key] ?? {}) }, value)
      continue
    }
    target[key] = value
  }
  return target
}

#!/usr/bin/env node
// S5 — upstream reference drift gate.
//
// Policy (BRIEF S5): upstream latest stable must be rebaseable within one week.
// We treat Git tags on the upstream repository as the stable source of truth. npm is not
// authoritative for the tracked upstream reference in this repo.

import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
const ROOT = join(HERE, '..')
const REF_DIR = join(ROOT, 'reference/nanoclaw')
const PIN_PATH = join(REF_DIR, 'package.json')
const PIN_MANIFEST_PATH = join(ROOT, 'reference/upstream/nanoclaw-pin.json')

const UPSTREAM_REMOTE = process.env.UPSTREAM_NANOCLAW_REMOTE ?? 'https://github.com/qwibitai/nanoclaw.git'
const DRIFT_MAX_DAYS = Number(process.env.DRIFT_MAX_DAYS ?? '7')
const DRIFT_MAX_MINORS = Number(process.env.DRIFT_MAX_MINORS ?? '0')
const DRIFT_MAX_PATCHES = Number(process.env.DRIFT_MAX_PATCHES ?? '20')

function readLocalPin() {
  if (existsSync(PIN_MANIFEST_PATH)) {
    const manifest = JSON.parse(readFileSync(PIN_MANIFEST_PATH, 'utf8'))
    if (typeof manifest.version !== 'string') {
      throw new Error('reference/upstream/nanoclaw-pin.json missing "version"')
    }
    return manifest.version
  }
  if (!existsSync(PIN_PATH)) {
    throw new Error(`upstream pin not found at ${PIN_PATH}`)
  }
  const pkg = JSON.parse(readFileSync(PIN_PATH, 'utf8'))
  if (typeof pkg.version !== 'string') {
    throw new Error('reference/nanoclaw/package.json missing "version"')
  }
  return pkg.version
}

function readReferenceCommitDate() {
  if (existsSync(PIN_MANIFEST_PATH)) {
    const manifest = JSON.parse(readFileSync(PIN_MANIFEST_PATH, 'utf8'))
    if (typeof manifest.checked_at !== 'string') {
      throw new Error('reference/upstream/nanoclaw-pin.json missing "checked_at"')
    }
    const date = new Date(manifest.checked_at)
    if (Number.isNaN(date.getTime())) throw new Error(`invalid pin checked_at: ${manifest.checked_at}`)
    return date
  }
  const out = execFileSync('git', ['-C', REF_DIR, 'log', '-1', '--format=%cI'], {
    encoding: 'utf8',
    timeout: 10_000,
  }).trim()
  const date = new Date(out)
  if (Number.isNaN(date.getTime())) throw new Error(`invalid reference commit date: ${out}`)
  return date
}

function readUpstreamLatest() {
  const fromEnv = process.env.UPSTREAM_NANOCLAW_VERSION
  if (fromEnv) return normalizeVersion(fromEnv)

  try {
    const out = execFileSync('git', ['ls-remote', '--tags', '--refs', UPSTREAM_REMOTE, 'v*'], {
      encoding: 'utf8',
      timeout: 15_000,
    })
    const versions = out
      .split('\n')
      .map((line) => /refs\/tags\/v?(\d+\.\d+\.\d+)$/.exec(line)?.[1])
      .filter((v) => v !== undefined)
      .sort(compareSemver)
    return versions.at(-1) ?? null
  } catch (err) {
    if (process.env.ALLOW_OFFLINE_UPSTREAM === '1') {
      console.warn('[drift] upstream tag lookup failed; ALLOW_OFFLINE_UPSTREAM=1 so gate is skipped')
      return null
    }
    throw err
  }
}

function normalizeVersion(v) {
  return v.startsWith('v') ? v.slice(1) : v
}

function parseSemver(v) {
  const m = /^(\d+)\.(\d+)\.(\d+)$/.exec(normalizeVersion(v))
  if (!m) throw new Error(`invalid semver: ${v}`)
  return { major: Number(m[1]), minor: Number(m[2]), patch: Number(m[3]) }
}

function compareSemver(a, b) {
  const av = parseSemver(a)
  const bv = parseSemver(b)
  return av.major - bv.major || av.minor - bv.minor || av.patch - bv.patch
}

function daysSince(date, now = new Date(process.env.DRIFT_NOW ?? Date.now())) {
  return Math.floor((now.getTime() - date.getTime()) / 86_400_000)
}

function main() {
  const local = normalizeVersion(readLocalPin())
  const upstream = readUpstreamLatest()
  const referenceDate = readReferenceCommitDate()
  const ageDays = daysSince(referenceDate)

  console.log(`local pin          : ${local}`)
  console.log(`reference commit  : ${referenceDate.toISOString()} (${ageDays}d old)`)
  console.log(`upstream latest   : ${upstream ?? '<unknown>'}`)

  if (upstream == null) {
    console.log('drift              : unknown (offline skip)')
    process.exit(0)
  }

  const l = parseSemver(local)
  const u = parseSemver(upstream)
  if (u.major > l.major) {
    console.error(`FAIL: upstream major bumped (${local} -> ${upstream})`)
    process.exit(2)
  }

  const minorDrift = u.minor - l.minor
  const patchDrift = u.minor === l.minor ? u.patch - l.patch : Infinity
  console.log(`version drift      : ${minorDrift} minor / ${patchDrift === Infinity ? '∞' : patchDrift} patch`)

  if (minorDrift > DRIFT_MAX_MINORS) {
    console.error(`FAIL: minor drift ${minorDrift} > ${DRIFT_MAX_MINORS}`)
    process.exit(2)
  }
  if (patchDrift > DRIFT_MAX_PATCHES) {
    console.error(`FAIL: patch drift ${patchDrift} > ${DRIFT_MAX_PATCHES}`)
    process.exit(2)
  }
  if (compareSemver(local, upstream) < 0 && ageDays > DRIFT_MAX_DAYS) {
    console.error(`FAIL: upstream is ahead and reference checkout is ${ageDays}d old > ${DRIFT_MAX_DAYS}d`)
    process.exit(2)
  }

  console.log('PASS: within upstream rebase budget')
}

main()

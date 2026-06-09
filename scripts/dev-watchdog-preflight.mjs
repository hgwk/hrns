#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { existsSync, writeFileSync } from 'node:fs'
import { homedir } from 'node:os'
import {
  launchctl,
  listenerPids,
  loadLocalEnv,
  pidAlive,
  providerEnvPresence,
  readPid,
  sleep,
} from './lib/dev-runtime-common.mjs'
import { diskGuard, safeFsWrite } from './lib/disk-guard.mjs'
import { collectUpdateChannelSnapshot } from './lib/desktop-update-channel-preflight.mjs'

const root = process.cwd()
const uid = process.getuid?.() ?? Number(execFileSync('id', ['-u'], { encoding: 'utf8' }).trim())
const label = 'com.agent-zero.dev-watchdog'
const plistPath = `${homedir()}/Library/LaunchAgents/${label}.plist`
const controlPort = 4173
const gatewayPort = 4174
const updatePort = 4444
const controlBaseUrl = `http://127.0.0.1:${controlPort}`
const healthUrl = `${controlBaseUrl}/healthz`
const desktopConfigUrl = `${controlBaseUrl}/desktop-config`
const serverPidPath = `${root}/.pids/dev-server.pid`
const updatePidPath = `${root}/.pids/desktop-update-server.pid`
const watchdogPidPath = `${root}/.pids/dev-watchdog.pid`
const defaultDatabaseUrl = 'postgres://agentzero:agentzero@127.0.0.1:5432/agentzero'
const kbBlobRoot = `${root}/.agent-zero/kb-blobs`

loadLocalEnv(root)
execFileSync(process.execPath, ['scripts/dev-watchdog.mjs', 'repair'], {
  cwd: root,
  stdio: 'ignore',
})

const result = await waitForReady(30_000)
console.log(JSON.stringify(result, null, 2))
if (!result.ok) process.exitCode = 1

async function waitForReady(timeoutMs) {
  const deadline = Date.now() + timeoutMs
  let snapshot = await collectSnapshot()
  while (!snapshot.ok && Date.now() < deadline) {
    await sleep(1000)
    snapshot = await collectSnapshot()
  }
  return snapshot
}

async function collectSnapshot() {
  const health = await fetchJson(healthUrl)
  const desktopConfig = await fetchJson(desktopConfigUrl)
  const demoBaseline = await collectDemoBaselineSnapshot(desktopConfig.body)
  const kbIngestion = await collectKbIngestionSnapshot()
  const updateChannel = collectUpdateChannelSnapshot()
  const controlPids = listenerPids(controlPort)
  const gatewayPids = listenerPids(gatewayPort)
  const updatePids = listenerPids(updatePort)
  syncServerPid(controlPids, gatewayPids)
  syncUpdatePid(updatePids)
  const serverPid = readPid(serverPidPath)
  const updatePid = readPid(updatePidPath)
  const watchdogPid = readPid(watchdogPidPath)
  const launchctlState = launchctl(['print', `gui/${uid}/${label}`], { ignoreError: true })
  const launchAgent = {
    label,
    plistPath,
    installed: existsSync(plistPath),
    loaded: launchctlState.includes(label),
  }
  const config = sanitizeDesktopConfig(desktopConfig.body)
  const disk = diskGuard(`${root}/.pids`)
  const ok =
    health.ok &&
    desktopConfig.ok &&
    config.ok &&
    launchAgent.installed &&
    launchAgent.loaded &&
    controlPids.length === 1 &&
    gatewayPids.length === 1 &&
    controlPids[0] === gatewayPids[0] &&
    config.realProvider === true &&
    demoBaseline.ok &&
    kbIngestion.ok &&
    updateChannel.ok &&
    disk.ok

  return {
    ok,
    disk,
    fixedPorts: { control: controlPort, gateway: gatewayPort, update: updatePort },
    listeners: { controlPids, gatewayPids, updatePids },
    pids: {
      serverPid,
      serverPidAlive: serverPid ? pidAlive(serverPid) : false,
      updatePid,
      updatePidAlive: updatePid ? pidAlive(updatePid) : false,
      watchdogPid: watchdogPid && pidAlive(watchdogPid) ? watchdogPid : null,
      watchdogPidAlive: watchdogPid ? pidAlive(watchdogPid) : false,
    },
    health,
    desktopConfig: config,
    demoBaseline,
    kbIngestion,
    updateChannel,
    providerEnv: providerEnvPresence(),
    launchAgent,
  }
}

async function collectDemoBaselineSnapshot(configBody) {
  const token = configBody?.data?.bearerToken
  if (!token) return { ok: false, checked: false, error: 'desktop bearer missing' }
  const [mcp, mine, catalog] = await Promise.all([
    fetchAuthedJson('/me/mcp/presets', token),
    fetchAuthedJson('/me/skills', token),
    fetchAuthedJson('/me/skills/catalog', token),
  ])
  const mcpCount = dataCount(mcp.body)
  const mySkillCount = itemCount(mine.body)
  const catalogSkillCount = itemCount(catalog.body)
  return {
    ok: mcp.ok && mine.ok && catalog.ok && mcpCount > 0 && mySkillCount + catalogSkillCount > 0,
    checked: true,
    mcpCount,
    mySkillCount,
    catalogSkillCount,
  }
}

async function fetchAuthedJson(path, token) {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 2000)
  try {
    const res = await fetch(`${controlBaseUrl}${path}`, {
      signal: controller.signal,
      headers: { authorization: `Bearer ${token}` },
    })
    return { ok: res.ok, statusCode: res.status, body: await res.json().catch(() => null) }
  } catch (error) {
    return {
      ok: false,
      statusCode: null,
      error: error instanceof Error ? error.message : String(error),
    }
  } finally {
    clearTimeout(timer)
  }
}

function dataCount(body) {
  return Array.isArray(body?.data) ? body.data.length : itemCount(body)
}

function itemCount(body) {
  return Array.isArray(body?.data?.items) ? body.data.items.length : 0
}

async function collectKbIngestionSnapshot() {
  const databaseUrl = process.env.DATABASE_URL ?? defaultDatabaseUrl
  try {
    const out = execFileSync(
      'psql',
      [
        databaseUrl,
        '-At',
        '-c',
        `
          SELECT coalesce(json_agg(row_to_json(q)), '[]'::json)
          FROM (
            SELECT r.id, r.status, d.blob_sha, s.tenant_id, s.id AS source_id
            FROM kb_ingestion_runs r
            JOIN kb_documents d ON d.id = r.document_id
            JOIN kb_sources s ON s.id = d.source_id
            WHERE r.status = 'running'
               OR coalesce(r.error_message, '') ILIKE '%ENOENT%'
               OR coalesce(r.error_message, '') ILIKE '%no such file%'
            ORDER BY r.started_at DESC
            LIMIT 50
          ) q
        `,
      ],
      {
        encoding: 'utf8',
        env: { ...process.env, PGPASSWORD: process.env.PGPASSWORD ?? 'agentzero' },
      },
    )
    const rows = JSON.parse(out.trim() || '[]')
    const stale = rows.map((row) => ({
      id: row.id,
      status: row.status,
      sourceId: row.source_id ?? row.sourceId,
      tenantId: row.tenant_id ?? row.tenantId,
      blobExists: row.blob_sha ? existsSync(blobPath(row.blob_sha)) : false,
    }))
    const missingBlobRows = stale.filter((row) => !row.blobExists).length
    return {
      ok: stale.length === 0,
      checked: true,
      staleRows: stale.length,
      missingBlobRows,
      sample: stale.slice(0, 5),
    }
  } catch (error) {
    return {
      ok: false,
      checked: false,
      error: error instanceof Error ? error.message : String(error),
    }
  }
}

function blobPath(sha) {
  return `${kbBlobRoot}/${sha.slice(0, 2)}/${sha.slice(2, 4)}/${sha}`
}

function syncServerPid(controlPids, gatewayPids) {
  const same = controlPids.find((pid) => gatewayPids.includes(pid))
  if (same)
    safeFsWrite(
      () => writeFileSync(serverPidPath, `${same}\n`),
      (message) => process.stderr.write(`[preflight] ${message}\n`),
    )
}

function syncUpdatePid(updatePids) {
  if (updatePids.length === 1)
    safeFsWrite(
      () => writeFileSync(updatePidPath, `${updatePids[0]}\n`),
      (message) => process.stderr.write(`[preflight] ${message}\n`),
    )
}

function sanitizeDesktopConfig(body) {
  const data = body?.data ?? {}
  const demoProvider = data.demoProvider ?? {}
  const providerProfile = data.providerProfile ?? {}
  return {
    ok: Boolean(data.controlBaseUrl && data.gatewayUrl && data.bearerToken),
    controlBaseUrl: data.controlBaseUrl ?? null,
    gatewayUrl: data.gatewayUrl ?? null,
    bearerPrefix: redactToken(data.bearerToken),
    workspaceCwd: data.workspaceCwd ?? null,
    providerId: demoProvider.providerId ?? providerProfile.providerId ?? null,
    providerKind: demoProvider.kind ?? providerProfile.providerKind ?? null,
    model: demoProvider.model ?? providerProfile.model ?? null,
    label: demoProvider.label ?? null,
    realProvider: demoProvider.realProvider ?? null,
    tenantId: data.tenantId ?? null,
    userId: data.userId ?? null,
    endpointId: data.endpointId ?? null,
  }
}

async function fetchJson(url) {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 2000)
  try {
    const res = await fetch(url, { signal: controller.signal })
    const body = await res.json().catch(() => null)
    return { ok: res.ok, statusCode: res.status, body }
  } catch (error) {
    return {
      ok: false,
      statusCode: null,
      error: error instanceof Error ? error.message : String(error),
    }
  } finally {
    clearTimeout(timer)
  }
}

function redactToken(token) {
  return typeof token === 'string' && token ? `${token.slice(0, 12)}...` : null
}

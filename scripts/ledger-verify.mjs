#!/usr/bin/env node

import { existsSync, readFileSync } from 'node:fs'

const config = JSON.parse(readFileSync('ledger/config.json', 'utf8'))

const LEDGERS = [
  {
    path: config.ledgers.tickets,
    legacyRequired: [
      'n',
      'ts',
      'parent_ticket',
      'ticket',
      'agent',
      'role',
      'status',
      'task',
      'scope',
      'paths',
      'blocked_by',
      'branch',
    ],
    canonicalRequired: [
      'n',
      'ts',
      'id',
      'parent',
      'type',
      'state',
      'area',
      'priority',
      'title',
      'owner',
      'blocked_by',
      'acceptance',
      'evidence',
      'event',
    ],
  },
  {
    path: config.ledgers.worklogs,
    legacyRequired: [
      'n',
      'ts',
      'ticket',
      'agent',
      'task',
      'scope',
      'result',
      'paths',
      'commands',
      'notes',
      'branch',
      'commit',
    ],
    canonicalRequired: [
      'n',
      'ts',
      'ticket',
      'actor',
      'title',
      'summary',
      'paths',
      'commands',
      'notes',
    ],
  },
]

const ISO8601_UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/

function fail(message) {
  console.error(message)
  process.exitCode = 1
}

function warn(message) {
  console.warn(`WARN ${message}`)
}

function parseRows(path) {
  if (!existsSync(path)) {
    return []
  }
  return readFileSync(path, 'utf8')
    .split('\n')
    .map((line, index) => ({ line, lineNumber: index + 1 }))
    .filter(({ line }) => line.trim().length > 0)
}

function isCanonicalTicket(row) {
  return 'id' in row || 'state' in row || 'event' in row
}

function isCanonicalWorklog(row) {
  return 'actor' in row || 'summary' in row
}

function ticketID(row) {
  return row.ticket || row.id || ''
}

function ticketState(row) {
  return row.status || row.state || ''
}

function blockerList(row) {
  return Array.isArray(row.blocked_by) ? row.blocked_by : []
}

function isAuditRole(role) {
  return ['audit', 'auditor', 'reviewer'].includes(role)
}

function isDoneLike(state) {
  return ['done', 'cancelled', 'dropped'].includes(state)
}

const parsed = new Map()

for (const ledger of LEDGERS) {
  let previousTs = ''
  const rows = parseRows(ledger.path)
  if (ledger.optional && rows.length === 0) {
    console.log(`${ledger.path}: optional ledger not present`)
    continue
  }

  rows.forEach(({ line, lineNumber }, index) => {
    let row
    try {
      row = JSON.parse(line)
    } catch (error) {
      fail(`${ledger.path}:${lineNumber} invalid JSON: ${error.message}`)
      return
    }
    const parsedRows = parsed.get(ledger.path) ?? []
    parsedRows.push({ ...row, __lineNumber: lineNumber })
    parsed.set(ledger.path, parsedRows)

    const expectedN = index + 1
    if (row.n !== expectedN) {
      fail(`${ledger.path}:${lineNumber} expected n=${expectedN}, got ${row.n}`)
    }

    if (typeof row.ts !== 'string' || !ISO8601_UTC.test(row.ts)) {
      fail(`${ledger.path}:${lineNumber} ts must be ISO8601 UTC`)
    } else if (previousTs && row.ts < previousTs) {
      fail(`${ledger.path}:${lineNumber} ts decreases: ${previousTs} > ${row.ts}`)
    } else {
      previousTs = row.ts
    }

    const canonical =
      ledger.path === config.ledgers.tickets ? isCanonicalTicket(row) : isCanonicalWorklog(row)
    const required = canonical ? ledger.canonicalRequired : ledger.legacyRequired
    for (const field of required) {
      if (!(field in row)) {
        fail(`${ledger.path}:${lineNumber} missing required field: ${field}`)
      }
    }

    if (ledger.path === config.ledgers.tickets && !canonical && (!row.ticket || !row.task)) {
      warn(`${ledger.path}:${lineNumber} ghost ticket row has empty ticket/task`)
    }
    if (
      ledger.path === config.ledgers.worklogs &&
      !canonical &&
      (!row.task || ('ticket' in row && row.ticket === ''))
    ) {
      warn(`${ledger.path}:${lineNumber} ghost worklog row has empty ticket/task`)
    }

    if ('paths' in row && !Array.isArray(row.paths)) {
      fail(`${ledger.path}:${lineNumber} paths must be an array`)
    }
    if ('acceptance' in row && !Array.isArray(row.acceptance)) {
      fail(`${ledger.path}:${lineNumber} acceptance must be an array`)
    }
    if ('evidence' in row && !Array.isArray(row.evidence)) {
      fail(`${ledger.path}:${lineNumber} evidence must be an array`)
    }
    if (
      'event' in row &&
      (row.event === null || typeof row.event !== 'object' || Array.isArray(row.event))
    ) {
      fail(`${ledger.path}:${lineNumber} event must be an object`)
    }
    if ('worklogs' in row && !Array.isArray(row.worklogs)) {
      fail(`${ledger.path}:${lineNumber} worklogs must be an array`)
    }
    if ('blocked_by' in row && !Array.isArray(row.blocked_by)) {
      fail(`${ledger.path}:${lineNumber} blocked_by must be an array`)
    }
    if ('commands' in row && !Array.isArray(row.commands)) {
      fail(`${ledger.path}:${lineNumber} commands must be an array`)
    }
  })

  console.log(`${ledger.path}: ${rows.length} rows OK`)
}

const ticketRows = parsed.get(config.ledgers.tickets) ?? []
const latestTickets = new Map()
for (const row of ticketRows) {
  const id = ticketID(row)
  if (id) latestTickets.set(id, row)
}
for (const row of latestTickets.values()) {
  const state = ticketState(row)
  if (isDoneLike(state)) continue
  const blockers = blockerList(row)
  const stale = blockers.filter((ticket) =>
    isDoneLike(ticketState(latestTickets.get(ticket) ?? {})),
  )
  if (stale.length > 0) {
    warn(
      `${config.ledgers.tickets}:${row.__lineNumber} ${ticketID(row)} has stale done/cancelled blockers: ${stale.join(', ')}`,
    )
  }
  const unresolved = blockers.filter(
    (ticket) => !isDoneLike(ticketState(latestTickets.get(ticket) ?? {})),
  )
  if (state === 'blocked' && unresolved.length === 0) {
    warn(
      `${config.ledgers.tickets}:${row.__lineNumber} ${ticketID(row)} is blocked without unresolved blockers`,
    )
  }
}

if (config.schema_version === 1) {
  const worklogRows = parsed.get(config.ledgers.worklogs) ?? []
  const worklogTickets = new Set(worklogRows.map((row) => row.ticket).filter(Boolean))
  for (const row of latestTickets.values()) {
    if (!isCanonicalTicket(row) || ticketState(row) !== 'done') continue
    const id = ticketID(row)
    const event = row.event ?? {}
    const reviewedN = event.reviewed_n ?? row.reviewed_n
    if (!isAuditRole(event.role)) {
      warn(
        `${config.ledgers.tickets}:${row.__lineNumber} ${id} done row is not an audit/reviewer role`,
      )
    }
    if (event.result !== 'pass') {
      warn(
        `${config.ledgers.tickets}:${row.__lineNumber} ${id} done row event.result should be pass`,
      )
    }
    if (!reviewedN) {
      warn(`${config.ledgers.tickets}:${row.__lineNumber} ${id} done row has no reviewed_n`)
    }
    if (!Array.isArray(row.evidence) || row.evidence.length === 0) {
      warn(`${config.ledgers.tickets}:${row.__lineNumber} ${id} done row has empty evidence`)
    }
    const blockers = blockerList(row)
    if (blockers.length > 0) {
      warn(`${config.ledgers.tickets}:${row.__lineNumber} ${id} done row still has blockers`)
    }
    if (['task', 'bug'].includes(row.type) && !worklogTickets.has(id)) {
      warn(`${config.ledgers.tickets}:${row.__lineNumber} ${id} done row has no linked worklog`)
    }
  }
}

if (!existsSync(config.ledgers.goal)) {
  fail(`${config.ledgers.goal}: missing goal snapshot`)
} else {
  let goal
  try {
    goal = JSON.parse(readFileSync(config.ledgers.goal, 'utf8'))
  } catch (error) {
    fail(`${config.ledgers.goal}: invalid JSON: ${error.message}`)
  }

  for (const field of ['$schema', 'track', 'version', 'updated', 'source_of_truth']) {
    if (!(field in goal)) {
      fail(`${config.ledgers.goal}: missing required field: ${field}`)
    }
  }
  if (goal.updated && !ISO8601_UTC.test(goal.updated)) {
    fail(`${config.ledgers.goal}: updated must be ISO8601 UTC`)
  }
  console.log(`${config.ledgers.goal}: goal snapshot OK`)
}

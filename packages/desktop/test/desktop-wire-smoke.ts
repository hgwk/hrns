// MF.4 wire smoke — drives the exact same agent-client.ts the Tauri webview uses,
// from node. Proves: /desktop-config → bootstrap → ws ticket → ws → initialize →
// session/new → session/prompt → assistant output stream.

import WebSocket from 'ws'
import {
  DesktopAgentClient,
  applyDesktopUpdate,
  type DesktopMessage,
} from '../src/agent-client.js'
import {
  assertReplay,
  parseNonnegativeInt,
  parseOptionalNonnegativeInt,
  parseRequiredTexts,
  parseSmokeAttachments,
  parseSmokeSourceIntent,
  waitForReplay,
} from './desktop-wire-smoke-helpers.js'
;(globalThis as unknown as { WebSocket: typeof WebSocket }).WebSocket =
  WebSocket as unknown as typeof globalThis.WebSocket

async function main(): Promise<void> {
  const controlBaseUrl = process.env.CONTROL_URL ?? 'http://127.0.0.1:4173'
  console.log(`[smoke] GET ${controlBaseUrl}/desktop-config`)
  const cfgRes = await fetch(`${controlBaseUrl}/desktop-config`)
  if (!cfgRes.ok) throw new Error(`desktop-config ${cfgRes.status}`)
  const cfgBody = (await cfgRes.json()) as {
    data: {
      controlBaseUrl: string
      gatewayUrl: string
      bearerToken: string
      bootstrap: {
        tenantId: string
        userId: string
        providerId?: string
        model: string
        policyId: string
      }
      demoProvider?: {
        mode: string
        providerId: string
        kind: string
        model: string
        label: string
        realProvider: boolean
      }
      workspaceCwd?: string
      endpoint?: { id: string; state?: 'enrolled' | 'active' | 'frozen' | 'revoked' }
    }
  }
  const demoProvider = cfgBody.data.demoProvider
  console.log(
    `[smoke] config: gateway=${cfgBody.data.gatewayUrl} provider=${cfgBody.data.bootstrap.providerId ?? '(default)'} model=${cfgBody.data.bootstrap.model}`,
  )
  if (demoProvider) {
    console.log(
      `[smoke] demo provider: ${demoProvider.label} mode=${demoProvider.mode} kind=${demoProvider.kind} real=${demoProvider.realProvider}`,
    )
  }
  if (!demoProvider?.realProvider) {
    console.error('[smoke] FAIL: desktop smoke requires OpenAI or Anthropic real provider')
    process.exit(4)
  }
  if (demoProvider.kind !== 'openai' && demoProvider.kind !== 'anthropic') {
    console.error(`[smoke] FAIL: unsupported provider kind ${demoProvider.kind}`)
    process.exit(5)
  }
  if (demoProvider.providerId !== cfgBody.data.bootstrap.providerId) {
    console.error(
      `[smoke] FAIL: demo provider profile does not match bootstrap provider. profile=${demoProvider.providerId} bootstrap=${cfgBody.data.bootstrap.providerId ?? '(default)'}`,
    )
    process.exit(6)
  }
  if (demoProvider.model !== cfgBody.data.bootstrap.model) {
    console.error(
      `[smoke] FAIL: real provider profile model mismatch. profile=${demoProvider.model} bootstrap=${cfgBody.data.bootstrap.model}`,
    )
    process.exit(7)
  }

  const client = new DesktopAgentClient({
    controlBaseUrl: cfgBody.data.controlBaseUrl,
    gatewayUrl: cfgBody.data.gatewayUrl,
    bearerToken: cfgBody.data.bearerToken,
    bootstrap: cfgBody.data.bootstrap,
    ...(cfgBody.data.endpoint?.id ? { endpoint: { id: cfgBody.data.endpoint.id } } : {}),
  })

  let messages: DesktopMessage[] = []
  let currentSessionId = ''
  let deniedPermission = false
  let runtimeError: string | null = null
  client.onUpdate((update) => {
    messages = applyDesktopUpdate(messages, update)
    const event = update as { kind?: string; payload?: unknown }
    if (event.kind === 'error') {
      const payload = event.payload as { message?: unknown } | undefined
      runtimeError = typeof payload?.message === 'string' ? payload.message : 'runtime error'
    }
    if (event.kind === 'permission_request' && currentSessionId) {
      const payload = event.payload as { tool_call_id?: string } | undefined
      if (payload?.tool_call_id) {
        const shouldDeny =
          process.env.DESKTOP_SMOKE_PERMISSION_POLICY === 'deny_once_then_allow' &&
          !deniedPermission
        if (shouldDeny) deniedPermission = true
        client.respondPermission(
          currentSessionId,
          payload.tool_call_id,
          shouldDeny ? 'deny' : 'allow_once',
        )
      }
    }
  })

  const snap = await client.bootstrapSnapshot()
  console.log(`[smoke] snapshot: ${snap.configSnapshotId}`)
  await client.connect()
  console.log('[smoke] WS connected + initialize ok')
  const workspaceCwd = cfgBody.data.workspaceCwd ?? '/'
  const sess = await client.newSession({
    cwd: workspaceCwd,
    configSnapshotId: snap.configSnapshotId,
  })
  currentSessionId = sess.sessionId
  console.log(`[smoke] session: ${sess.sessionId}`)
  const promptText =
    process.env.DESKTOP_SMOKE_PROMPT ?? 'Reply with: desktop smoke real provider ok'
  const attachments = parseSmokeAttachments(process.env.DESKTOP_SMOKE_ATTACHMENTS_JSON)
  const sourceIntent = parseSmokeSourceIntent(process.env.DESKTOP_SMOKE_SOURCE_INTENT_JSON)
  console.log(`[smoke] prompt: "${promptText}"`)
  if (attachments.length > 0) {
    console.log(`[smoke] attachments: ${attachments.map((a) => a.path).join(', ')}`)
  }
  if (sourceIntent) console.log(`[smoke] sourceIntent: ${JSON.stringify(sourceIntent)}`)
  const promptResult = await withTimeout(
    client.prompt(sess.sessionId, promptText, attachments, sourceIntent),
    smokeTimeoutMs(),
    'initial prompt',
  )
  console.log(`[smoke] prompt result:`, JSON.stringify(promptResult))
  await waitForAssistant(() => messages, 0, () => runtimeError)
  const followupPrompt = process.env.DESKTOP_SMOKE_FOLLOWUP_PROMPT
  if (followupPrompt) {
    console.log(`[smoke] follow-up prompt: "${followupPrompt}"`)
    const beforeFollowup = messages.length
    const followupResult = await withTimeout(
      client.prompt(sess.sessionId, followupPrompt),
      smokeTimeoutMs(),
      'follow-up prompt',
    )
    console.log(`[smoke] follow-up result:`, JSON.stringify(followupResult))
    await waitForAssistant(() => messages, beforeFollowup, () => runtimeError)
  }
  console.log(`[smoke] messages count: ${messages.length}`)
  for (const m of messages) {
    console.log(`  - ${m.role}${m.streaming ? '*' : ''}: ${m.text}`)
  }
  if (messages.length === 0) {
    console.error('[smoke] FAIL: no assistant messages captured')
    process.exit(2)
  }
  const assistant = messages.find((m) => m.role === 'assistant')
  if (!assistant || assistant.text.trim().length === 0) {
    console.error(
      `[smoke] FAIL: assistant did not return text. Got: ${assistant?.text ?? '(none)'}`,
    )
    process.exit(3)
  }
  const replay = await waitForReplay(client, {
    sessionId: sess.sessionId,
    providerKind: demoProvider.kind,
    promptText,
    expectations: replayExpectations(),
  })
  console.log(
    `[smoke] replay: ui=${replay.ui_messages.length} provider_history=${replay.provider_history.length} provider_conversation=${replay.provider_conversation.provider}/${replay.provider_conversation.state}`,
  )
  assertReplay(replay, {
    sessionId: sess.sessionId,
    providerKind: demoProvider.kind,
    promptText,
    expectations: replayExpectations(),
  })
  const expectPattern = process.env.DESKTOP_SMOKE_EXPECT_PATTERN
  if (expectPattern) {
    const combinedText = messages.map((m) => `${m.role}: ${m.text}`).join('\n')
    const re = new RegExp(expectPattern, 'i')
    if (!re.test(combinedText)) {
      console.error(`[smoke] FAIL: expected pattern ${expectPattern} was not observed.`)
      console.error(combinedText)
      process.exit(8)
    }
  }
  const requiredTexts = parseRequiredTexts(process.env.DESKTOP_SMOKE_EXPECT_TEXTS)
  if (requiredTexts.length > 0) {
    const assistantText = messages
      .filter((m) => m.role === 'assistant')
      .map((m) => m.text)
      .join('\n')
    const missing = requiredTexts.filter((text) => !assistantText.includes(text))
    if (missing.length > 0) {
      console.error(`[smoke] FAIL: required assistant text missing: ${missing.join(', ')}`)
      console.error(assistantText)
      process.exit(9)
    }
  }
  const requiredToolTexts = parseRequiredTexts(process.env.DESKTOP_SMOKE_EXPECT_TOOL_TEXTS)
  if (requiredToolTexts.length > 0) {
    const toolText = messages
      .filter((m) => m.role === 'tool')
      .map((m) => [m.text, m.toolCommand, m.toolOutput].filter(Boolean).join('\n'))
      .join('\n')
    const missing = requiredToolTexts.filter((text) => !toolText.includes(text))
    if (missing.length > 0) {
      console.error(`[smoke] FAIL: required tool text missing: ${missing.join(', ')}`)
      console.error(toolText)
      process.exit(20)
    }
  }
  const followupRequiredTexts = parseRequiredTexts(process.env.DESKTOP_SMOKE_FOLLOWUP_EXPECT_TEXTS)
  if (followupRequiredTexts.length > 0) {
    const assistantText = messages
      .filter((m) => m.role === 'assistant')
      .map((m) => m.text)
      .join('\n')
    const missing = followupRequiredTexts.filter((text) => !assistantText.includes(text))
    if (missing.length > 0) {
      console.error(`[smoke] FAIL: required follow-up assistant text missing: ${missing.join(', ')}`)
      console.error(assistantText)
      process.exit(10)
    }
  }
  client.close()
  console.log('[smoke] PASS: assistant output captured via configured provider')
  process.exit(0)
}

function replayExpectations() {
  return {
    minToolMessages: parseNonnegativeInt(process.env.DESKTOP_SMOKE_EXPECT_MIN_TOOL_MESSAGES, 0),
    maxToolMessages: parseOptionalNonnegativeInt(process.env.DESKTOP_SMOKE_EXPECT_MAX_TOOL_MESSAGES),
    toolNames: parseRequiredTexts(process.env.DESKTOP_SMOKE_EXPECT_TOOL_NAMES),
  }
}

void main().catch((e) => {
  console.error('[smoke] error:', e)
  process.exit(1)
})

async function waitForAssistant(
  read: () => DesktopMessage[],
  afterIndex = 0,
  readError: () => string | null = () => null,
): Promise<void> {
  const timeoutMs = smokeTimeoutMs()
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const error = readError()
    if (error) throw new Error(`runtime error: ${error}`)
    const assistant = read()
      .slice(afterIndex)
      .find((m) => m.role === 'assistant' && !m.streaming)
    if (assistant && !assistant.streaming && assistant.text.trim().length > 0) return
    await new Promise((r) => setTimeout(r, 100))
  }
  throw new Error(`assistant message timeout after ${timeoutMs}ms`)
}

function smokeTimeoutMs(): number {
  return Number(process.env.DESKTOP_SMOKE_TIMEOUT_MS ?? 45_000)
}

async function withTimeout<T>(promise: Promise<T>, timeoutMs: number, label: string): Promise<T> {
  let timer: NodeJS.Timeout | undefined
  try {
    return await Promise.race([
      promise,
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error(`${label} timeout after ${timeoutMs}ms`)), timeoutMs)
      }),
    ])
  } finally {
    if (timer) clearTimeout(timer)
  }
}

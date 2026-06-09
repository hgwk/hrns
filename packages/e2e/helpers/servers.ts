// Boots control plane + gateway + web static server with real core wiring (sessions/router/bus/
// cancel + prompt-runner). Provider selection: providers.kind via DbProviderResolver →
// ProviderFactory, with decrypted provider_keys for OpenAI/Anthropic.
// Each on ephemeral or fixed test port. global-setup writes ports + seed to .runtime.json.

import { Kysely, PostgresDialect } from 'kysely'
import pg from 'pg'
import {
  buildControlServer,
  loadEnvelopeCryptoFromEnv,
  type ControlDatabase,
} from '@agent-zero/control'
import type { Id } from '@agent-zero/shared'
import { BlobStore, IngestionWorker, KbIngestListener, type KbDatabase } from '@agent-zero/kb'
import { startGatewayRuntime } from './gateway-runtime.js'
import { startWebStaticServer } from './web-static-server.js'

export interface BootResult {
  controlUrl: string
  gatewayUrl: string
  webUrl: string
  close(): Promise<void>
}

export interface BootOptions {
  // Fixed port for stable local web; undefined means ephemeral e2e port.
  controlPort?: number
  gatewayPort?: number
  permissionTimeoutMs?: number
  // When set, control plane serves the SPA bundle and injects `window.__AGENT_ZERO__`.
  webBundle?: {
    distPath: string
    // Caller fills in any keys it knows up front; boot() merges the gatewayUrl after the
    // gateway listens (host can't know the port until then unless gatewayPort is fixed).
    runtimeConfig: Record<string, unknown>
    updateChannelRootPath?: string
  }
  kbIngest?: {
    blobRootDir: string
    providerId: string
    resolveApiKey: () => Promise<string>
  }
}

export async function boot(
  databaseUrl: string,
  kek: string,
  options: BootOptions = {},
): Promise<BootResult> {
  const controlDb = new Kysely<ControlDatabase>({
    dialect: new PostgresDialect({ pool: new pg.Pool({ connectionString: databaseUrl }) }),
  })
  // Build gateway port up front so the injected SPA config can reference it.
  const gatewayPort = options.gatewayPort ?? 30000 + Math.floor(Math.random() * 1000)
  const runtimeConfig = options.webBundle
    ? {
        ...options.webBundle.runtimeConfig,
        controlBaseUrl: '',
        gatewayUrl: `ws://127.0.0.1:${gatewayPort}`,
      }
    : null
  const controlApp = buildControlServer({
    db: controlDb,
    crypto: loadEnvelopeCryptoFromEnv({ KEY_ENCRYPTION_KEY: kek }),
    ...(options.webBundle && runtimeConfig
      ? {
          webBundle: {
            distPath: options.webBundle.distPath,
            runtimeConfig,
            ...(options.webBundle.updateChannelRootPath
              ? { updateChannelRootPath: options.webBundle.updateChannelRootPath }
              : {}),
          },
        }
      : {}),
  })
  // Temp verification hook — log every HTTP request the control plane receives. Enables
  // observation of what the Tauri webview actually calls. Toggle via AGENT_ZERO_DEV_TRACE.
  if (process.env['AGENT_ZERO_DEV_TRACE']) {
    controlApp.addHook('onRequest', async (req) => {
      console.log(
        `[trace] ${req.method} ${req.url} ua=${(req.headers['user-agent'] ?? '').slice(0, 40)}`,
      )
    })
  }
  await controlApp.listen({ port: options.controlPort ?? 0, host: '127.0.0.1' })
  const controlPort = (controlApp.server.address() as { port: number }).port
  const controlUrl = `http://127.0.0.1:${controlPort}`

  const listener = options.kbIngest
    ? await startKbIngestListener(databaseUrl, controlDb as unknown as Kysely<KbDatabase>, {
        blobRootDir: options.kbIngest.blobRootDir,
        providerId: options.kbIngest.providerId,
        resolveApiKey: options.kbIngest.resolveApiKey,
      })
    : null

  const gateway = await startGatewayRuntime(databaseUrl, kek, {
    gatewayPort,
    ...(options.permissionTimeoutMs !== undefined
      ? { permissionTimeoutMs: options.permissionTimeoutMs }
      : {}),
  })

  const webServer = await startWebStaticServer(controlUrl)

  return {
    controlUrl,
    gatewayUrl: gateway.gatewayUrl,
    webUrl: webServer.url,
    close: async () => {
      await webServer.close()
      await gateway.close()
      await listener?.stop()
      await controlApp.close()
      await controlDb.destroy()
    },
  }
}

async function startKbIngestListener(
  databaseUrl: string,
  db: Kysely<KbDatabase>,
  options: {
    blobRootDir: string
    providerId: string
    resolveApiKey: () => Promise<string>
  },
): Promise<KbIngestListener> {
  const worker = new IngestionWorker({
    db,
    blobs: new BlobStore({ db, rootDir: options.blobRootDir }),
    provider_id: options.providerId as Id<'provider'>,
    resolveApiKey: options.resolveApiKey,
  })
  const listener = new KbIngestListener({
    databaseUrl,
    worker,
    onError: (error, runId) => {
      console.error(`[kb] ingestion run ${runId} failed`, error)
    },
  })
  await listener.start()
  return listener
}

import { createTerminus } from '@godaddy/terminus'
import * as http from 'http'
import * as https from 'https'
import winston from 'winston'

import { CreateApplication } from './Application'
import {
  App,
  HTTP_PORT,
  HTTPS_PORT,
  S3,
  SSL_CERT,
  SSL_KEY,
  SSL_KEY_PASSPHRASE,
} from './config'

const ENABLE_SSL = Boolean(SSL_CERT && SSL_KEY)
const PORT = ENABLE_SSL ? HTTPS_PORT : HTTP_PORT

const logger = winston.createLogger({
  format: winston.format.json(),
  transports: [new winston.transports.Console()],
})

const app = CreateApplication({
  app: App,
  s3: S3,
})

const server = ENABLE_SSL
  ? https.createServer(
      {
        key: SSL_KEY,
        cert: SSL_CERT,
        passphrase: SSL_KEY_PASSPHRASE || undefined,
      },
      app,
    )
  : http.createServer(app)

createTerminus(server, {
  timeout: 1000,
  signals: ['SIGINT', 'SIGTERM', 'SIGHUP'],
  onSignal: async () => {
    logger.log({
      level: 'info',
      message: 'Server received shutdown signal',
    })
  },
  onShutdown: async () => {
    logger.log({
      level: 'info',
      message: 'Server is shutting down',
    })
  },
  healthChecks: {
    '/healthz': async () => undefined,
  },
})

server.on('listening', () => {
  logger.log({
    level: 'info',
    message: `Server is listening on port ${PORT}`,
    port: PORT,
    protocol: ENABLE_SSL ? 'https' : 'http',
    s3Bucket: S3.BUCKET,
    s3Endpoint: S3.ENDPOINT,
    s3SignatureVersion: S3.SIGNATURE_VERSION,
    s3Folder: S3.FOLDER,
    s3ImmutableTree: S3.IMMUTABLE_TREE,
    s3CacheResponses: S3.CACHE_RESPONSES,
    s3CacheTTL: S3.CACHE_TTL,
  })
})

server.listen(PORT)

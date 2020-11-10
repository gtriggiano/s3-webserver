import { App, HTTP_PORT, HTTPS_PORT, S3 } from './config'
import { Server } from './Server'

const PORT = App.enableSSL ? HTTPS_PORT : HTTP_PORT

const server = Server({
  app: App,
  s3: S3,
})

const SIGNALS = {
  SIGHUP: 1,
  SIGINT: 2,
  SIGTERM: 15,
}

type Signal = keyof typeof SIGNALS

const httpServer = server.listen(PORT)

httpServer.on('listening', () => {
  console.log(
    `Serving ${S3.BUCKET}/${S3.FOLDER} from ${S3.ENDPOINT} on port ${PORT}`,
  )

  const shutdown = (SIGNAL: Signal) => {
    console.log('shutdown!')
    process.removeAllListeners()
    httpServer.close(() => {
      const value = SIGNALS[SIGNAL]
      console.log(`server stopped by ${SIGNAL} with value ${value}`)
      process.exit(128 + value)
    })
  }

  Object.keys(SIGNALS).forEach((SIGNAL) => {
    process.on(SIGNAL, () => {
      console.log(`process received a ${SIGNAL} signal`)
      shutdown(SIGNAL as Signal)
    })
  })
})

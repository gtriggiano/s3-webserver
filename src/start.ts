import { HTTP_PORT, HTTPS_PORT, S3, TemplatedApp } from './config'
import { closeSocket, Server } from './Server'

const PORT = TemplatedApp.enableSSL ? HTTPS_PORT : HTTP_PORT

const server = Server({
  app: TemplatedApp,
  s3: S3,
})

const SIGNALS = {
  SIGHUP: 1,
  SIGINT: 2,
  SIGTERM: 15,
}

type Signal = keyof typeof SIGNALS

server.listen('0.0.0.0', PORT, (socket) => {
  console.log(
    `Serving ${S3.BUCKET}/${S3.FOLDER} from ${S3.ENDPOINT} on port ${PORT}`,
  )

  const shutdown = (SIGNAL: Signal) => {
    console.log('shutdown!')
    process.removeAllListeners()
    closeSocket(socket)

    const value = SIGNALS[SIGNAL]
    console.log(`server stopped by ${SIGNAL} with value ${value}`)
    process.exit(128 + value)
  }

  Object.keys(SIGNALS).forEach((SIGNAL) => {
    process.on(SIGNAL, () => {
      console.log(`process received a ${SIGNAL} signal`)
      shutdown(SIGNAL as Signal)
    })
  })
})

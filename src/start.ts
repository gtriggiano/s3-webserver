import { HTTP_PORT, HTTPS_PORT, S3, TemplatedApp } from './config'
import { Server } from './Server'

const PORT = TemplatedApp.enableSSL ? HTTPS_PORT : HTTP_PORT

const server = Server({
  app: TemplatedApp,
  s3: S3,
})

server.listen(PORT, () => {
  console.log(
    `Serving ${S3.BUCKET}/${S3.FOLDER} from ${S3.ENDPOINT} on port ${PORT}`,
  )
})

import { AWSError, S3 } from 'aws-sdk'
import { compact } from 'lodash'
import { HttpResponse, RecognizedString, TemplatedApp } from 'uWebSockets.js'

import {
  DOCUMENT_404_KEY,
  S3 as S3Config,
  TemplatedApp as TemplatedAppConfig,
} from '../config'
import { App } from './App'

export const Server = ({
  app: { enableSSL, SSL_CERT, SSL_KEY, SSL_KEY_PASSPHRASE },
  s3: { ACCESS_KEY_ID, SECRET_ACCESS_KEY, ENDPOINT, BUCKET, FOLDER },
}: Config): TemplatedApp => {
  const app = App({
    enableSSL,
    SSL_CERT,
    SSL_KEY,
    SSL_KEY_PASSPHRASE,
  })

  const s3 = new S3({
    credentials: {
      accessKeyId: ACCESS_KEY_ID,
      secretAccessKey: SECRET_ACCESS_KEY,
    },
    s3ForcePathStyle: true,
    signatureVersion: 'v4',
    endpoint: ENDPOINT,
  })

  const loadFolder = (path: string) => {
    return s3
      .listObjects({
        Bucket: BUCKET,
        Delimiter: '/',
        Prefix: path,
      })
      .promise()
  }

  const serveKey = async (key: string, res: HttpResponse) => {
    try {
      const request = s3
        .getObject({
          Bucket: BUCKET,
          Key: key,
        })
        .promise()

      const result = await request

      res.writeStatus(`200 OK`)

      const {
        Body,
        ContentEncoding,
        ContentLength,
        ContentType,
        ETag,
        Expiration,
        LastModified,
      } = result.$response.data as S3.GetObjectOutput

      const headers: Record<string, string | undefined> = {
        'Content-Length': ContentLength ? String(ContentLength) : undefined,
        'Last-Modified': LastModified ? LastModified.toUTCString() : undefined,
        Expiration: Expiration,
        Etag: ETag,
        'Content-Encoding': ContentEncoding,
        'Content-Type': ContentType,
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET',
        'Access-Control-Allow-Headers': 'Host,Content-*',
        'Access-Control-Max-Age': '3000',
      }

      Object.keys(headers).forEach((headerName) => {
        const value = headers[headerName]
        if (value) {
          res.writeHeader(headerName, value)
        }
      })
      if (Body) {
        res.write(Body as RecognizedString)
      }
      res.end()
    } catch (error) {
      const { statusCode } = error as AWSError
      if (statusCode === 404 && DOCUMENT_404_KEY && DOCUMENT_404_KEY !== key) {
        serveKey(DOCUMENT_404_KEY, res)
        return
      }

      res.writeStatus(String(error.statusCode))
      res.end()
    }
  }

  const serveKeysList = async (
    folderKey: string,
    keys: string[],
    res: HttpResponse,
  ) => {
    res.write(
      `<html>
<head>
    <title>${folderKey}</title>
</head>
<body>
    <ul>
        ${keys.reduce<string>(
          (tags, key) =>
            `${tags}<li><a href="/${key}">${key.replace(
              folderKey,
              '',
            )}</a></li>`,
          '',
        )}
    </ul>
</body>
</html>
      `,
    )
    res.end()
  }

  app.get('/healthz', (res) => {
    res.writeStatus('200')
    res.end()
  })

  app.get('/*', async (res, req) => {
    res.onAborted(() => undefined)

    const path = req.getUrl().substr(1)
    const prefix = FOLDER && `${FOLDER}/`
    const pathInBucket = unescape(`${prefix}${path}`)

    if (pathInBucket === prefix || pathInBucket.substr(-1) === '/') {
      try {
        const response = await loadFolder(pathInBucket)
        const { Contents } = response

        if (Contents?.length) {
          const indexFile = Contents.find(
            ({ Key }) => Key?.indexOf('index.html') !== -1,
          )

          if (indexFile && indexFile.Key) {
            serveKey(indexFile.Key, res)
          } else {
            serveKeysList(
              pathInBucket.replace(prefix, ''),
              compact((Contents || []).map(({ Key }) => Key)).map((key) =>
                escape(key.replace(prefix, '')),
              ),
              res,
            )
          }
        } else {
          serveKeysList(pathInBucket.replace(prefix, ''), [], res)
        }
        return
      } catch (error) {
        const { statusCode, message } = error as AWSError
        res.writeStatus(String(statusCode))
        res.write(message)
        res.end()
        return
      }
    }

    serveKey(pathInBucket, res)
  })

  return app
}

interface Config {
  app: TemplatedAppConfig
  s3: S3Config
}

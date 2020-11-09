import { AWSError, S3 } from 'aws-sdk'
import express, { Application, Response } from 'express'
import { compact } from 'lodash'

import {
  DOCUMENT_404_KEY,
  S3 as S3Config,
  TemplatedApp as TemplatedAppConfig,
} from '../config'

export const Server = ({
  s3: { ACCESS_KEY_ID, SECRET_ACCESS_KEY, ENDPOINT, BUCKET, FOLDER },
}: Config): Application => {
  const app = express()

  app.set('trust proxy', true)

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

  const serveKey = async (key: string, res: Response) => {
    try {
      const request = s3
        .getObject({
          Bucket: BUCKET,
          Key: key,
        })
        .promise()

      const result = await request

      res.status(200)

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
        'Access-Control-Allow-Headers': '*',
        'Access-Control-Max-Age': '3000',
      }

      console.log(headers)

      Object.keys(headers).forEach((headerName) => {
        const value = headers[headerName]
        if (value) {
          res.setHeader(headerName, value)
        }
      })
      if (Body) {
        console.log(Body)
        res.send(Body)
      } else {
        res.end()
      }
    } catch (error) {
      console.log(error)
      const { statusCode } = error as AWSError
      if (statusCode === 404 && DOCUMENT_404_KEY && DOCUMENT_404_KEY !== key) {
        serveKey(DOCUMENT_404_KEY, res)
        return
      }

      res.sendStatus(error.statusCode)
    }
  }

  const serveKeysList = async (
    folderKey: string,
    keys: string[],
    res: Response,
  ) => {
    res.send(
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
  }

  app.get('/healthz', (req, res) => {
    console.log('Health check')
    res.sendStatus(200)
  })

  app.get('/*', async (req, res) => {
    // res.onAborted(() => undefined)

    const path = req.url.substr(1)
    const prefix = FOLDER && `${FOLDER}/`
    const pathInBucket = unescape(`${prefix}${path}`)

    console.log(`Request: ${pathInBucket}`)

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
        statusCode && res.status(statusCode)
        res.send(message)
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

import { S3 } from 'aws-sdk'
import D from 'debug'
import express, { Application, Request, Response } from 'express'
import expressWinston from 'express-winston'
import NodeCache from 'node-cache'
import winston from 'winston'

import { App as AppConfig, S3 as S3Config } from '../config'
import {
  GetBucketKeyResult,
  ListBucketPathResult,
  S3Utils,
} from '../utils/S3Utils'

const httpDebug = D('http')
const s3QueryDebug = D('s3Query')

const MINIMUM_CACHE_TTL = 60

const LOG_MIDDLEWARE = expressWinston.logger({
  transports: [new winston.transports.Console()],
  format: winston.format.combine(
    winston.format.colorize(),
    winston.format.json(),
  ),
})

export const Server = ({
  app: {
    ENABLE_DIRECTORY_LISTING,
    DEFAULT_403_FILE,
    DEFAULT_404_FILE,
    FOLDER_INDEX_FILE_NAME,
  },
  s3: {
    ACCESS_KEY_ID,
    SECRET_ACCESS_KEY,
    ENDPOINT,
    BUCKET,
    FOLDER,
    FORCE_PATH_STYLE,
    IMMUTABLE_TREE,
    CACHE_RESPONSES,
  },
}: Config): Application => {
  const PREFIX = FOLDER && `${FOLDER}/`
  const USE_CACHE = IMMUTABLE_TREE || CACHE_RESPONSES
  const FILE_403_KEY = DEFAULT_403_FILE ? `${PREFIX}${DEFAULT_403_FILE}` : null
  const FILE_404_KEY = DEFAULT_404_FILE ? `${PREFIX}${DEFAULT_404_FILE}` : null

  const s3 = new S3({
    credentials: {
      accessKeyId: ACCESS_KEY_ID,
      secretAccessKey: SECRET_ACCESS_KEY,
    },
    s3ForcePathStyle: FORCE_PATH_STYLE,
    signatureVersion: 'v4',
    endpoint: ENDPOINT,
  })

  const renderFolderListIndexHtml = (
    folderPath: string,
    paths: string[],
  ): string => `<html>
<head>
    <title>${folderPath}</title>
</head>
<body>
    <ul>
        ${paths.reduce<string>(
          (tags, path) =>
            `${tags}<li><a href="/${path}">${path.replace(
              folderPath,
              '',
            )}</a></li>`,
          '',
        )}
    </ul>
</body>
</html>`

  const listBucketPathResults = new NodeCache({
    checkperiod: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? MINIMUM_CACHE_TTL : 0,
    useClones: false,
    stdTTL: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? MINIMUM_CACHE_TTL : 0,
  })
  const getBucketKeyReults = new NodeCache({
    checkperiod: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? MINIMUM_CACHE_TTL : 0,
    useClones: false,
    stdTTL: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? MINIMUM_CACHE_TTL : 0,
  })

  const s3Utils = S3Utils(s3)

  const isFolderIndexFileKey = (fileKey: string): boolean =>
    fileKey.endsWith(FOLDER_INDEX_FILE_NAME)

  const listBucketPath = async (
    path: string,
  ): Promise<ListBucketPathResult> => {
    const cachedResult: ListBucketPathResult | null =
      (USE_CACHE && listBucketPathResults.get(path)) || null
    const result =
      cachedResult || (await s3Utils.listBucketPath({ bucket: BUCKET, path }))
    if (USE_CACHE) {
      listBucketPathResults.set(path, result)
    }

    if (cachedResult === null && s3QueryDebug.enabled) {
      s3QueryDebug(`List Bucket Path: ${path}`)
      s3QueryDebug(JSON.stringify(result, null, 2))
      s3QueryDebug('')
    }

    return result
  }

  const getBucketKey = async (key: string): Promise<GetBucketKeyResult> => {
    const cachedResult: GetBucketKeyResult | null =
      (USE_CACHE && getBucketKeyReults.get(key)) || null
    const result =
      cachedResult || (await s3Utils.getBucketKey({ bucket: BUCKET, key }))
    if (USE_CACHE) {
      getBucketKeyReults.set(key, result)
    }

    if (cachedResult === null && s3QueryDebug.enabled) {
      const { body, ...rest } = result
      s3QueryDebug(`Get Bucket Key: ${key}`)
      s3QueryDebug(JSON.stringify(rest, null, 2))
      s3QueryDebug('')
    }

    return result
  }

  const serveKeyAsFile = async (
    key: string,
    req: Request,
    res: Response,
  ): Promise<void> => {
    const keyResult = await getBucketKey(key)

    if (keyResult.statusCode === 403) {
      await serveKeyAsDirectory(key, req, res)
      return
    }

    if (keyResult.statusCode === 404) {
      await serveKeyAsDirectory(key, req, res)
      return
    }

    res.status(keyResult.statusCode)
    Object.keys(keyResult.headers).forEach((key) => {
      const value = keyResult.headers[key]
      if (value) {
        res.setHeader(key, value)
      }
    })
    if (keyResult.body) {
      res.send(keyResult.body)
    } else if (keyResult.message) {
      res.send(keyResult.message)
    } else {
      res.end()
    }
  }

  const serveKeyAsDirectory = async (
    key: string,
    req: Request,
    res: Response,
  ): Promise<void> => {
    try {
      const { files, folders, error } = await listBucketPath(key)
      const indexFileKey = files.find(isFolderIndexFileKey)

      if (error) {
        const { statusCode } = error
        if (statusCode === 403 && FILE_403_KEY) {
          serveKeyAsFile(FILE_403_KEY, req, res)
          return
        }
        res.status(statusCode || 500)
        res.send(error.message)
        return
      }

      if (indexFileKey) {
        return serveKeyAsFile(indexFileKey, req, res)
      }

      if (files.length + folders.length === 0) {
        res.sendStatus(404)
        return
      }

      if (ENABLE_DIRECTORY_LISTING) {
        if (req.is('json')) {
          res.send({
            files,
            folders,
          })
        } else {
          res.send(
            renderFolderListIndexHtml(
              key.replace(PREFIX, ''),
              files.concat(folders).map((path) => path.replace(PREFIX, '')),
            ),
          )
        }
        return
      }

      res.sendStatus(403)
      return
    } catch (error) {
      res.status(500)
      res.send(error.message)
      return
    }
  }

  const app = express()
  app.set('trust proxy', true)
  app.disable('x-powered-by')

  app.get('/healthz', (req, res) => {
    res.sendStatus(200)
  })

  if (process.env.NODE_ENV === 'production') {
    app.use(LOG_MIDDLEWARE)
  }

  app.get('/*', async (req, res) => {
    const path = req.url.substr(1) // We remove the initial /
    const pathInBucket = unescape(`${PREFIX}${path}`)

    if (path === '' || pathInBucket.endsWith('/')) {
      await serveKeyAsDirectory(pathInBucket, req, res)
      return
    }

    await serveKeyAsFile(pathInBucket, req, res)
  })

  httpDebug(`Instantiated Express App`)
  httpDebug(`ENDPOINT: ${ENDPOINT}`)
  httpDebug(`BUCKET: ${BUCKET}`)
  httpDebug(`FOLDER: ${FOLDER}`)
  httpDebug(`FORCE_PATH_STYLE: ${FORCE_PATH_STYLE}`)
  httpDebug(`FILE_403_KEY: ${FILE_403_KEY}`)
  httpDebug(`FILE_404_KEY: ${FILE_404_KEY}`)

  return app
}

interface Config {
  app: AppConfig
  s3: S3Config
}

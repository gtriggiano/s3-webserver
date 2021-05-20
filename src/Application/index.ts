import { S3 } from 'aws-sdk'
import compression from 'compression'
import cors from 'cors'
import express, { Application, Request, Response } from 'express'
import expressWinston from 'express-winston'
import helmet from 'helmet'
import { some } from 'lodash'
import NodeCache from 'node-cache'
import winston from 'winston'

import { App as AppConfig, S3 as S3Config } from '../config'
import { getFolderHTML } from './utils/getFolderHTML'
import { GetBucketKeyResult, makeGetBucketKey } from './utils/makeGetBucketKey'
import {
  ListBucketPathResult,
  makeListBucketPath,
} from './utils/makeListBucketPath'
import { removeLeadingCharRecursively } from './utils/removeLeadingCharRecursively'
import { removeTrailingCharRecursively } from './utils/removeTrailingCharRecursively'

const GATSBY_REDIRECT_REGEX = /^<script>window\.location\.href="(.+)"<\/script>$/

export const CreateApplication = ({
  app: {
    DEFAULT_403_FILE,
    DEFAULT_404_FILE,
    ENABLE_DIRECTORY_LISTING,
    FOLDER_INDEX_FILE_NAME,
    TRUST_PROXY,
    CACHE_CONTROL_MAX_AGE,
    CACHE_CONTROL_REGEXP_LIST,
    LOG_HTTP_CALLS,
    HANDLE_GATSBY_REDIRECTS,
  },
  s3: {
    ACCESS_KEY_ID,
    BUCKET,
    CACHE_RESPONSES,
    CACHE_TTL,
    ENDPOINT,
    FOLDER,
    FORCE_PATH_STYLE,
    IMMUTABLE_TREE,
    LOG_LEVEL,
    SECRET_ACCESS_KEY,
    SIGNATURE_VERSION,
  },
}: ServerConfiguration): Application => {
  const SAFE_FOLDER = removeTrailingCharRecursively(
    removeLeadingCharRecursively(FOLDER, '/'),
    '/',
  )
  const LOOKUP_PREFIX = SAFE_FOLDER && `${SAFE_FOLDER}/`
  const USE_CACHE = IMMUTABLE_TREE || CACHE_RESPONSES
  const FILE_403_KEY = DEFAULT_403_FILE
    ? `${LOOKUP_PREFIX}${DEFAULT_403_FILE}`
    : null
  const FILE_404_KEY = DEFAULT_404_FILE
    ? `${LOOKUP_PREFIX}${DEFAULT_404_FILE}`
    : null

  const isFileKeyAFolderIndex = (fileKey: string): boolean =>
    fileKey === FOLDER_INDEX_FILE_NAME ||
    fileKey.endsWith(`/${FOLDER_INDEX_FILE_NAME}`)

  const getBucketKeyCachedReults = new NodeCache({
    checkperiod: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? CACHE_TTL : 0,
    useClones: false,
    stdTTL: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? CACHE_TTL : 0,
  })

  const listBucketPathCachedResults = new NodeCache({
    checkperiod: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? CACHE_TTL : 0,
    useClones: false,
    stdTTL: IMMUTABLE_TREE ? 0 : CACHE_RESPONSES ? CACHE_TTL : 0,
  })

  const s3Client = new S3({
    credentials: {
      accessKeyId: ACCESS_KEY_ID,
      secretAccessKey: SECRET_ACCESS_KEY,
    },
    s3ForcePathStyle: FORCE_PATH_STYLE,
    signatureVersion: SIGNATURE_VERSION,
    endpoint: ENDPOINT,
  })
  const getBucketKey = makeGetBucketKey(
    s3Client,
    LOG_LEVEL === 'none' ? undefined : LOG_LEVEL,
  )
  const cachedGetBucketKey = async (
    key: string,
  ): Promise<GetBucketKeyResult> => {
    const cachedResult: GetBucketKeyResult | null =
      (USE_CACHE && getBucketKeyCachedReults.get(key)) || null

    const result = cachedResult || (await getBucketKey({ bucket: BUCKET, key }))

    if (USE_CACHE) {
      getBucketKeyCachedReults.set(key, result)
    }

    return result
  }

  const listBucketPath = makeListBucketPath(
    s3Client,
    LOG_LEVEL === 'none' ? undefined : LOG_LEVEL,
  )
  const cachedListBucketPath = async (
    path: string,
  ): Promise<ListBucketPathResult> => {
    const cachedResult: ListBucketPathResult | null =
      (USE_CACHE && listBucketPathCachedResults.get(path)) || null

    const result =
      cachedResult || (await listBucketPath({ bucket: BUCKET, path }))

    if (USE_CACHE) {
      listBucketPathCachedResults.set(path, result)
    }

    return result
  }

  const serveKeyAsFile = async (
    key: string,
    req: Request,
    res: Response,
    serveDirectoryFallback: boolean,
  ): Promise<void> => {
    const getResult = await cachedGetBucketKey(key)

    if (getResult.error) {
      if (serveDirectoryFallback) {
        return serveKeyAsDirectory(key, req, res)
      } else {
        if (
          getResult.error.statusCode === 403 &&
          FILE_403_KEY &&
          FILE_403_KEY !== key
        ) {
          return serveKeyAsFile(FILE_403_KEY, req, res, false)
        }

        if (
          getResult.error.statusCode === 404 &&
          FILE_404_KEY &&
          FILE_404_KEY !== key
        ) {
          return serveKeyAsFile(FILE_404_KEY, req, res, false)
        }

        res.status(getResult.error.statusCode || 500)
        res.send(getResult.error.message)
      }
    } else {
      const { headers, body } = getResult

      if (body && isFileKeyAFolderIndex(key) && HANDLE_GATSBY_REDIRECTS) {
        const [, pathForRedirect] = [
          ...(body.toString().match(GATSBY_REDIRECT_REGEX) || []),
        ]

        if (pathForRedirect) {
          return res.redirect(301, pathForRedirect)
        }
      }

      Object.keys(headers).forEach((key) => {
        const value = headers[key]
        if (value) {
          res.setHeader(key, value)
        }
      })
      if (body) {
        const shouldAddCacheControl = some(
          CACHE_CONTROL_REGEXP_LIST,
          (regExp) => regExp.test(key),
        )

        if (shouldAddCacheControl) {
          res.setHeader('Cache-Control', `max-age=${CACHE_CONTROL_MAX_AGE}`)
        }

        res.send(body)
      } else {
        res.end()
      }
    }
  }

  const serveKeyAsDirectory = async (
    key: string,
    req: Request,
    res: Response,
  ): Promise<void> => {
    const { files, folders, error } = await cachedListBucketPath(key)
    const indexFileKey = files.find(isFileKeyAFolderIndex)

    if (error) {
      if (error.statusCode === 403 && FILE_403_KEY) {
        return serveKeyAsFile(FILE_403_KEY, req, res, false)
      }
      res.status(error.statusCode || 500)
      res.send(error.message)
      return
    }

    if (indexFileKey) {
      return serveKeyAsFile(indexFileKey, req, res, false)
    }

    if (files.length + folders.length === 0) {
      if (FILE_404_KEY) {
        return serveKeyAsFile(FILE_404_KEY, req, res, false)
      } else {
        res.sendStatus(404)
        return
      }
    }

    if (ENABLE_DIRECTORY_LISTING) {
      if (req.xhr) {
        res.send({
          files: files.map((path) => `/${path.replace(LOOKUP_PREFIX, '')}`),
          folders: folders.map((path) => `/${path.replace(LOOKUP_PREFIX, '')}`),
        })
      } else {
        res.send(
          getFolderHTML(
            `/${key.replace(LOOKUP_PREFIX, '')}`,
            folders
              .concat(files)
              .map((path) => `/${path.replace(LOOKUP_PREFIX, '')}`),
          ),
        )
      }
      return
    }

    res.sendStatus(403)
  }

  const app = express()
  app.set('trust proxy', TRUST_PROXY)
  app.use(compression())
  app.use(cors())

  if (LOG_HTTP_CALLS) {
    app.use(
      expressWinston.logger({
        transports: [new winston.transports.Console()],
        format: winston.format.json(),
      }),
    )
  }

  app.use(helmet.hidePoweredBy())
  app.use(helmet.noSniff())
  app.use(helmet.dnsPrefetchControl())
  app.use(helmet.ieNoOpen())
  app.use(helmet.frameguard())
  app.use(helmet.xssFilter())

  const hstsMiddleware = helmet.hsts()
  app.use((req, res, next) => {
    if (req.protocol === 'https') {
      hstsMiddleware(req, res, next)
    } else {
      next()
    }
  })

  const cspMiddleware = helmet.contentSecurityPolicy()
  app.use((req, res, next) => {
    if (AppConfig.ENABLE_CSP && req.protocol === 'https') {
      cspMiddleware(req, res, next)
    } else {
      next()
    }
  })

  app.get('/*', async (req, res) => {
    const path = req.path.substr(1) // We remove the initial /
    const pathInBucket = unescape(`${LOOKUP_PREFIX}${path}`)

    if (path === '' || pathInBucket.endsWith('/')) {
      return serveKeyAsDirectory(pathInBucket, req, res)
    }

    return serveKeyAsFile(pathInBucket, req, res, true)
  })

  if (process.env.NODE_ENV === 'production') {
    app.use(
      expressWinston.errorLogger({
        transports: [new winston.transports.Console()],
        format: winston.format.combine(
          winston.format.colorize(),
          winston.format.json(),
        ),
      }),
    )
  }

  return app
}

interface ServerConfiguration {
  app: AppConfig
  s3: S3Config
}

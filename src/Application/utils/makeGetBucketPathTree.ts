import { AWSError } from 'aws-sdk'
import now from 'performance-now'
import winston from 'winston'

import { ListBucketPath } from './makeListBucketPath'
import { removeTrailingCharRecursively } from './removeTrailingCharRecursively'

export type GetBucketPathTree = (
  input: GetBucketPathTreeInput,
) => Promise<GetBucketPathTreeResult>

export interface GetBucketPathTreeInput {
  bucket: string
  path: string
  depth: number
}

export interface GetBucketPathTreeResult {
  tree: Tree
  error?: AWSError
}

type Tree = { [key: string]: string | Tree }

export const makeGetBucketPathTree = (
  listBucketPath: ListBucketPath,
  logLevel?: 'info' | 'error',
): GetBucketPathTree => {
  const logger =
    logLevel &&
    winston.createLogger({
      format: winston.format.json(),
      defaultMeta: {
        method: 'getBucketPathTree',
      },
      level: logLevel,
      transports: [new winston.transports.Console()],
    })

  const getBucketPathTree: GetBucketPathTree = async ({
    bucket,
    depth,
    path,
  }) => {
    const start = now()

    const tree: Tree = {}

    const safeDepth = toSafeDepth(depth)

    const listResult = await listBucketPath({ bucket, path })

    if (listResult.error) {
      logger &&
        logger.log({
          level: 'error',
          message: listResult.error.message,
          path,
          duration: Math.round(now() - start),
        })

      return {
        tree,
        error: listResult.error,
      }
    }

    listResult.files.forEach((file) => {
      tree[getLastPathSegment(file)] = file
    })

    await Promise.all(
      listResult.folders.map(async (folder) => {
        if (depth > 0) {
          const { tree } = await getBucketPathTree({
            bucket,
            depth: safeDepth - 1,
            path: folder,
          })
          tree[getLastPathSegment(folder)] = tree
        } else {
          tree[getLastPathSegment(folder)] = {}
        }
      }),
    )

    logger &&
      logger.log({
        level: 'info',
        message: 'Success',
        path,
        duration: Math.round(now() - start),
      })

    return { tree }
  }

  return getBucketPathTree
}

/**
 * Returns a number where 0 <= n <= Number.MAX_SAFE_INTEGER
 */
const toSafeDepth = (u: number): number => {
  let n = Number(u)
  n = Number.isNaN(n) ? Number.MAX_SAFE_INTEGER : n
  return n < 0 ? Number.MAX_SAFE_INTEGER : n
}

const getLastPathSegment = (path: string): string =>
  removeTrailingCharRecursively(path, '/').split('/').reverse()[0]

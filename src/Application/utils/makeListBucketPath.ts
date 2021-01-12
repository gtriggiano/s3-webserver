import { AWSError, S3 } from 'aws-sdk'
import { compact } from 'lodash'
import now from 'performance-now'
import winston from 'winston'

import { removeLeadingCharRecursively } from './removeLeadingCharRecursively'
import { removeTrailingCharRecursively } from './removeTrailingCharRecursively'

export type ListBucketPath = (
  input: ListBucketPathInput,
) => Promise<ListBucketPathResult>

export interface ListBucketPathInput {
  bucket: string
  path: string
}

export interface ListBucketPathResult {
  files: string[]
  folders: string[]
  error?: AWSError
}

export const makeListBucketPath = (
  s3: S3,
  logLevel?: 'info' | 'error',
): ListBucketPath => {
  const logger =
    logLevel &&
    winston.createLogger({
      format: winston.format.json(),
      defaultMeta: {
        method: 'listBucketPath',
      },
      level: logLevel,
      transports: [new winston.transports.Console()],
    })

  return async ({ bucket, path }) => {
    const start = now()

    const pathWithOneTrailingSlash =
      removeTrailingCharRecursively(path, '/') + '/'
    const prefix = removeLeadingCharRecursively(pathWithOneTrailingSlash, '/')

    const result: ListBucketPathResult = { files: [], folders: [] }

    const populateResult = async (
      awsData: S3.ListObjectsV2Output,
    ): Promise<void> => {
      result.files = result.files.concat(
        compact((awsData.Contents || []).map((i) => i.Key)),
      )
      result.folders = result.folders.concat(
        compact((awsData.CommonPrefixes || []).map((i) => i.Prefix)),
      )

      if (awsData.IsTruncated) {
        try {
          const result = await s3
            .listObjectsV2({
              Bucket: bucket,
              MaxKeys: 1000000, // Maximum allowed by Wasabi S3 API
              Delimiter: '/',
              Prefix: prefix,
              ContinuationToken: awsData.NextContinuationToken,
            })
            .promise()

          if (result.$response.data) {
            await populateResult(result.$response.data)
          }
        } catch (error) {
          result.error = error
        }
      }
    }

    try {
      const result = await s3
        .listObjectsV2({
          Bucket: bucket,
          MaxKeys: 1000000, // Maximum allowed by Wasabi S3 API
          Delimiter: '/',
          Prefix: prefix,
          StartAfter: prefix, // removes the folder name from listing
        })
        .promise()

      if (result.$response.data) {
        await populateResult(result.$response.data)
      }
    } catch (error) {
      result.error = error
    }

    result.files.sort()
    result.folders.sort()

    logger &&
      logger.log({
        level: result.error ? 'error' : 'info',
        message: result.error ? result.error.message : 'Success',
        path,
        duration: Math.round(now() - start),
        foundFiles: result.files.length,
        foundFolders: result.folders.length,
      })

    return result
  }
}

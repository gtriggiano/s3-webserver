import { AWSError, S3 } from 'aws-sdk'
import now from 'performance-now'
import winston from 'winston'

export type GetBucketKey = (
  input: GetBucketKeyInput,
) => Promise<GetBucketKeyResult>

export interface GetBucketKeyInput {
  bucket: string
  key: string
}

export type GetBucketKeyResult =
  | {
      error: undefined
      body?: S3.Body
      headers: Record<string, string | undefined>
    }
  | {
      error: AWSError
    }

export const makeGetBucketKey = (
  s3: S3,
  logLevel?: 'info' | 'error',
): GetBucketKey => {
  const logger =
    logLevel &&
    winston.createLogger({
      format: winston.format.json(),
      defaultMeta: {
        method: 'getBucketKey',
      },
      level: logLevel,
      transports: [new winston.transports.Console()],
    })

  return async ({ bucket, key }) => {
    const start = now()

    try {
      const {
        Body,
        ContentEncoding,
        ContentLength,
        ContentType,
        ETag,
        Expiration,
        LastModified,
      } = await s3
        .getObject({
          Bucket: bucket,
          Key: key,
        })
        .promise()

      logger &&
        logger.log({
          level: 'info',
          message: `Success`,
          key,
          duration: Math.round(now() - start),
        })

      return {
        error: undefined,
        body: Body,
        headers: {
          'Content-Length': ContentLength ? String(ContentLength) : undefined,
          'Last-Modified': LastModified
            ? LastModified.toUTCString()
            : undefined,
          Expiration: Expiration,
          Etag: ETag,
          'Content-Encoding': ContentEncoding,
          'Content-Type': ContentType,
        },
      }
    } catch (error) {
      logger &&
        logger.log({
          level: 'error',
          message: error.message,
          key,
          duration: Math.round(now() - start),
        })

      return {
        error,
      }
    }
  }
}

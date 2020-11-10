import { AWSError, S3 } from 'aws-sdk'

interface GetBucketKeyInput {
  bucket: string
  key: string
}

export interface GetBucketKeyResult {
  statusCode: number
  headers: Record<string, string | undefined>
  message?: string
  body?: S3.Body
}

export type GetBucketKey = (
  input: GetBucketKeyInput,
) => Promise<GetBucketKeyResult>
export const makeGetBucketKey = (s3: S3): GetBucketKey => {
  return async ({ bucket, key }) => {
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

      return {
        statusCode: 200,
        headers: {
          'Content-Length': ContentLength ? String(ContentLength) : undefined,
          'Last-Modified': LastModified
            ? LastModified.toUTCString()
            : undefined,
          Expiration: Expiration,
          Etag: ETag,
          'Content-Encoding': ContentEncoding,
          'Content-Type': ContentType,
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'GET',
          'Access-Control-Allow-Headers': '*',
          'Access-Control-Max-Age': '3000',
        },
        body: Body,
      }
    } catch (error) {
      const { statusCode, message } = error as AWSError
      return {
        statusCode: statusCode || 500,
        headers: {},
        message,
      }
    }
  }
}

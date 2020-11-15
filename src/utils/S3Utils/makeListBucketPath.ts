import { AWSError, S3 } from 'aws-sdk'
import { compact } from 'lodash'

interface ListBucketPathInput {
  bucket: string
  path: string
}

export interface ListBucketPathResult {
  files: string[]
  folders: string[]
  error?: AWSError
}

export type ListBucketPath = (
  input: ListBucketPathInput,
) => Promise<ListBucketPathResult>

const trimStart = (s: string, ch: string): string =>
  s[0] === ch ? trimStart(s.substr(1), ch) : s

const trimEnd = (s: string, ch: string): string =>
  s[s.length - 1] === ch ? trimEnd(s.substr(0, s.length - 1), ch) : s

export const makeListBucketPath = (s3: S3): ListBucketPath => {
  return async ({ bucket, path }) => {
    const prefix = trimStart(trimEnd(path, '/') + '/', '/')
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
              MaxKeys: 2147483647, // Maximum allowed by S3 API
              Delimiter: '/',
              Prefix: prefix,
              ContinuationToken: awsData.NextContinuationToken,
            })
            .promise()

          await populateResult(result.$response.data as S3.ListObjectsV2Output)
        } catch (error) {}
      }
    }

    try {
      const response = await s3
        .listObjectsV2({
          Bucket: bucket,
          MaxKeys: 1000000, // Maximum allowed by Wasabi S3 API
          Delimiter: '/',
          Prefix: prefix,
          StartAfter: prefix, // removes the folder name from listing
        })
        .promise()

      await populateResult(response.$response.data as S3.ListObjectsV2Output)
    } catch (error) {
      result.error = error
    }

    return result
  }
}

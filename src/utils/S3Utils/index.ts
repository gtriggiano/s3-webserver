import { S3 } from 'aws-sdk'

import { GetBucketKey, makeGetBucketKey } from './makeGetBucketKey'
import {
  GetBucketPathTree,
  makeGetBucketPathTree,
} from './makeGetBucketPathTree'
import { ListBucketPath, makeListBucketPath } from './makeListBucketPath'

export { GetBucketPathTreeResult } from './makeGetBucketPathTree'
export { ListBucketPathResult } from './makeListBucketPath'
export { GetBucketKeyResult } from './makeGetBucketKey'

export interface S3Utils {
  getBucketPathTree: GetBucketPathTree
  getBucketKey: GetBucketKey
  listBucketPath: ListBucketPath
}

export const S3Utils = (s3: S3): S3Utils => {
  const listBucketPath = makeListBucketPath(s3)
  const getBucketPathTree = makeGetBucketPathTree(listBucketPath)
  const getBucketKey = makeGetBucketKey(s3)

  return {
    getBucketPathTree,
    getBucketKey,
    listBucketPath,
  }
}

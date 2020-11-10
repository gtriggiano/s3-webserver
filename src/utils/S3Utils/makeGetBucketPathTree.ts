import { AWSError } from 'aws-sdk'

import { ListBucketPath } from './makeListBucketPath'

interface GetBucketPathTreeInput {
  bucket: string
  path: string
  depth: number
}

type Tree = { [key: string]: string | Tree }

export interface GetBucketPathTreeResult {
  tree: Tree
  error?: AWSError
}

export type GetBucketPathTree = (
  input: GetBucketPathTreeInput,
) => Promise<GetBucketPathTreeResult>

export const makeGetBucketPathTree = (
  listBucketPath: ListBucketPath,
): GetBucketPathTree => {
  const getBucketPathTree: GetBucketPathTree = async ({
    bucket,
    depth,
    path,
  }) => {
    const tree: Tree = {}

    try {
      depth = toSafeDepth(depth)

      const listResult = await listBucketPath({ bucket, path })

      if (listResult.error) {
        return {
          tree,
          error: listResult.error,
        }
      }

      listResult.files.forEach((file) => {
        tree[getLastPathPart(file)] = file
      })

      if (listResult.folders.length) {
        await Promise.all(
          listResult.folders.map(async (folder) => {
            if (depth > 0) {
              const { tree } = await getBucketPathTree({
                bucket,
                depth: depth - 1,
                path: folder,
              })
              tree[getLastPathPart(path)] = tree
            } else {
              tree[getLastPathPart(path)] = {}
            }
          }),
        )
      }

      return {
        tree,
      }
    } catch (error) {
      return {
        tree,
        error,
      }
    }
  }

  return getBucketPathTree
}

const trimEnd = (s: string, ch: string): string =>
  s[s.length - 1] === ch ? trimEnd(s.substr(0, s.length - 1), ch) : s

const toSafeDepth = (u: number): number => {
  let n = Number(u)
  n = Number.isNaN(n) ? Number.MAX_SAFE_INTEGER : n
  return n < 0 ? Number.MAX_SAFE_INTEGER : n
}

const getLastPathPart = (path: string): string => {
  path = trimEnd(path, '/')
  const lastIndex = path.lastIndexOf('/')
  return path.substr(lastIndex + 1)
}

import { bool, cleanEnv, port, str } from 'envalid'

const env = cleanEnv(process.env, {
  HTTP_PORT: port({ default: 80 }),
  HTTPS_PORT: port({ default: 443 }),
  AWS_ACCESS_KEY_ID: str(),
  AWS_SECRET_ACCESS_KEY: str(),
  S3_BUCKET: str(),
  S3_ENDPOINT: str(),
  S3_FOLDER: str({ default: '' }),
  S3_FORCE_PATH_STYLE: bool({ default: false }),
  S3_IMMUTABLE_TREE: bool({ default: false }),
  S3_CACHE_RESPONSES: bool({ default: true }),
  SSL_CERT: str({ default: '' }),
  SSL_KEY: str({ default: '' }),
  SSL_KEY_PASSPHRASE: str({ default: '' }),

  DEFAULT_404_FILE: str({ default: '' }),
  DEFAULT_403_FILE: str({ default: '' }),
  ENABLE_DIRECTORY_LISTING: bool({ default: false }),
  FOLDER_INDEX_FILE_NAME: str({ default: 'index.html' }),
})

export const { HTTP_PORT, HTTPS_PORT } = env

export const S3 = {
  ACCESS_KEY_ID: env.AWS_ACCESS_KEY_ID,
  SECRET_ACCESS_KEY: env.AWS_SECRET_ACCESS_KEY,
  BUCKET: env.S3_BUCKET,
  ENDPOINT: env.S3_ENDPOINT.replace(/\/$/, ''),
  FOLDER: env.S3_FOLDER.replace(/^\/|\/$/g, ''),
  FORCE_PATH_STYLE: env.S3_FORCE_PATH_STYLE,
  IMMUTABLE_TREE: env.S3_IMMUTABLE_TREE,
  CACHE_RESPONSES: env.S3_CACHE_RESPONSES,
}
export type S3 = typeof S3

export const App = {
  enableSSL: Boolean(env.SSL_CERT && env.SSL_KEY),
  SSL_CERT: env.SSL_CERT,
  SSL_KEY: env.SSL_KEY,
  SSL_KEY_PASSPHRASE: env.SSL_KEY_PASSPHRASE,

  DEFAULT_404_FILE: env.DEFAULT_404_FILE,
  DEFAULT_403_FILE: env.DEFAULT_403_FILE,
  ENABLE_DIRECTORY_LISTING: env.ENABLE_DIRECTORY_LISTING,
  FOLDER_INDEX_FILE_NAME: env.FOLDER_INDEX_FILE_NAME,
}
export type App = typeof App

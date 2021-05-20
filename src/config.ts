import { bool, cleanEnv, num, port, str } from 'envalid'

const env = cleanEnv(process.env, {
  HTTP_PORT: port({ default: 80 }),
  HTTPS_PORT: port({ default: 443 }),
  AWS_ACCESS_KEY_ID: str(),
  AWS_SECRET_ACCESS_KEY: str(),
  S3_BUCKET: str(),
  S3_ENDPOINT: str(),
  S3_SIGNATURE_VERSION: str({ default: 'v4', choices: ['v2', 'v3', 'v4'] }),
  S3_FOLDER: str({ default: '' }),
  S3_FORCE_PATH_STYLE: bool({ default: false }),
  S3_IMMUTABLE_TREE: bool({ default: false }),
  S3_CACHE_RESPONSES: bool({ default: true }),
  S3_CACHE_TTL: num({ default: 60 }),
  S3_LOG_LEVEL: str({ default: 'none', choices: ['none', 'info', 'error'] }),
  SSL_CERT: str({ default: '' }),
  SSL_KEY: str({ default: '' }),
  SSL_KEY_PASSPHRASE: str({ default: '' }),

  TRUST_PROXY: bool({ default: true }),
  ENABLE_CSP: bool({ default: false }),
  DEFAULT_404_FILE: str({ default: '' }),
  DEFAULT_403_FILE: str({ default: '' }),
  ENABLE_DIRECTORY_LISTING: bool({ default: false }),
  FOLDER_INDEX_FILE_NAME: str({ default: 'index.html' }),
  CACHE_CONTROL_REGEXP_LIST: str({ default: '' }),
  CACHE_CONTROL_REGEXP_BLACKLIST: str({ default: '' }),
  CACHE_CONTROL_MAX_AGE: num({ default: 60 * 60 * 24 * 5 /** 5 days */ }),
  LOG_HTTP_CALLS: bool({ default: true }),

  HANDLE_GATSBY_REDIRECTS: bool({ default: false }),
})

export const {
  HTTP_PORT,
  HTTPS_PORT,
  SSL_CERT,
  SSL_KEY,
  SSL_KEY_PASSPHRASE,
} = env

export const S3 = {
  ACCESS_KEY_ID: env.AWS_ACCESS_KEY_ID,
  SECRET_ACCESS_KEY: env.AWS_SECRET_ACCESS_KEY,
  BUCKET: env.S3_BUCKET,
  ENDPOINT: env.S3_ENDPOINT.replace(/\/$/, ''),
  SIGNATURE_VERSION: env.S3_SIGNATURE_VERSION,
  FOLDER: env.S3_FOLDER.replace(/^\/|\/$/g, ''),
  FORCE_PATH_STYLE: env.S3_FORCE_PATH_STYLE,
  IMMUTABLE_TREE: env.S3_IMMUTABLE_TREE,
  CACHE_RESPONSES: env.S3_CACHE_RESPONSES,
  CACHE_TTL: Math.min(10, Math.floor(env.S3_CACHE_TTL)),
  LOG_LEVEL: env.S3_LOG_LEVEL,
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
  TRUST_PROXY: env.TRUST_PROXY,
  ENABLE_CSP: env.ENABLE_CSP,
  CACHE_CONTROL_REGEXP_LIST: env.CACHE_CONTROL_REGEXP_LIST.split('|||')
    .filter((str) => !!str.trim())
    .map((regexp) => new RegExp(regexp)),
  CACHE_CONTROL_REGEXP_BLACKLIST: env.CACHE_CONTROL_REGEXP_BLACKLIST.split(
    '|||',
  )
    .filter((str) => !!str.trim())
    .map((regexp) => new RegExp(regexp)),
  CACHE_CONTROL_MAX_AGE: env.CACHE_CONTROL_MAX_AGE,
  LOG_HTTP_CALLS: env.LOG_HTTP_CALLS,
  HANDLE_GATSBY_REDIRECTS: env.HANDLE_GATSBY_REDIRECTS,
}
export type App = typeof App

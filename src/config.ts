import { cleanEnv, port, str } from 'envalid'

const env = cleanEnv(process.env, {
  HTTP_PORT: port({ default: 80 }),
  HTTPS_PORT: port({ default: 443 }),
  AWS_ACCESS_KEY_ID: str(),
  AWS_SECRET_ACCESS_KEY: str(),
  S3_BUCKET: str(),
  S3_ENDPOINT: str(),
  S3_FOLDER: str({ default: '' }),
  SSL_CERT: str({ default: '' }),
  SSL_KEY: str({ default: '' }),
  SSL_KEY_PASSPHRASE: str({ default: '' }),

  DOCUMENT_404_KEY: str({ default: '' }),
})

export const { HTTP_PORT, HTTPS_PORT, DOCUMENT_404_KEY } = env

export const S3 = {
  ACCESS_KEY_ID: env.AWS_ACCESS_KEY_ID,
  SECRET_ACCESS_KEY: env.AWS_SECRET_ACCESS_KEY,
  BUCKET: env.S3_BUCKET,
  ENDPOINT: env.S3_ENDPOINT.replace(/\/$/, ''),
  FOLDER: env.S3_FOLDER.replace(/^\/|\/$/g, ''),
}
export type S3 = typeof S3

export const TemplatedApp = {
  enableSSL: Boolean(env.SSL_CERT && env.SSL_KEY),
  SSL_CERT: env.SSL_CERT,
  SSL_KEY: env.SSL_KEY,
  SSL_KEY_PASSPHRASE: env.SSL_KEY_PASSPHRASE,
}
export type TemplatedApp = typeof TemplatedApp

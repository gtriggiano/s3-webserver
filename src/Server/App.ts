import { App as NoSSLApp, SSLApp, TemplatedApp } from 'uWebSockets.js'

import { TemplatedApp as TemplatedAppConfig } from '../config'

export const App = ({
  enableSSL,
  SSL_CERT,
  SSL_KEY,
  SSL_KEY_PASSPHRASE,
}: TemplatedAppConfig): TemplatedApp => {
  return enableSSL
    ? SSLApp({
        cert_file_name: SSL_CERT,
        key_file_name: SSL_KEY,
        passphrase: SSL_KEY_PASSPHRASE || undefined,
      })
    : NoSSLApp()
}

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package template

const DefaultRobot = `{{ with $w := . -}}
cryptoSrc: local
defaultBatchLimits:
  batchBlocksCountLimit: 10
  batchLenLimit: 1000
  batchSizeLimit: 100000
  batchTimeoutLimit: 300ms
defaultRobotExecOpts:
  executeTimeout: 30s
delayAfterChRobotError: 3s
googleCryptoSettings:
  gcloudCreds: null
  gcloudProject: null
  userCert: null
logLevel: debug
logType: lr-txt-dev
profilePath: {{ .ConnectionPath User }}
redisStor:
  addr:
    - 127.0.0.1:6379
  dbPrefix: robot
  password: ""
  rootCAs: {{ .CACertsBundlePath }}
  withTLS: false
robots:{{ range .Channels }}
  {{- if ne . "acl" }}
  {{- $chName := . }}
  - chName: {{ . }}
    collectorsBufSize: 1000
    src: {{ range $w.Channels }}
      {{- if and (ne . "acl") (ne . $chName) }}
      - chName: {{ . }}
        initBlockNum: 0
      {{- end }}
    {{- end }}
  {{- end }}
{{- end }}
serverPort: {{ .RobotPort "Listen" }}
txMultiSwapPrefix: multi_swap
txPreimagePrefix: batchTransactions
txSwapPrefix: swaps
userName: backend
vaultCryptoSettings:
  useRenewableVaultTokens: false
  userCert: ""
  vaultAddress: http://vault.vault:8200
  vaultAuthPath: /v1/auth/kubernetes/login
  vaultNamespace: atomyze/robot/
  vaultRole: ""
  vaultServiceTokenPath: null
  vaultToken: ""
{{ end }}
`

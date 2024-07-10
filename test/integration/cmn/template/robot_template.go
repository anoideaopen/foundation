/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package template

const DefaultRobot = `{{ with $w := . -}}
defaultBatchLimits:
  batchBlocksCountLimit: 10
  batchLenLimit: 1000
  batchSizeLimit: 100000
  batchTimeoutLimit: 300ms
defaultRobotExecOpts:
  executeTimeout: 30s
delayAfterChRobotError: 3s
logLevel: debug
logType: lr-txt
profilePath: {{ .ConnectionPath User }}
redisStor:
  addr:{{ range .Robot.RedisAddresses }}
    - {{ . }}
  {{- end }}
  dbPrefix: robot
  password: ""
  rootCAs: {{ .CACertsBundlePath }}
  withTLS: false
robots:{{ range .Channels }}
  {{- if ne . "acl" }}
  - chName: {{ . }}
    collectorsBufSize: 1000
    src: {{- range $w.Channels }}
      {{- if ne . "acl" }}
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
{{ end }}
`

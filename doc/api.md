
# API

Description of API functions and examples of their usage

# TOC

- [API](#api)
- [TOC](#toc)
  - [Methods BaseContract](#methods-basecontract)
    - [QueryBuildInfo](#querybuildinfo)
    - [QueryCoreChaincodeIDName](#querycorechaincodeidname)
    - [QueryNameOfFiles](#querynameoffiles)
    - [QuerySrcFile](#querysrcfile)
    - [QuerySrcPartFile](#querysrcpartfile)
    - [QuerySystemEnv](#querysystemenv)
  - [Example](#example)
- [Links](#links)

## Methods BaseContract

Methods of the `BaseContract` structure. Any chaincode that embeds `BaseContract` has these methods.

### QueryBuildInfo

```
func (bc *BaseContract) QueryBuildInfo() (*debug.BuildInfo, error)
```

QueryBuildInfo returns the result of evaluating `debug.ReadBuildInfo()` in the chaincode.

### QueryCoreChaincodeIDName

```
func (bc *BaseContract) QueryCoreChaincodeIDName() (string, error)
```

QueryCoreChaincodeIDName returns the value of the environment variable `CORE_CHAINCODE_ID_NAME` in the chaincode.

### QueryNameOfFiles

```
func (bc *BaseContract) QueryNameOfFiles() ([]string, error)
```

QueryNameOfFiles returns a list of names of source code files embedded in the chaincode (see [embedded](embed.md)).

### QuerySrcFile

```
func (bc *BaseContract) QuerySrcFile(name string) (string, error)
```

QuerySrcFile returns the source code file embedded in the chaincode with the specified name.

### QuerySrcPartFile

```
func (bc *BaseContract) QuerySrcPartFile(name string, start int, end int) (string, error)
```

QuerySrcPartFile returns a portion of the source code file (if the file is large) embedded in the chaincode, specified by the start and end indices.

### QuerySystemEnv

```
func (bc *BaseContract) QuerySystemEnv() (map[string]string, error)
```

QuerySystemEnv returns the system environment of the chaincode. This includes files:
- `/etc/issue`
- `/etc/resolv.conf`
- `/proc/meminfo`
- `/proc/cpuinfo`
- `/etc/timezone`
- `/proc/diskstats`
- `/proc/loadavg`
- `/proc/version`
- `/proc/uptime`
- `/etc/hyperledger/fabric/client.crt`
- `/etc/hyperledger/fabric/peer.crt`

## Example

All examples are designed for sending to hlf-proxy.

- QueryBuildInfo
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "buildInfo",
  "args": []
}'
```

- QueryCoreChaincodeIDName
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "coreChaincodeIDName",
  "args": []
}'
```

- QueryNameOfFiles
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "nameOfFiles",
  "args": []
}'
```

- QuerySrcFile
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "srcFile",
  "args": ["folder/token.go"]
}'
```

- QuerySrcPartFile
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "srcPartFile",
  "args": ["folder/token.go", "8", "23"]
}'
```

- QuerySystemEnv
```shell
curl -X 'POST' \
  'http://127.0.0.1:9001/query' \
  -H 'accept: */*' \
  -H 'Content-Type: application/json' \
  -d '{
  "channel": "cc",
  "chaincodeId": "cc",
  "fcn": "systemEnv",
  "args": [],
  "options": {
    "targetEndpoints": ["test-peer-001.org0"]
  }
}'
```

# Links

* No
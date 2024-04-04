# EMBED

Description of embedding source code in chaincode.

## Table of Contents
- [EMBED](#embed)
  - [Table of Contents](#table-of-contents)
    - [Prepare Chaincode for Embed](#prepare-chaincode-for-embed)
      - [Correct Chaincode](#correct-chaincode)
      - [Incorrect Chaincode](#incorrect-chaincode)
    - [Embed Chaincode](#embed-chaincode)
  - [Links](#links)

### Prepare Chaincode for Embed

To include source code files in the chaincode, you need to organize the code so that all *.go files are in the same directory as the file with the main() function, or in lower-level directories. *.go files should not be in higher-level directories or in parallel directories (to access them, you need to go up to the parent directory).

#### Correct Chaincode

```
.
├── go.mod
├── go.sum
├── industrial_token
│   ├── buyback.go
│   ├── distribute.go
│   ├── metadata.go
│   ├── methods.go
│   ├── redeem.go
│   ├── token.go
│   └── transfer.go
└── main.go
```

#### Incorrect Chaincode

```
.
├── README.md
├── go.mod
├── go.sum
├── realizations
│   ├── fiat
│   │   └── fiat.go
│   └── rub
│       └── rub.go
├── token
│   ├── balances.go
│   ├── fiat.go
│   ├── fiat_test.go
│   ├── methods.go
│   ├── trading.go
│   └── trading_test.go
└── vendor.tar.gz
```

### Embed Chaincode

To embed files in the chaincode, you need to add the section in the file with the main() function.

```go
//go:embed go.mod go.sum *.go industrial_token/*.go
var f embed.FS
```

Files that should be included:
- `go.mod`
- `go.sum`
- `*.go`
- Go files from all child directories (`folder1/*.go folder1/folder2/*go`)

When creating the chaincode, pass the parameter `f`.

```go
cc, err := core.NewChainCode(ft, "org0", nil, core.WithSrcFS(&f))
```

If you don't want to use `embed.FS`, you can create the chaincode in the old way.

```go
cc, err := core.NewChainCode(ft, "org0", nil)
```

Or set `embed.FS` to `nil`.

```go
cc, err := core.NewChainCode(ft, "org0", nil, core.WithSrcFS(nil))
```

## Links

* None

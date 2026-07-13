# AGENTS.md — Foundation

Foundation is a **Go library** (module `github.com/anoideaopen/foundation`) for building Hyperledger Fabric v2 and v3 chaincodes.

## Commands

```shell
go build ./...                                 # build
go fix ./...                                   # fix API usage after Go version bump
golangci-lint run                              # lint (v2 config, 59 linters)
go test -count 1 ./...                         # unit tests (no cache)
go test ./... -coverprofile=./coverage.out     # coverage (excludes *.pb.go)
go generate ./proto/...                        # protoc + counterfeiter mocks
go generate ./core/routing/grpc/...            # gRPC method options proto
go generate ./test/unit/...                    # test ext_config proto
```

## Multi-module

| Module | Path | Purpose |
|--------|------|---------|
| `github.com/anoideaopen/foundation` | `.` | Main library |
| `github.com/anoideaopen/foundation/test/integration` | `test/integration/` | Integration tests (separate `go.mod`, replaces `../../`) |
| `github.com/anoideaopen/foundation/fixture/gost` | `fixture/gost/` | GOST crypto fixture (separate `go.mod`) |

Run commands from the correct module directory.

## Integration tests

Prerequisites: Docker images `redis:7.2.4`, `hyperledger/fabric-ccenv`, `hyperledger/fabric-baseos`.

```shell
go install github.com/onsi/ginkgo/v2/ginkgo@v2.31.0
cd test/integration
ginkgo --keep-going --poll-progress-after 60s --timeout 24h <suite-names>
```

CI matrix suites (space-separated when passed to ginkgo):
- `general basic custom-acl-channel`
- `swap multisigned-user`
- `version-and-nonce`
- `channel-transfer-only-tx`
- `channel-transfer`
- `task_executor-only-tx channel-transfer-multi channel-transfer-task-executor-only-tx`

## CI pipeline (strict order)

1. **check-cyrillic-comments** — `grep` fails on Cyrillic chars in non-`.github` files
2. **validate-go** — `go mod tidy` + `git diff --exit-code`, then `go fmt ./...` + `git diff --exit-code`
3. **golangci-lint** — v2.9.0 action
4. **go-test-unit** — `go test -count 1 ./...`
5. **go-test-coverage** — `go test ./... -coverprofile=./coverage.out` + threshold check
6. **go-test-integration** — ginkgo matrix (`fail-fast: false`)

## Codegen

- `proto/generate.go` — protoc for 6 `.proto` files + `protoc-gen-validate` + counterfeiter mocks (`shim.ChaincodeStubInterface`, `shim.StateQueryIteratorInterface`)
- Required tools: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-validate`
- Counterfeiter: `tool github.com/maxbrunsfeld/counterfeiter/v6` in `go.mod`
- Generated files (`.pb.go`, `.pb.validate.go`) are committed

## Architecture

- **Library, not a binary.** Test chaincodes in `test/chaincode/` show real usage.
- **Reflect router method prefixes** (`core/routing/reflect/`):
  - `Tx*` — batched invoke, requires `*types.Sender` param
  - `NBTx*` — no-batch invoke, immediate execution
  - `Query*` — read-only, no state changes
- **gRPC + reflect routers** combine via `routing/mux` (see `test/chaincode/fiat/`).
- **Chaincode entrypoint** must embed source: `core.NewCC(impl, core.WithSrcFS(&f))`.
- **Config** loaded in `Init()` from JSON args or positional args.
- **Key types**: `ed25519` (default), `secp256k1` (Ethereum), `gost` (Russian standards).
- **Balance types**: single-byte prefixes (`0x2b` Token, `0x2e` TokenLocked, `0x2c` Allowed, ...).

## Style

- **No Cyrillic** in any source file (CI-enforced).
- `.golangci.yml` sets `tests: false` (does not lint test files), excludes generated code.
- Dot-imports allowed only for `gomega` and `ginkgo/v2`.

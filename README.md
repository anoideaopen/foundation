# Foundation

[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/8822/badge)](https://www.bestpractices.dev/projects/8822)
[![Go Report Card](https://goreportcard.com/badge/github.com/anoideaopen/foundation)](https://goreportcard.com/report/github.com/anoideaopen/foundation)
[![Go Reference](https://pkg.go.dev/badge/github.com/anoideaopen/foundation.svg)](https://pkg.go.dev/github.com/anoideaopen/foundation)
![GitHub License](https://img.shields.io/github/license/anoideaopen/foundation)

[![Go Verify Build](https://github.com/anoideaopen/foundation/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/anoideaopen/foundation/actions/workflows/go.yml)
[![Security vulnerability scan](https://github.com/anoideaopen/foundation/actions/workflows/vulnerability-scan.yml/badge.svg?branch=main)](https://github.com/anoideaopen/foundation/actions/workflows/vulnerability-scan.yml)
![GitHub go.mod Go version (branch)](https://img.shields.io/github/go-mod/go-version/anoideaopen/foundation/main)
![GitHub Tag](https://img.shields.io/github/v/tag/anoideaopen/foundation)

A basic library for creating platform chaincodes.

## Table of Contents
- [Foundation](#foundation)
  - [Table of Contents](#table-of-contents)
  - [Description](#description)
  - [Topics](#topics)
  - [Links](#links)
  - [License](#license)

## Description

The library contains basic primitives for creating chaincodes.

* BaseToken
* BaseContract

The library implements the following functionality:

* Batching
* Swapping
* Multiswap

The library implements functionality for interacting with access control list (ACL) chaincode.

## Topics

* [API](doc/api.md)
* [Chaincode configuration](doc/cc_cfg.md)
* [Versioning](doc/versioning.md)
* [Routing](doc/routing.md)
* [QA](doc/qa.md)
* [Embed Source](doc/embed.md)
* [Swap](doc/swap.md)
* [External Locks](doc/external-locks.md)
* [Balance reverse index](doc/balance-indexing.md)
* [Balance package](core/balance/balance-indexing.md)

## Links

* [Original Repository](https://github.com/anoideaopen/foundation)

## License

[Default License](LICENSE)

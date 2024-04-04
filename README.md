# Foundation

[![Go Verify Build](https://github.com/anoideaopen/foundation/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/anoideaopen/foundation/actions/workflows/go.yml)
[![Security vulnerability scan](https://github.com/anoideaopen/foundation/actions/workflows/vulnerability-scan.yml/badge.svg?branch=main)](https://github.com/anoideaopen/foundation/actions/workflows/vulnerability-scan.yml)

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

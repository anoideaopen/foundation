# External Locks

## Table of Contents
- [External Locks](#external-locks)
  - [Table of Contents](#table-of-contents)
  - [General Information](#general-information)
  - [Description of Functions](#description-of-functions)
    - [lockTokenBalance](#locktokenbalance)
    - [unlockTokenBalance](#unlocktokenbalance)
    - [getLockedTokenBalance](#getlockedtokenbalance)
    - [lockAllowedBalance](#lockallowedbalance)
    - [unlockAllowedBalance](#unlockallowedbalance)
    - [getLockedAllowedBalance](#getlockedallowedbalance)
  - [Errors](#errors)

## General Information

The functions described in this document allow locking and unlocking funds from an external source.

**Key Features:**

- External lock functions allow holding and releasing tokens through direct function calls, while traditional holding is invoked within the core.
- Ability to lock token balance.
- Ability to lock allowed balance.

## Description of Functions

### lockTokenBalance

**Method Name:** `lockTokenBalance`

**Description:** This method allows you to lock a user's balance.

**Features:**

- This method can only be called by the chaincode admin.
- The function checks for the existence of a duplicate request (by ID). If there is a duplicate, an error is returned.

**In the request (proto with JSON serialization):**

```go
string id = 1; // lock identifier (optional parameter, if not specified, txID is used)
string address = 2; // owner's address (required parameter)
string token = 3; // token identifier/ticker (required parameter)
string amount = 4; // big.Int amount of tokens to lock (required parameter)
string reason = 5; // reason for locking (required parameter)
repeated bytes docs = 6 ; // hashes of documents with justification (optional parameter)
bytes payload = 7 ; // additional information (optional parameter)
```

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string amount = 4; // big.Int amount of tokens to lock
string reason = 5; // reason for locking
repeated bytes docs = 6; // hashes of documents with justification (optional parameter)
bytes payload = 7; // additional information (optional parameter)
```

### unlockTokenBalance

**Method:** `unlockTokenBalance`

**Description:** This method allows you to release a user's balance.

**Features:**

- Tokens can be unlocked either in full or partially.
- The method must be called by the chaincode admin.
- It checks the existence of the request.
- It is not possible to unlock more funds than were locked.

**The same parameters are passed in the request as in `lockTokenBalance`.**

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string amount = 4; // big.Int amount of tokens to unlock
string reason = 5; // reason for locking
repeated bytes docs = 6; // hashes of documents with justification (optional parameter)
bytes payload = 7; // additional information (optional parameter)
bool complete_operation = 8; // flag indicating that it is completely unlocked
```

### getLockedTokenBalance

**Method:** `getLockedTokenBalance`

**Description:** This function returns information about a user's locked tokens for Token Balance.

**In the request, you pass:

- `Lock ID` - The identifier of the lock operation returned after executing the `lockTokenBalance` function.

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier (optional parameter, if not specified, txID is used)
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string init_amount = 4; // big.Int initial amount of tokens to lock
string current_amount = 5; // big.Int current amount of locked tokens
string reason = 6; // reason for locking
repeated bytes docs = 7; // hashes of documents with justification (optional parameter)
bytes payload = 8; // additional information (optional parameter)
```

### lockAllowedBalance

**Method:** `lockAllowedBalance`

**Description:** This method allows you to lock a user's balance.

**Features, validation, and parameters duplicate the description for `lockTokenBalance`.**

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string amount = 4; // big.Int amount of tokens to lock
string reason = 5; // reason for locking
repeated bytes docs = 6; // hashes of documents with justification (optional parameter)
bytes payload = 7; // additional information (optional parameter)
```

### unlockAllowedBalance

**Method:** `unlockAllowedBalance`

**Description:** This method allows you to release (partially or completely) an allowed balance of a user.

**Features, validation, and parameters duplicate the description for** `unlockTokenBalance`.

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string amount = 4; // big.Int amount of tokens to unlock
string reason = 5; // reason for locking
repeated bytes docs = 

6; // hashes of documents with justification (optional parameter)
bytes payload = 7; // additional information (optional parameter)
bool complete_operation = 8; // flag indicating that it is completely unlocked
```

### getLockedAllowedBalance

**Method:** `getLockedAllowedBalance`

**Description:** This function allows you to get information about locked tokens for an application created by the request.

**Features, validation, parameters, and description duplicate the description for** `getLockedTokenBalance`.

**The response includes (proto with JSON serialization):**

```go
string id = 1; // lock identifier (optional parameter, if not specified, txID is used)
string address = 2; // owner's address
string token = 3; // token identifier/ticker
string init_amount = 4; // big.Int initial amount of tokens to lock
string current_amount = 5; // big.Int current amount of locked tokens
string reason = 6; // reason for locking
repeated bytes docs = 7; // hashes of documents with justification (optional parameter)
bytes payload = 8; // additional information (optional parameter)
```

## Errors

Errors are returned with each request. If something goes wrong, the following errors are passed:

- `ErrBigIntFromString`: Error converting `string` to `big.Int`.
- `ErrPlatformAdminOnly`: The function is called by a non-chaincode administrator.
- `ErrEmptyLockID`: When submitting requests to unlock funds or retrieve information about an existing lock, the `Lock ID` is not transmitted (it is returned in the response when executing the `lockTokenBalance` and `lockAllowedBalance` functions).
- `ErrReason`: The reason for the lock is not specified in the `Reason` field (for requests to lock funds for allowed balance and token balance).
- `ErrLockNotExists`: When submitting a request to unlock funds or retrieve information about an existing lock, the submitted `Lock ID` is incorrect.
- `ErrAddressRequired`: The user's wallet is not specified when submitting a request to lock/unlock funds.
- `ErrAmountRequired`: The amount of tokens is not provided in the request to lock/unlock.
- `ErrTokenTickerRequired`: The token ticker is not provided in the request to lock/unlock.
- `ErrAlreadyExist`: When checking for duplicates, identical requests were found (checking is done when requesting to lock allowed balance and token balance).
- `ErrInsufficientFunds`: When checking for sufficient tokens for locking/unlocking funds, a shortage of tokens compared to the request was found.
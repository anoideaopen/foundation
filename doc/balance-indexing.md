# Package documentation `balance`

## Problematics

In the previous version of the foundation library the work with balances was in the core package. The functions were jumbled together, there was no systematization of balance types. There were also extra arguments ```...path`` that were not actually used or used only for one parameter. That is, there was one optional argument.

In order to get the owners of any token, we had to go through the whole stack and filter the necessary tokens from it. As the number of users grows, this becomes a problem.

To solve this problem, it was decided to move atomrar work with balances to a separate package and add additional indexing of balances. At the same time retain compatibility with current tokens.

## Composite keys

### Existing composite keys

To store balances, composite keys were used, consisting of a prefix depending on the type of balance (`BalanceType`) and the address of the owner. In case the balance was associated with a particular token, the token identifier was appended to the key. This information was obtained by parsing the foundation code and tokens.

Examples of composite key structures:

- Without a token: `BalanceTypeString|Address`
- With a token: `BalanceTypeString|Address|TokenID`

### Added composite keys

To optimize the search for token owners, a reverse index was introduced. This index uses composite keys including balance type, token ID and owner address.

An example of a reverse index composite key structure:

- `InverseBalanceObjectType|BalanceTypeString|TokenID|Address`

## Main Functions

### Balance handling functions

#### `Get`

Retrieves the balance value for the specified address and token.

#### `Put`

Saves the balance value for the specified address and token to the ledger.

#### `Move`

Moves the specified quantity from the balance of one address to the balance of another address.

### Operations with balances

#### `Add`

Adds the specified amount to the balance of the specified address and token.

#### `Sub`

Subtracts the specified amount from the balance of the specified address and token.

### Indexing functions

#### `CreateIndex`

Creates an index for states corresponding to the specified balance type.

#### `HasIndexCreatedFlag`

Checks if an index has been created for this balance type.

### Query functions

#### `ListBalancesByAddress`

Retrieves all balance records associated with this address.

#### `ListOwnersByToken`

Retrieves all holders and their balances for a particular token using an index.

## Indexing of token holders

For new tokens, index construction is performed automatically when the balance is saved for the first time using the `Put` function. For existing tokens, it is necessary to manually initiate index creation by calling the `CreateIndex` function. After successful index creation, a flag is set, which is stored in the Ledger and allows to determine whether the index has already been created for this balance type.

The index creation flag is a keyed entry consisting of the `IndexCreatedKey` constant and a string representing `BalanceType`. If the flag is set, that is, the `HasIndexCreatedFlag` function returns `true`, you can query the index without having to traverse the entire stack. This greatly speeds up the process of searching for token owners.

An example of using the indexing flag.
IMPORTANT! THIS IS SAMPLE CODE FOR EXISTING TOKENS. AFTER SUCCESSFUL INDEX CONSTRUCTION, IT IS NECESSARY TO REMOVE THE FLAG AND SPARE LOGIC PATH:

```go
indexExists, err := HasIndexCreatedFlag(stub, balanceTypeToken)
if err != nil {
    // Error handling
}

if indexExists {
    // Executing a query using an index
    owners, err := ListOwnersByToken(stub, balanceTypeToken, tokenID)
    if err != nil {
        // Error handling
    }
    // Working with owner data
} else {
    // The index is not created, you need to traverse the stack or initiate index creation
}
```

For correct operation with indexes in existing tokens, it is necessary to perform indexing. The operation is not dangerous, because the index has a dedicated prefix and does not change the values of balances.
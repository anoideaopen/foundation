# batcher

## Config

To verify creatorSKI by batcherSKI (from proto.ContractConfig)

## API

- batcherBatchExecute
  - method signed by batcher cert
  - needed for fix MVCC conflict by executing few requests in one transaction as batch of requsts used method 
  - count of arguments is 1. string with json BatcherBatchDTO
  - batch requests has a type of transaction, current supported BatcherRequestType is "tx", other execute with error 'unsupported batcher request type'

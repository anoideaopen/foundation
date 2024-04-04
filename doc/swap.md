# SWAP

Description of the features of working with swaps in Foundation.

## Table of Contents

- [SWAP](#swap)
  - [Table of Contents](#table-of-contents)
  - [Swap Cancel](#swap-cancel)
  - [Links](#links)

## Swap Cancel

Rules for invoking/working with `swapCancel`.

1. When performing `swapCancel` transactions, timeouts and senders are no longer checked.

2. The platform (backend) should filter `swapCancel` transactions.
   Only the platform should send `swapCancel` transactions.
   All responsibility for sending `swapCancel` in `HLF` now rests with the platform.

3. The order of canceling swaps.
   For example, when trying to transfer funds from the `FROM` channel to the `TO` channel.
   In this case, you should cancel the swap first in the `TO` channel.
   And only after the successful completion of this transaction, you should cancel the swap in the `FROM` channel.

4. Never execute `swapCancel` on the `FROM` channel before the `TO` channel.
   There is a possibility that you canceled the swap on `FROM`, sent `swapDone` on `TO`, and the platform ended up in a deficit.

## Links

* No
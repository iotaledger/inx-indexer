---
description: Query the Indexer for Outputs.
image: /img/logo/HornetLogo.png
keywords:
- IOTA Node 
- HORNET Node
- Configuration
- REST API
- Indexer
- Simple Outputs
- Query
- how to
---

# Querying Basic Outputs

While the protocol offers different [kinds of outputs](https://github.com/lzpap/tips/blob/master/tips/TIP-0018/tip-0018.md#output-design) and [ways to manage](https://wiki.iota.org/introduction/develop/explanations/what_is_stardust/unlock_conditions) them, most of the time token owners would simply want to know how many tokens they have at their unrestricted disposal. For that you would have to query outputs that satisfy the following limitations:

1. The output must be [basic](https://github.com/lzpap/tips/blob/master/tips/TIP-0018/tip-0018.md#basic-output).
2. The output must have an address unlock condition that matches your address (in this example it will be `rms1qrnspqhq6jhkujxak8aw9vult5uaa38hj8fv9klsvnvchdsf2q06wmr2c7j`).
3. The output must not have any other conditions.

## Construct the Query

To query data from the Shimmer test network, you can use the public node that is run by IOTA Foundation. Its [indexer extension API](https://editor.swagger.io/?url=https://raw.githubusercontent.com/iotaledger/tips/indexer-api/tips/TIP-0026/indexer-rest-api.yaml) is exposed at the following URL:

`https://api.testnet.shimmer.network/api/indexer/v1/outputs/basic`

It returns a thousand basic outputs as there are no any query parameters yet. These parameters would be:


|The Parameter                                                              |Its Meaning                                                  |
|---                                                                        |---                                                          |
|`address=rms1qrnspqhq6jhkujxak8aw9vult5uaa38hj8fv9klsvnvchdsf2q06wmr2c7j`  | Has an address unlock condition with the specified address. |
|`hasStorageDepositReturn=false`                                            | Does not have a storage deposit return unlock condition.    |
|`hasExpiration=false`                                                      | Does not have an expiration unlock condition.               |
|`hasTimelock=false`                                                        | And does not a timelock unlock condition.                   |
|---                                                                        |---                                                          |

Combining everything into the query:

```
https://api.testnet.shimmer.network/api/indexer/v1/outputs/basic?address=rms1qrnspqhq6jhkujxak8aw9vult5uaa38hj8fv9klsvnvchdsf2q06wmr2c7j&hasStorageDepositReturn=false&hasExpiration=false&hasTimelock=false
```

The result will be a JSON with the list of output IDs in the `items` array:

```json
{
  "ledgerIndex": 101,
  "items": [
    "0x0c78e998f5177834ecb3bae1596d5056af76e487386eecb19727465b4be86a790000",
    "0x0c78e998f5177834ecb3bae1596d5056af76e487386eecb19727465b4be86a790100",
    "0x0c78e998f5177834ecb3bae1596d5056af76e487386eecb19727465b4be86a790200"
  ]
}
```
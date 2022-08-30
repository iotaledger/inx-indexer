---
description: INX-Indexer is a ledger indexing tool to provide structured and queryable data to wallets and other applications.
image: /img/Banner/banner_hornet.png
keywords:
- IOTA Node
- Hornet Node
- INX
- Indexer
- IOTA
- Shimmer
- Node Software
- Welcome
- explanation
---

# Welcome to INX-Indexer


To maintain the ledger state, the Hornet nodes need to quickly read and write information about [transaction outputs](https://wiki.iota.org/learn/about-iota/messages#utxo). The token owners, however, are very concerned about the total balance of all UTXO belonging to their accounts — but this information has no use in maintaining the state. If clients were accessing this information directly, that would cause nodes to waste resources on either restructuring the database or resolving unoptimized queries.

INX-Indexer maintains a separate database and optimizes it for wallet-related queries. This way the Hornet software focuses on maintaining the network and INX-Indexer on presenting its state to clients. You can find more information in the [Ledger Indexing](https://github.com/iotaledger/tips/discussions/53) discussion.

## Setup

We recommend you to use the [Docker images](https://hub.docker.com/r/iotaledger/inx-indexer).
These images are also used in the [Docker setup](http://wiki.iota.org/hornet/develop/how_tos/using_docker) of Hornet.

## Configuration

The indexer connects to the local Hornet instance by default.

You can find all the configuration options in the [configuration section](configuration.md).

## API

The indexer exposes a custom set of REST APIs that can be used by wallets and applications to find UTXO in the ledger with a given query.

You can find more information about the API in the [API reference section](api_reference.md).

## Source Code

The source code of the project is available on [GitHub](https://github.com/iotaledger/inx-indexer).
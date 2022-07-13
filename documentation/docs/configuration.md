---
description: This section describes the configuration parameters and their types for INX-Indexer.
keywords:
- IOTA Node 
- Hornet Node
- Indexer
- Configuration
- JSON
- Customize
- Config
- reference
---


# Core Configuration

INX-Indexer uses a JSON standard format as a config file. If you are unsure about JSON syntax, you can find more information in the [official JSON specs](https://www.json.org).

You can change the path of the config file by using the `-c` or `--config` argument while executing `inx-indexer` executable.

For example:
```bash
inx-indexer -c config_defaults.json
```

You can always get the most up-to-date description of the config parameters by running:

```bash
inx-indexer -h --full
```

## <a id="app"></a> 1. Application

| Name            | Description                                                                                            | Type    | Default value |
| --------------- | ------------------------------------------------------------------------------------------------------ | ------- | ------------- |
| checkForUpdates | Whether to check for updates of the application or not                                                 | boolean | true          |
| stopGracePeriod | The maximum time to wait for background processes to finish during shutdown before terminating the app | string  | "5m"          |

Example:

```json
  {
    "app": {
      "checkForUpdates": true,
      "stopGracePeriod": "5m"
    }
  }
```

## <a id="inx"></a> 2. INX

| Name    | Description                            | Type   | Default value    |
| ------- | -------------------------------------- | ------ | ---------------- |
| address | The INX address to which to connect to | string | "localhost:9029" |

Example:

```json
  {
    "inx": {
      "address": "localhost:9029"
    }
  }
```

## <a id="indexer"></a> 3. Indexer

| Name              | Description                                                      | Type   | Default value    |
| ----------------- | ---------------------------------------------------------------- | ------ | ---------------- |
| [db](#indexer_db) | Configuration for Database                                       | object |                  |
| bindAddress       | The bind address on which the Indexer HTTP server listens        | string | "localhost:9091" |
| maxPageSize       | The maximum number of results that may be returned for each page | int    | 1000             |

### <a id="indexer_db"></a> Database

| Name | Description                     | Type   | Default value |
| ---- | ------------------------------- | ------ | ------------- |
| path | The path to the database folder | string | "database"    |

Example:

```json
  {
    "indexer": {
      "db": {
        "path": "database"
      },
      "bindAddress": "localhost:9091",
      "maxPageSize": 1000
    }
  }
```

## <a id="profiling"></a> 4. Profiling

| Name        | Description                                       | Type    | Default value    |
| ----------- | ------------------------------------------------- | ------- | ---------------- |
| enabled     | Whether the profiling plugin is enabled           | boolean | false            |
| bindAddress | The bind address on which the profiler listens on | string  | "localhost:6060" |

Example:

```json
  {
    "profiling": {
      "enabled": false,
      "bindAddress": "localhost:6060"
    }
  }
```

## <a id="prometheus"></a> 5. Prometheus

| Name            | Description                                                     | Type    | Default value    |
| --------------- | --------------------------------------------------------------- | ------- | ---------------- |
| enabled         | Whether the prometheus plugin is enabled                        | boolean | false            |
| bindAddress     | The bind address on which the Prometheus HTTP server listens on | string  | "localhost:9312" |
| goMetrics       | Whether to include go metrics                                   | boolean | false            |
| processMetrics  | Whether to include process metrics                              | boolean | false            |
| promhttpMetrics | Whether to include promhttp metrics                             | boolean | false            |

Example:

```json
  {
    "prometheus": {
      "enabled": false,
      "bindAddress": "localhost:9312",
      "goMetrics": false,
      "processMetrics": false,
      "promhttpMetrics": false
    }
  }
```


# Logging

Daedalus emits structured (JSON) log records at key decision points. Logs are
written to **standard error** so they never interfere with the interface, which
uses standard output.

## Log levels

Set the minimum log level with the `DAEDALUS_LOG_LEVEL` environment variable.
Supported values are `debug`, `info`, `warn`, and `error`. The default is
`info`. Unknown or empty values fall back to `info`.

```sh
# Show debug-level detail
DAEDALUS_LOG_LEVEL=debug ./daedalus

# Only warnings and errors
DAEDALUS_LOG_LEVEL=warn ./daedalus
```

## What the logs look like

Each record is a single JSON object with at least a timestamp, level, and
message, plus any contextual key/value fields:

```json
{"time":"2026-06-23T10:00:00Z","level":"INFO","msg":"daedalus starting","version":"0.1.0-dev","interactive":true}
```

## Capturing logs

Because logs go to standard error, you can redirect them independently of normal
output:

```sh
./daedalus 2> daedalus.log
```

## Privacy

Daedalus does not log secrets, tokens, credentials, or personal data. Log
records contain identifiers and the decisions taken, never the sensitive values
behind them.

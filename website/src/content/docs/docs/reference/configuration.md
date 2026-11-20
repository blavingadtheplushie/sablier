---
title: Configuration
description: Files and environment variables sablier uses.
---

## Config file

`~/.sablier.cfg`, written by `sablier login`, readable only by you:

```ini
api_key = sab_...
api_url = https://...
```

## Data folder

The local database, `sablier.db`, is in:

| OS | Folder |
| --- | --- |
| macOS | `~/Library/Application Support/sablier` |
| Linux | `$XDG_DATA_HOME/sablier`, or `~/.local/share/sablier` |
| Windows | `%AppData%\sablier` |

## Environment variables

| Variable | Effect |
| --- | --- |
| `SABLIER_CONFIG` | path of the config file |
| `SABLIER_HOME` | data folder |
| `SABLIER_DEBUG=1` | also logs raw hook payloads to `hooks-debug.jsonl` in the data folder. They contain prompt text, and never leave your machine |

## Sync

When you're signed in, events are uploaded when an agent starts, every 2 minutes while it runs, and once more when it exits (giving up after 5 seconds if you're offline). Anything not uploaded is sent next time, or with `sablier sync`.

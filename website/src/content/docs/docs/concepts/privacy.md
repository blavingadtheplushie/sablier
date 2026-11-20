---
title: Privacy
description: Are we evil,,,
---

## What we never look at

- prompt text, which is only kept as a salted hash so we can see if you just spam the same prompt
- what you type, we only record events _when_ you type
- code, file contents and tool inputs or outputs
- command lines
- local paths

## What's sent to the server

Sablier works with a bunch of events, heartbeats that we use along with some heuristics to calculate the time you spent on something. (The server's algorithm is the same as the one used in `sablier today`)

| Event            | Data                                                                        |
| ---------------- | --------------------------------------------------------------------------- |
| input            | window length, key count, scroll and click count, whether you pasted        |
| focus            | in or out                                                                   |
| prompt           | the hash, whether it was queued, whether it was a short approval like "yes" |
| turn start / end | the agent's turn id, and why it ended                                       |
| tool call        | its category (shell, edit, read, web, agent, other) and duration            |
| approval         | allowed or not, how long you took, the tool's category                      |
| session start    | the agent's own session id, and the model when it reports one               |

Your machine also gets a random install id.

## What's stored on your device

Everything is in a sqlite database in your data folder (see [configuration](/docs/reference/configuration/)).

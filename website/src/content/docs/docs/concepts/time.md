---
title: How sablier counts time
description: uh what is time.
---

Sablier tracks time **you** spent doing things, never when the agent's doing all the work.

All the rules for how we compute time from events is in [`internal/credit`](https://github.com/espcaa/sablier/blob/main/internal/credit/credit.go). We store events on the server, not time, so we can always change those and rederive time from heartbeats.

## Heartbeats

| Event            | From     | When                                                    |
| ---------------- | -------- | ------------------------------------------------------- |
| input            | launcher | every 10 seconds you typed, scrolled, clicked or pasted |
| focus            | launcher | you switch to or away from the terminal                 |
| prompt           | agent    | you send a prompt                                       |
| approval         | agent    | you allow or reject a tool                              |
| interrupt        | agent    | you stop the agent mid turn                             |
| turn start / end | agent    | the agent starts or stops working                       |
| tool call        | agent    | a tool finished running                                 |
| session start    | agent    | a new agent session                                     |

## Heartbeats -> Time rules

1. Actions earn time around them

Every action earns _5 minutes after it_, to read what the agent's doing. (but that gets cancelled if you are not focusing your terminal anymore)\
It can also earn up to _5 minutes before it_, for the time you spent reading the answer and writing your next prompt.

2.  Typing and scrolling earn time

Each 10 second window where you typed or scrolled counts, plus **2 minutes after it**.

3.  Leaving cuts time

If you leave the terminal for _more than 2 minutes_, everything from the moment you left is cut. Quick focuses out/in (checking the docs, a message) aren't counted by that since they usually last less than 2 minutes.

## Example

```
10:00  you send a prompt               → 10:00 to 10:05
10:00  the agent works until 10:12     → nothing from it
10:13  you scroll through its answer   → 10:12:50 to 10:15
10:16  you send the next prompt        → 10:12 to 10:21 (back to when the agent finished)
```

That's 10:00 to 10:05, then 10:12 to 10:21: **14 minutes**, while the session lasted 21.

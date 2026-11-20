---
title: What is sablier?
description: uhhh,,,
---

Sablier starts your agent for you and logs a bunch of pieces of data (heartbeats) when you focus the terminal, write a prompt, approve something, and measures the time **you** actually spent working.

## How it works

You just start your agent through sablier!

```sh
sablier claude
```

Behind the scenes, sablier has two main components:

- A small wrapper around your cli agent that tracks focus, keystrokes and scrolling.
- An observer that checks what your agent is actually doing (approvals, prompts sent, things interrupted, ...)

For now, this only works with terminal based agents, but this architecture could be expanded to apps pretty easily. \
If you're old school and still use an ide, you can also use hackatime in parallel to track the edits you make with your human hands!

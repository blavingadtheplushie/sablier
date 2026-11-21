---
title: sablier login
---

## sablier login

sign in through your browser, or paste your api key with --key

### Synopsis

sign in through your browser: sablier opens a page on the server, you allow
the cli, and the api key is saved for you. nothing to copy.

with --key, paste the api key from the dashboard instead (handy over ssh).

```
sablier login [flags]
```

### Options

```
      --api-url string   server url (default: from the config, else https://sablier.hackclub.cc)
  -h, --help             help for login
      --key              paste an api key instead of signing in through the browser
      --no-browser       print the sign in link instead of opening it
```

### SEE ALSO

* [sablier](../sablier/)	 - track the time you spend coding with ai agents


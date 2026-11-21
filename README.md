# sablier

## Install

```sh
curl -fsSL https://sablier.hackclub.cc/install.sh | sh          # macOS, Linux
irm https://sablier.hackclub.cc/install.ps1 | iex               # Windows
```

Then sign in on https://sablier.hackclub.cc, run `sablier login`, and start your agent through it!

```sh
sablier claude
```

`sablier update` updates it.

## Development

```sh
make dev          # localhost:8080, go reloads with air, the site with astro
make build        # → bin/sablier and bin/sablier-server
make test
make test-server  # needs the local postgres (make db)
make docs         # regenerate the cli reference in website/
make release      # every platform into dist/
```

The CLI talks to https://sablier.hackclub.cc by default. To use a local server: `sablier login --api-url http://localhost:8080`.

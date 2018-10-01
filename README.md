<h1 align="center">
    <img src="asset/static/away.png" alt="Away" width="256">
    <div style="font-size: 48px;font-family: monospace;color: #7F78C8;">A way to Away!</div>
</h1>

## Getting started

Away started with Go 1.9. We recommend installing Go 1.9+.

### Installation

```bash
go get -u -v github.com/A-way/away
```

### Usage

Run remote:

```
away -rp 8080 -pk "passkey you like"
```

Run local:

```
away -lp 1080 -pk "passkey you like" -ru http://remote-url:8080
```

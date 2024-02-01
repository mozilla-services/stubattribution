# dmgmodify

## Usage

```
# From the top-level folder of the repo, we'd have to use `dmgmodify/main.go` instead of `main.go`
go run main.go <attributable firefox DMG> <name of the modified DMG> <stringified attribution data>
```

DMGs can be downloaded from https://ftp.mozilla.org/pub/firefox/. For example,
here is how we could very quickly add attribution data to an attributable DMG:

```
# 1. Download a Firefox DMG (we use Nightly 124 because we know that's an attributable DMG)
wget https://ftp.mozilla.org/pub/firefox/nightly/2024/02/2024-02-01-09-53-46-mozilla-central/firefox-124.0a1.en-US.mac.dmg

# 2. We build a new DMG with random attribution data
go run main.go firefox-124.0a1.en-US.mac.dmg modified.dmg "$(openssl rand -hex 100)"
```

We can then use the [fx-attribution-data-reader][] to verify the data in the
`modified.dmg` file.

[fx-attribution-data-reader]: https://github.com/willdurand/fx-attribution-data-reader

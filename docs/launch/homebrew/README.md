# homebrew-cosmoflare — CosmoLabs Homebrew tap

Install the Cosmoflare CLI (Go library + CLI for the full Cloudflare
developer platform):

```bash
brew tap CosmoLabs-org/cosmoflare
brew install cosmoflare
```

Or one-shot:

```bash
brew install CosmoLabs-org/cosmoflare/cosmoflare
```

## Maintainer: publishing a new version

Releases are cut locally (no CI, by policy). After `make release TAG=vX.Y.Z`
stages `dist/` and the GitHub release is published:

1. Fill the formula from the staged checksums (from the cosmoflare repo):
   ```bash
   TAG=vX.Y.Z; V=X.Y.Z
   for p in darwin-arm64 darwin-amd64 linux-arm64 linux-amd64; do
     sha=$(grep "cosmoflare-$TAG-$p.tar.gz" dist/checksums-sha256.txt | cut -d' ' -f1)
     # uppercase + underscores for the placeholder, e.g. DARWIN_ARM64
     key=$(echo $p | tr 'a-z-' 'A-Z_')
     sed -i '' "s/__SHA256_${key}__/$sha/g" Formula/cosmoflare.rb
   done
   sed -i '' "s/__TAG__/$TAG/g; s/__VERSION__/$V/g" Formula/cosmoflare.rb
   ```
   (Source template lives in cosmoflare's `docs/launch/homebrew/` — copy it
   here first if the formula doesn't exist yet.)
2. `brew audit --formula cosmoflare` and `brew install --build-from-source cosmoflare`
   as a smoke test.
3. Commit + push the formula update.

Archives contain a bare `cosmoflare` binary at the archive root plus
README/LICENSE (fixed packaging since v0.28.1 — older archives with `../`
paths are broken and uninstallable).

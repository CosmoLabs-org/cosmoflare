# Cosmoflare Homebrew formula (ROAD-089).
# Template: __SHA256_*__ placeholders are filled from dist/checksums at
# release time — see docs/launch/homebrew/README.md.
class Cosmoflare < Formula
  desc "Go CLI and library for the full Cloudflare developer platform"
  homepage "https://github.com/CosmoLabs-org/cosmoflare"
  url "https://github.com/CosmoLabs-org/cosmoflare/releases/download/__TAG__/cosmoflare-__TAG__-darwin-arm64.tar.gz"
  sha256 "__SHA256_DARWIN_ARM64__"
  version "__VERSION__"
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  on_macos do
    on_arm do
      url "https://github.com/CosmoLabs-org/cosmoflare/releases/download/__TAG__/cosmoflare-__TAG__-darwin-arm64.tar.gz"
      sha256 "__SHA256_DARWIN_ARM64__"
    end
    on_intel do
      url "https://github.com/CosmoLabs-org/cosmoflare/releases/download/__TAG__/cosmoflare-__TAG__-darwin-amd64.tar.gz"
      sha256 "__SHA256_DARWIN_AMD64__"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/CosmoLabs-org/cosmoflare/releases/download/__TAG__/cosmoflare-__TAG__-linux-arm64.tar.gz"
      sha256 "__SHA256_LINUX_ARM64__"
    end
    on_intel do
      url "https://github.com/CosmoLabs-org/cosmoflare/releases/download/__TAG__/cosmoflare-__TAG__-linux-amd64.tar.gz"
      sha256 "__SHA256_LINUX_AMD64__"
    end
  end

  def install
    bin.install "cosmoflare"
  end

  def caveats
    <<~EOS
      Set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID, or run
      `cosmoflare config init` for the interactive setup wizard.
    EOS
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/cosmoflare --version")
  end
end

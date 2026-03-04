# Template Homebrew formula for whoopy.
# Copy to the totocaster/homebrew-tap repository; GoReleaser updates fields on release.

class Whoopy < Formula
  desc "Official WHOOP data CLI"
  homepage "https://github.com/totocaster/whoopy"
  version "0.1.0"
  license "MIT"

  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/totocaster/whoopy/releases/download/v0.1.0/whoopy_0.1.0_Darwin_arm64.tar.gz"
      sha256 "TO_BE_UPDATED_BY_GORELEASER"
    else
      url "https://github.com/totocaster/whoopy/releases/download/v0.1.0/whoopy_0.1.0_Darwin_x86_64.tar.gz"
      sha256 "TO_BE_UPDATED_BY_GORELEASER"
    end
  elsif OS.linux?
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/totocaster/whoopy/releases/download/v0.1.0/whoopy_0.1.0_Linux_arm64.tar.gz"
      sha256 "TO_BE_UPDATED_BY_GORELEASER"
    else
      url "https://github.com/totocaster/whoopy/releases/download/v0.1.0/whoopy_0.1.0_Linux_x86_64.tar.gz"
      sha256 "TO_BE_UPDATED_BY_GORELEASER"
    end
  end

  depends_on "go" => :optional

  def install
    bin.install "whoopy"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/whoopy --version")
  end
end

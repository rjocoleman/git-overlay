# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class GitOverlay < Formula
  desc "Git overlay tool for managing upstream repositories"
  homepage "https://github.com/rjocoleman/git-overlay"
  version "0.1.0"

  on_macos do
    on_intel do
      url "https://github.com/rjocoleman/git-overlay/releases/download/v0.1.0/git-overlay_Darwin_x86_64.tar.gz"
      sha256 "cd34bb9e535b5669b4d2f817d14e6eeb02ad86bcfe8084c000af5b23e1391753"

      def install
        bin.install "git-overlay"
      end
    end
    on_arm do
      url "https://github.com/rjocoleman/git-overlay/releases/download/v0.1.0/git-overlay_Darwin_arm64.tar.gz"
      sha256 "59a99750004225243a572347d2b9b3ddfe252f019f58c3a68b9cb38dd40ae384"

      def install
        bin.install "git-overlay"
      end
    end
  end

  on_linux do
    on_intel do
      if Hardware::CPU.is_64_bit?
        url "https://github.com/rjocoleman/git-overlay/releases/download/v0.1.0/git-overlay_Linux_x86_64.tar.gz"
        sha256 "2ae1eaff79a8537a89a84128a29dd590ed0bc24874b75a49808ae83317fb0302"

        def install
          bin.install "git-overlay"
        end
      end
    end
    on_arm do
      if Hardware::CPU.is_64_bit?
        url "https://github.com/rjocoleman/git-overlay/releases/download/v0.1.0/git-overlay_Linux_arm64.tar.gz"
        sha256 "abe7078ecc79a3fdf55ed16feaf01013ed94a59e52b0716ce52cf3d87eee46b7"

        def install
          bin.install "git-overlay"
        end
      end
    end
  end

  test do
    system "#{bin}/git-overlay", "--version"
  end
end

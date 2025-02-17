# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class GitOverlay < Formula
  desc "Git overlay tool for managing upstream repositories"
  homepage "https://github.com/rjocoleman/git-overlay"
  version "0.2.0"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/rjocoleman/git-overlay/releases/download/v0.2.0/git-overlay_Darwin_x86_64.tar.gz"
      sha256 "ec8f7b1c64809b26b85a0a22dceec7c81577ed414345dae5debd1f48675717ad"

      def install
        bin.install "git-overlay"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/rjocoleman/git-overlay/releases/download/v0.2.0/git-overlay_Darwin_arm64.tar.gz"
      sha256 "102a4276440cddadbee565dd027f426bcd3b63300e42c0466e3a1704bf64e985"

      def install
        bin.install "git-overlay"
      end
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/rjocoleman/git-overlay/releases/download/v0.2.0/git-overlay_Linux_x86_64.tar.gz"
        sha256 "7eec21f47b2e7d834132eab9c37f3f892d5395e4f5602bf2d92dfdff2563d9e8"

        def install
          bin.install "git-overlay"
        end
      end
    end
    if Hardware::CPU.arm?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/rjocoleman/git-overlay/releases/download/v0.2.0/git-overlay_Linux_arm64.tar.gz"
        sha256 "7d921e80179cbebac655b1665ac4feb52520f8219eca4332bc1c31dbe962e1ca"

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

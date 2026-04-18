class GoPackageYourself < Formula
  desc "A Go packager for NPM, Homebrew, Chocolatey and Docker"
  homepage "https://github.com/Tillman32/go-package-yourself"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Tillman32/go-package-yourself/releases/download/{{version}}/go-package-yourself__darwin_arm64.tar.gz"
      sha256 "91fe0ce36351e19992af588c120ef483f738444a57543644c676e1d5a8d775b5"
    else
      url "https://github.com/Tillman32/go-package-yourself/releases/download/{{version}}/go-package-yourself__darwin_amd64.tar.gz"
      sha256 "19c53b1f218758820e5aa66b8fd47300989fdcb195a033a56e4a84d2abd18732"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/Tillman32/go-package-yourself/releases/download/{{version}}/go-package-yourself__linux_arm64.tar.gz"
      sha256 "9a8bcd096d8f623916fe445d1c2e77ecfbd48b9c59ffd0c4f253d560c5eda709"
    else
      url "https://github.com/Tillman32/go-package-yourself/releases/download/{{version}}/go-package-yourself__linux_amd64.tar.gz"
      sha256 "bd8060948247b2a88fbb3b74376c8e9cda8ba68c0416d00d4d3f1660631e114d"
    end
  end

  def install
    bin.install "go-package-yourself"
  end

  test do
    assert_match /help/, shell_output("#{bin}/go-package-yourself --help")
  end
end

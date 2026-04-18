# go-package-yourself

This is an npm wrapper package for [go-package-yourself](https://github.com/Tillman32/go-package-yourself).

## Installation

Install globally via npm:

```bash
npm install -g go-package-yourself
```

Or install locally in your project:

```bash
npm install go-package-yourself
```

## Usage

Run the binary directly:

```bash
go-package-yourself [options] [arguments]
```

For help and available options, run:

```bash
go-package-yourself --help
```

## How It Works

This npm package is a wrapper that:

1. Detects your operating system and CPU architecture
2. Downloads the precompiled binary from GitHub releases
3. Verifies the download with SHA256
4. Caches the binary locally for future runs
5. Executes the binary with your arguments

## Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Node.js Requirement

Node.js >=18 or higher is required.

## License



## Repository

https://github.com/Tillman32/go-package-yourself

## Support

For issues, feature requests, or bug reports, please visit the [project repository](https://github.com/Tillman32/go-package-yourself).

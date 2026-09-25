# Code Card Generator

Utility to generate a credit-card sized grid of random letter number associations.

The idea is that if you need help to memorize PIN numbers, then you choose a secret word,
and use the numbers on the card to pick your PIN number. When you need a new PIN number,
just print a new card and set the new PIN to the numbers mapped by your secret word.

An encoding function exists for shared PIN numbers: specify both the number and secret
word to generate a randomized card which correctly encodes the PIN.

## Building

```sh
go run mage.go binary
./code-card --help
```

`go run mage.go binary` builds a self-contained static binary into `bin/` and symlinks it into
the repository root. `go run mage.go releaseAll` cross-compiles release archives for every
platform into `release/`.

## Usage

Generate a random card:

```sh
./code-card --title "My Card" random
```

Encode an existing PIN against a keyword. The keyword and code are prompted for (hidden, with
confirmation) if not supplied:

```sh
./code-card --title "Shared PIN" encoded
```

When a letter repeats in the keyword, each letter has several digits: the first use of the
letter reads the first digit, the second use the second, and so on. Keywords may repeat a
letter at most 3 times.

Encoded cards are deterministic: the same salt, keyword and code always produce the same card,
so printing multiple cards for the same PIN doesn't leak information. Tables are identical to
those produced by the original Python implementation. Use `--salt` (at least 16 bytes) to set a
personal salt instead of the default.

Cards are written to `code-card.pdf` by default. Use `--output-file` to change it, or `-` to
write to stdout.

### Options

| Flag | Environment variable | Default | Description |
|---|---|---|---|
| `--title` | `CODE_CARD_TITLE` | | Title for the card |
| `--output-file` | `CODE_CARD_OUTPUT_FILE` | `code-card.pdf` | Output filename (`-` for stdout) |
| `encoded --salt` | `CODE_CARD_ENCODED_SALT` | `1337code8Ackz0rs` | Salt value for encoded cards |
| `encoded --keyword` | `CODE_CARD_ENCODED_KEYWORD` | prompted | Keyword which will produce the code |
| `encoded --code` | `CODE_CARD_ENCODED_CODE` | prompted | Numeric PIN which will be encoded |

## Static site

[`staticsite/`](staticsite/) is a standalone web page implementing the same algorithm, which
generates and prints cards in the browser. See [staticsite/README.md](staticsite/README.md).

Download it and run use it locally - don't run it off a random web site.

## Implementation

| Path | Purpose |
|---|---|
| `cmd/code-card/main.go` | Minimal `main` package. It only calls the entrypoint. |
| `pkg/entrypoints/code_card/` | Command line parsing, logging and the `random` and `encoded` commands. |
| `pkg/codecard/` | Table generation and PDF rendering. |
| `pkg/pyrandom/` | CPython-compatible Mersenne Twister, used so encoded tables match the Python implementation. |
| `staticsite/` | Static web page version of the generator. |
| `version/` | Application name, description and version. `Version` is set at link time by the build. |
| `magefile.go`, `mage.go` | The build system. Run `go run mage.go -l` to list targets. |
| `Dockerfile` | Container image. Cards are written to `/data`. |
| `.github/workflows/` | CI: build and test on every push, releases on `v*` tags. |

Random cards use the system's secure random source. PDFs are rendered with
[fpdf](https://codeberg.org/go-pdf/fpdf) using the built-in Courier font, so no CGO or font
files are needed.

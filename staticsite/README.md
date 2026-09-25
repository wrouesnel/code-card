# Code Card static site

A static web page implementing the same algorithm as the `code-card` binary. It generates and
prints credit-card sized (56mm × 85.6mm) code cards entirely in the browser. It has no build step
and no server-side component, and nothing entered on the page is sent anywhere.

Open `index.html` directly (`file://` works) or serve this folder from any static web host.

| File | Purpose |
|---|---|
| `index.html` | The page: form, SVG card rendering and print styles. |
| `codecard.js` | The algorithm: random and encoded tables, a CPython-compatible Mersenne Twister, SHA-256 and SHA-512. |
| `codecard_test.go` | Runs `codecard.js` in an embedded JS engine and checks it against the Go implementation. |

## URL parameters

All parameters are optional.

| Parameter | Description |
|---|---|
| `title` | Title for the card |
| `encoded_salt` | Salt value for encoded cards (default `1337code8Ackz0rs`) |
| `mode` | `encoded` to start in encoded mode, otherwise random |
| `count` | Number of random cards to generate, from 1 to 8 (default 1) |

For example `index.html?title=My%20Card&mode=encoded&encoded_salt=my-personal-salt-value`.

The keyword and code are only accepted from the page's password fields, never from the URL, so
they don't end up in browser history or web server logs.

## Printing

Print at 100% scale ("actual size") with headers and footers turned off. Cards print on an A4
landscape sheet, laid out side by side in a grid centered inside a 10mm margin from the page
edges. Each card has the same dimensions and layout as the PDF from `code-card`.

Random mode can generate several cards at once: at most 8 (4 × 2) fit inside the margin without
being cut off.

## Implementation

Encoded tables are identical to those from `code-card encoded` for the same salt, keyword and
code. `codecard.js` includes its own SHA-2 implementations instead of using `crypto.subtle`,
which browsers only provide on HTTPS or localhost pages, so encoded cards also work over plain
HTTP and from `file://`. Random cards use `crypto.getRandomValues`.

The tests run as part of `go run mage.go test`, and compare the JavaScript against Go's `crypto`
packages, `pkg/pyrandom`, `pkg/codecard` and the original Python implementation's output.

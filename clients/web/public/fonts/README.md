# Fonts

Geist Sans and Geist Mono, self-hosted. Both are variable fonts carrying the
full 100–900 weight axis in a single file, so every weight the UI uses costs
nothing beyond these two requests.

| File                    | Source                          | Size    |
| ----------------------- | ------------------------------- | ------- |
| `geist-sans-latin.woff2` | `geist@1.7.2` `Geist-Variable`     | ~36 KB |
| `geist-mono-latin.woff2` | `geist@1.7.2` `GeistMono-Variable` | ~37 KB |

Licensed under the SIL Open Font License 1.1 — see `OFL.txt`, which the license
requires to travel with the files.

## Why self-hosted rather than a CDN

The design system's original rule was "no third-party font requests," which
kept self-hosted StageFlow deployments private and offline-capable. Self-hosting
keeps that property intact. The reason for adopting a webfont at all is that the
previous stack named `'Avenir Next'` for display and `'Segoe UI Variable Text'`
for body: on anything that is not macOS or Windows, both fell through to the
same `system-ui` face and headings differed from body text by weight alone. Most
visitors saw a flatter site than the author did.

## Regenerating

The committed files are subsets. Reproduce them with `fonttools`:

```sh
pip install fonttools brotli
npm pack geist@1.7.2 && tar xzf geist-1.7.2.tgz

UNICODES='U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+2074,U+20AC,U+2122,U+2190-21FF,U+2212,U+2215,U+2264-2265,U+2713-2714,U+25A0-25FF,U+2605,U+FEFF,U+FFFD'

pyftsubset package/dist/fonts/geist-sans/Geist-Variable.ttf \
  --output-file=geist-sans-latin.woff2 --flavor=woff2 \
  --unicodes="$UNICODES" --layout-features='*' --name-IDs='*'

pyftsubset package/dist/fonts/geist-mono/GeistMono-Variable.ttf \
  --output-file=geist-mono-latin.woff2 --flavor=woff2 \
  --unicodes="$UNICODES" --layout-features='*' --name-IDs='*'
```

`--layout-features='*'` is not optional. The default subset profile drops
non-default OpenType features, and this UI sets `font-variant-numeric:
tabular-nums` on every number that sits in a column — dropping `tnum` would
leave those columns ragged with no error anywhere.

The unicode range is the standard Google Fonts latin block widened to cover the
arrows, geometric shapes, and math symbols the UI actually renders. `▸ ▾ ◉ ✓ ✕`
are absent from Geist itself, not from the subset; they fall through to
`system-ui` by design, which is why the stack keeps a fallback after Geist.

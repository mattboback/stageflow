# Documentation image regeneration

The recruiter-facing product images are generated from committed, local inputs.
Neither workflow contacts the hosted StageFlow service or a scanned website.

## Review workspace

The capture script builds the web client, starts a loopback-only preview,
mocks the API from the Unified Report v2 fixture, and creates its page evidence
from inline deterministic HTML:

```bash
node clients/web/qa/capture-report-review.mjs docs/images/report-review.png
```

Pass `--no-build` while iterating against an existing `clients/web/build`.
The script requires Bun and Playwright Chromium. Its output is the static PNG
used by the README.

## Social sharing card

The social card is rasterized from the versioned SVG template with ImageMagick:

```bash
magick -background none \
  clients/web/qa/social-card.svg \
  clients/web/public/social/stageflow-og.png
```

Keep the output at `1200 × 630` so Open Graph previews retain their expected
aspect ratio. The architecture diagram is already a source SVG and needs no
generation step.

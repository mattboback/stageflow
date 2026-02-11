# StageFlow Architecture

## Table of Contents

1. [System Overview](#system-overview)
2. [Component Architecture](#component-architecture)
3. [Data Flow](#data-flow)
4. [Job Lifecycle](#job-lifecycle)
5. [Scanner Plugin System](#scanner-plugin-system)
6. [AI Navigator Scanner](#ai-navigator-scanner)
7. [Provenance System](#provenance-system)
8. [Pre-Scan Actions](#pre-scan-actions)
9. [Real-Time Progress (SSE)](#real-time-progress-sse)
10. [Event-Driven Architecture](#event-driven-architecture)
11. [ZIP Extractor](#zip-extractor)
12. [Stage Logging](#stage-logging)
13. [Database Schema](#database-schema)
14. [Report Schema](#report-schema)
15. [Security Architecture](#security-architecture)
16. [Infrastructure](#infrastructure)

---

## System Overview

StageFlow is a polyglot microservices platform built with:

- **Go 1.25** - Backend services (API, Orchestrator, Extractor)
- **TypeScript/Bun** - Scanner runtime with Playwright
- **SvelteKit 5** - Frontend SPA
- **NATS JetStream** - Event messaging
- **MinIO** - S3-compatible object storage
- **Podman** - Container orchestration
- **PostgreSQL** - Durable relational persistence

### High-Level Architecture

```
+-----------------------------------------------------------------------+
|                              Caddy                                     |
|                         (Reverse Proxy)                                |
|   - Auto HTTPS via Let's Encrypt                                       |
|   - Routes: /api/* -> Platform API, /monitoring/* -> Grafana, /* -> SPA |
+---------------------------+-------------------------------------------+
                            |
            +---------------+---------------+
            |                               |
            v                               v
+---------------------+
|   Frontend          |         |   MinIO             |
|   (SvelteKit SPA)   |         |   (presigned URLs)  |
|   :3000             |         |   /scanner-*        |
+----------+----------+         +---------------------+
           |
           | API requests (/api/*)
           v
+---------------------+         +---------------------+
|   Platform API      |<------->|   NATS JetStream    |
|   (Go)              |         |   :4222             |
|   :8080             |         |                     |
+----------+----------+         +----------+----------+
           |                               ^
           | (job.created)                 | (events)
           v                               |
+---------------------+                    |
|   Orchestrator      |<-------------------+
|   (Go)              |
|   :8081             |
+----------+----------+
           |
           | (spawns containers)
           v
+-----------------------------------------------------------------------+
|                           Podman                                       |
|   +-------------------+   +-------------------+   +-------------------+ |
|   |    Extractor      |   |  Scanner Runner   |   |  Scanner Runner   | |
|   |    (Go)           |   |  (axe)            |   |  (lighthouse)     | |
|   |    Ephemeral      |   |  TypeScript/Bun   |   |  TypeScript/Bun   | |
|   +-------------------+   +-------------------+   +-------------------+ |
|                                                                        |
|   Volumes: workspace-{jobID}, results-{jobID}                         |
+-----------------------------------------------------------------------+
```

---

## Component Architecture

### Platform API (`platform/api`)

The REST API service handling job submission and status queries.

```
platform/api/
+-- cmd/server/
|   +-- main.go              # Entry point
|   +-- config.go            # Environment configuration
+-- internal/
    +-- api/
    |   +-- router.go        # HTTP routing and middleware chain
    |   +-- middleware.go    # Auth, CORS, logging, timeout
    |   +-- security.go      # SSRF protection for URL validation
    |   +-- handlers_jobs_*.go  # Job handlers (zip, url, status)
    |   +-- handlers_sse.go  # Server-Sent Events
    |   +-- handlers_scanners.go
    +-- messaging/
    |   +-- service.go       # NATS event publishing
    +-- statussource/
        +-- client.go        # Orchestrator status source client
```

**Key Responsibilities:**
- Accept scan requests (ZIP upload or URL list)
- Validate input (SSRF protection, size limits)
- Upload ZIPs to MinIO staging bucket
- Publish `job.created` events to NATS
- Serve job status via REST and SSE
- Generate presigned URLs for artifacts

**Middleware Stack:**
```
Request -> loggingMiddleware -> corsMiddleware -> apiKeyMiddleware -> timeoutMiddleware -> Handler
```

### Orchestrator (`platform/orchestrator`)

The job coordination service managing the complete scan lifecycle.

```
platform/orchestrator/
+-- cmd/orchestrator/
|   +-- main.go              # Entry point
|   +-- config.go            # Configuration
+-- internal/
    +-- orchestrator/
    |   +-- orchestrator.go  # Core orchestrator struct
    |   +-- events.go        # NATS event handlers
    |   +-- extraction.go    # ZIP job extraction phase
    |   +-- url_jobs.go      # URL job handling
    |   +-- scanning.go      # Scanner container spawning
    |   +-- deadline.go      # Timeout watchdogs
    |   +-- job_cleanup.go   # Resource cleanup
    |   +-- report_aggregator*.go  # Result aggregation
    |   +-- rule_deduplication.go  # Cross-scanner dedup
    +-- api/
    |   +-- server.go        # Internal admin API
    +-- messaging/
    |   +-- consumers.go     # NATS consumer setup
    +-- db/
    |   +-- database.go      # Jobs PostgreSQL store
    |   +-- job_events.go    # Event audit log
    |   +-- schema.sql       # Orchestrator DB schema
    +-- fsm/
        +-- state.go         # State machine validation
```

**Key Responsibilities:**
- Consume `job.created` events
- Create Podman pods with appropriate networking
- Spawn extractor containers (for ZIP jobs)
- Spawn scanner containers (axe, lighthouse, etc.)
- Monitor container exit codes
- Handle scanner completion/failure events
- Aggregate results into unified reports
- Clean up pods and volumes

### Scanner Runner (`platform/scanner-runner`)

The TypeScript/Bun runtime executing accessibility scans.

```
platform/scanner-runner/
+-- src/
|   +-- worker.ts            # Main entry point
|   +-- core/
|   |   +-- scanner-base.ts  # Base scanner class with lifecycle
|   |   +-- types.ts         # Core type definitions
|   |   +-- config-loader.ts # Configuration loader
|   |   +-- browser-manager.ts
|   |   +-- page-iterator.ts
|   |   +-- event-publisher.ts
|   |   +-- storage-provider.ts
|   |   +-- plugins/
|   |       +-- plugin-loader.ts
|   |       +-- plugin-discovery.ts
|   |       +-- plugin-load.ts
|   +-- scanners/
|   |   +-- axe/             # Axe-core accessibility
|   |   +-- lighthouse/      # Google Lighthouse
|   |   +-- seo/             # SEO best practices
|   |   +-- security-headers/
|   |   +-- link-checker/
|   |   +-- ai-navigator/    # LLM-powered navigation
|   +-- screenshots/
+-- tests/
```

**Key Responsibilities:**
- Load scanner plugin by manifest
- Initialize Playwright browser
- Iterate pages from provenance.json
- Execute scanner-specific logic
- Capture screenshots on violations
- Upload results to MinIO
- Publish completion events to NATS

### Frontend (`frontend`)

The SvelteKit single-page application.

```
frontend/
+-- src/
|   +-- routes/
|   |   +-- +page.svelte     # Home
|   |   +-- +layout.svelte   # Root layout
|   |   +-- playground/      # Scanner playground
|   |   +-- scan/[id]/       # Scan status/results
|   +-- lib/
|   |   +-- components/
|   |   |   +-- ui/          # Design system primitives
|   |   |   +-- playground/  # Scanner UI components
|   |   |   +-- report/      # Report rendering
|   |   |   +-- scan-status/ # Real-time status
|   |   +-- stores/
|   |   |   +-- scan-status.svelte.ts
|   |   |   +-- scan-report.svelte.ts
|   |   +-- api/
|   |   |   +-- client.ts    # API client
|   |   +-- types/
|   +-- app.css              # Tailwind v4 + design tokens
```

**Key Patterns:**
- Svelte 5 runes (`$state`, `$derived`, `$effect`)
- Factory-based stores for per-instance state
- SSE for real-time updates with polling fallback
- CVA (class-variance-authority) for component variants

---

## Data Flow

### ZIP Scan Flow

```
+----------+     +----------+     +----------+     +------------+
|  Client  |---->| Frontend |---->|   API    |---->|   MinIO    |
| (upload) |     |  :3000   |     |  :8080   |     | (staging)  |
+----------+     +----------+     +----+-----+     +------------+
                                       |
                                       | job.created (NATS)
                                       v
+------------+     +------------+     +------------+
|   MinIO    |<----| Extractor  |<----|Orchestrator|
| (download) |     | Container  |     |   :8081    |
+------------+     +------+-----+     +------------+
                          |
                          | extraction.ready (NATS)
                          v
+------------+     +------------+     +------------+
|  Scanner   |---->|   MinIO    |---->|Orchestrator|
| Containers |     | (results)  |     | (aggregate)|
+------------+     +------------+     +------+-----+
                                             |
                                             | job.completed (NATS)
                                             v
+----------+     +----------+     +------------+
|  Client  |<----| Frontend |<----|    API     |
| (results)|     | (SSE)    |     | (status)   |
+----------+     +----------+     +------------+
```

### URL Scan Flow

```
+----------+     +----------+     +----------+
|  Client  |---->| Frontend |---->|   API    |
| (URLs)   |     |  :3000   |     |  :8080   |
+----------+     +----------+     +----+-----+
                                       |
                                       | job.created (NATS)
                                       v
                              +------------+
                              |Orchestrator|
                              |   :8081    |
                              +------+-----+
                                     |
                    +----------------+----------------+
                    |                                 |
                    v                                 v
           +------------+                    +------------+
           |  Scanner   |                    |  Scanner   |
           |   (axe)    |                    |(lighthouse)|
           +------+-----+                    +------+-----+
                  |                                 |
                  | scan.completed                  | scan.completed
                  +----------------+----------------+
                                   |
                                   v
                           +------------+
                           |Orchestrator|
                           | (aggregate)|
                           +------+-----+
                                  |
                                  | job.completed
                                  v
                           +------------+
                           |   Client   |
                           |  (SSE)     |
                           +------------+
```

---

## Job Lifecycle

### State Machine

```
                                  +--------+
                                  |PENDING |
                                  +---+----+
                                      |
                    +-----------------+-----------------+
                    | (ZIP job)                         | (URL job)
                    v                                   |
              +------------+                            |
              |EXTRACTING  |                            |
              +-----+------+                            |
                    |                                   |
                    | extraction.ready                  |
                    v                                   v
              +------------+<---------------------------+
              |READY_TO_   |
              |SCAN        |
              +-----+------+
                    |
                    | start_scanning
                    v
              +------------+
              | SCANNING   |
              +-----+------+
                    |
                    | all scanners complete
                    v
              +------------+
              |COMPLETING  |
              +-----+------+
                    |
                    | report uploaded
                    v
              +------------+
              |   DONE     |
              +------------+

     (any state) ----error----> FAILED
```

### State Definitions

| State           | Description                                          |
|-----------------|------------------------------------------------------|
| `PENDING`       | Job created, waiting for processing                  |
| `EXTRACTING`    | ZIP being extracted to workspace volume              |
| `READY_TO_SCAN` | Workspace ready, scanners can start                  |
| `SCANNING`      | Scanner containers actively running                  |
| `COMPLETING`    | All scanners done, aggregating final report          |
| `DONE`          | Job completed successfully                           |
| `FAILED`        | Job failed at any stage                              |

### Timeouts

| Phase      | Default Timeout | Watchdog Poll |
|------------|-----------------|---------------|
| Extraction | 5 minutes       | 30 seconds    |
| Scanning   | 30 minutes      | 30 seconds    |

---

## Scanner Plugin System

StageFlow's scanner architecture is designed for extensibility. Scanners are self-contained plugins discovered at runtime via manifest files. The system handles browser management, page iteration, event publishing, and artifact storage - scanners only implement the scanning logic.

### Plugin Discovery

Scanners are discovered via manifest files at startup:

```
Search Paths (in order):
1. Built-in: platform/scanner-runner/src/scanners/
2. Volume-mounted: /plugins (Docker/Podman)
3. User plugins: ~/.stageflow/plugins
4. Environment: PLUGIN_PATHS (colon-separated)

Discovery Process:
+-- Search directories for manifest.json or scanner.json
+-- Parse and validate manifest against JSON Schema
+-- Build lookup maps (by ID and aliases)
+-- Return PluginInfo[] with paths and manifests
```

### Manifest Schema (Complete Reference)

**Source**: `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json`

```json
{
  "id": "my-scanner",                    // Required: lowercase alphanumeric + hyphens
  "name": "My Scanner",                  // Required: human-readable name (max 128 chars)
  "version": "1.0.0",                    // Required: semver
  "description": "What this scanner does",
  "author": "Your Name",
  "license": "MIT",
  "homepage": "https://example.com",
  "repository": "https://github.com/...",
  "aliases": ["alt-name", "shortcut"],   // Alternative identifiers for lookup

  "capabilities": {
    "categories": ["accessibility", "performance", "security", "seo", "quality", "custom"],
    "outputFormats": ["json", "html", "csv"],
    "supportsScreenshots": true,         // Requires requiresBrowser: true
    "supportsConcurrency": true,         // Can process multiple pages in parallel
    "requiresBrowser": true,             // Needs Playwright browser
    "supportsOffline": false,            // Can run without network
    "maxConcurrency": 5,                 // Max parallel pages (if supportsConcurrency)
    "estimatedTimePerPage": 3000         // Milliseconds, for progress estimation
  },

  "entry": {
    "module": "./index.js",              // Required: path relative to manifest
    "exportName": "MyScanner",           // Named export class (optional)
    "factoryName": "createScanner"       // Factory function name (optional)
  },

  "configSchema": {                      // JSON Schema for SCANNER_OPTIONS validation
    "type": "object",
    "properties": {
      "timeout": { "type": "number", "minimum": 1000 },
      "rules": { "type": "array", "items": { "type": "string" } }
    }
  },

  "requirements": {
    "browser": {
      "type": "chromium",                // chromium | firefox | webkit
      "headless": true,
      "args": ["--no-sandbox"]
    },
    "nodeVersion": ">=18",               // Semver range
    "externalTools": [{
      "name": "curl",
      "command": "curl",
      "version": "8.0.0",
      "optional": true
    }],
    "network": {
      "requiresInternet": true,
      "allowedHosts": ["api.example.com"]
    },
    "resources": {
      "maxMemoryMB": 1024,
      "maxTimeoutMs": 60000
    }
  },

  "output": {
    "severityMapping": {                 // Map custom severities to standard
      "error": "critical",
      "warning": "moderate",
      "info": "minor"
    },
    "categoryPrefix": "my-scanner",
    "includeRawResults": true
  }
}
```

### Creating a Custom Scanner

**Step 1: Implement the Scanner Class**

All scanners extend `ScannerBase` and implement `scanPage()`:

```typescript
// my-scanner/index.ts
import { ScannerBase, type ScanContext, type PageScanResult } from "@stageflow/scanner-runner";

export class MyScanner extends ScannerBase {
  readonly metadata = {
    name: "my-scanner",        // Must match manifest.id
    version: "1.0.0",
    description: "My custom scanner"
  };

  async scanPage(context: ScanContext): Promise<PageScanResult> {
    const { page, pageEntry, logger } = context;
    const startTime = Date.now();

    try {
      // Your scanning logic
      const issues = await this.runChecks(page);

      return {
        pageId: pageEntry.id,
        url: pageEntry.url,
        path: pageEntry.path,
        success: true,
        issues,
        durationMs: Date.now() - startTime,
        startedAt: new Date(startTime).toISOString(),
        finishedAt: new Date().toISOString()
      };
    } catch (error) {
      return {
        pageId: pageEntry.id,
        url: pageEntry.url,
        success: false,
        issues: [],
        durationMs: Date.now() - startTime,
        startedAt: new Date(startTime).toISOString(),
        finishedAt: new Date().toISOString(),
        error: error instanceof Error ? error.message : String(error)
      };
    }
  }

  private async runChecks(page: Page): Promise<Issue[]> {
    // Implement your checks
    return [];
  }
}
```

**Step 2: Create the Manifest**

```json
{
  "id": "my-scanner",
  "name": "My Scanner",
  "version": "1.0.0",
  "capabilities": {
    "categories": ["custom"],
    "outputFormats": ["json"],
    "supportsScreenshots": false,
    "supportsConcurrency": true,
    "requiresBrowser": true
  },
  "entry": {
    "module": "./index.js",
    "exportName": "MyScanner"
  }
}
```

**Step 3: Deploy**

Place your scanner in one of the search paths:
- Development: `~/.stageflow/plugins/my-scanner/`
- Docker: Mount to `/plugins/my-scanner/`
- Custom: Add path to `PLUGIN_PATHS` env var

### Entry Point Strategies

The plugin loader supports three ways to export your scanner:

**1. Factory Function (most flexible)**
```typescript
// manifest: { "entry": { "module": "./index.js", "factoryName": "createScanner" } }
export function createScanner() {
  return new MyScanner({ customOption: true });
}
```

**2. Named Export Class**
```typescript
// manifest: { "entry": { "module": "./index.js", "exportName": "MyScanner" } }
export class MyScanner extends ScannerBase { ... }
```

**3. Default Export (simplest)**
```typescript
// manifest: { "entry": { "module": "./index.js" } }
export default class MyScanner extends ScannerBase { ... }
```

### Scanner Lifecycle

```
+-----------------+
| 1. Initialize   |
|  - Create dirs  |
|  - Init browser |
|  - Init storage |
|  - Connect NATS |
+-----------------+
        |
        v
+-----------------+
| 2. Load         |
|  Provenance     |
|  - Validate     |
|  - Parse pages  |
+-----------------+
        |
        v
+-----------------+
| 3. Iterate      |
|  Pages          |
|  for each page: |
|  - Navigate     |
|  - Pre-actions  |
|  - scanPage()   |  <-- Your code runs here
|  - Publish evt  |
+-----------------+
        |
        v
+-----------------+
| 4. Build        |
|  Results        |
|  - Aggregate    |
|  - Summarize    |
+-----------------+
        |
        v
+-----------------+
| 5. Upload       |
|  Artifacts      |
|  - results.json |
|  - screenshots/ |
+-----------------+
        |
        v
+-----------------+
| 6. Publish      |
|  Completion     |
|  - scan.done    |
|  - or scan.fail |
+-----------------+
        |
        v
+-----------------+
| 7. Cleanup      |
|  - Close browser|
|  - Drain NATS   |
+-----------------+
```

### Lifecycle Hooks

Scanners can register callbacks for lifecycle events:

```typescript
interface ScannerLifecycleHooks {
  onScanStart?(config: ScannerConfig): Promise<void>;
  onScanEnd?(results: ScanResults): Promise<void>;
  onPageStart?(pageEntry: PageEntry): Promise<void>;
  onPageEnd?(result: PageScanResult): Promise<void>;
  onError?(error: Error, context?: { pageEntry?: PageEntry }): Promise<void>;
}

// Pass hooks to scanner constructor
const scanner = new MyScanner({
  onScanStart: async (config) => {
    console.log("Starting scan for job:", config.jobId);
  },
  onPageEnd: async (result) => {
    console.log(`Page ${result.url}: ${result.issues.length} issues`);
  }
});
```

### Overridable Methods

Customize scanner behavior by overriding protected methods:

```typescript
class MyScanner extends ScannerBase {
  // Called before scanning begins
  protected override async initialize(): Promise<void> {
    await super.initialize();
    // Custom setup
  }

  // Validate the provenance data
  protected override validateProvenance(provenance: Provenance): void {
    // Custom validation
  }

  // Customize result aggregation
  protected override buildResults(provenance: Provenance, pageResults: PageScanResult[]): ScanResults {
    // Custom summarization
  }

  // Custom artifact upload logic
  protected override async uploadArtifacts(): Promise<void> {
    // Upload additional files
    await super.uploadArtifacts();
  }

  // Cleanup after scan
  protected override async cleanup(): Promise<void> {
    // Release custom resources
    await super.cleanup();
  }
}
```

### Built-in Scanners

| Scanner           | File                                  | Concurrency | Browser |
|-------------------|---------------------------------------|-------------|---------|
| `axe`             | `scanners/axe/index.ts`               | Up to 5     | Yes     |
| `lighthouse`      | `scanners/lighthouse/index.ts`        | Serial      | Yes (dedicated) |
| `seo`             | `scanners/seo/index.ts`               | Up to 10    | Yes     |
| `security-headers`| `scanners/security-headers/index.ts`  | Up to 10    | Yes     |
| `link-checker`    | `scanners/link-checker/index.ts`      | Up to 5     | Yes     |
| `ai-navigator`    | `scanners/ai-navigator/index.ts`      | Serial      | Yes     |

---

## AI Navigator Scanner

The AI Navigator is an experimental scanner that uses vision-capable LLMs to autonomously navigate websites and complete goals. Instead of checking static rules, it attempts user flows and reports success or failure.

### How It Works

```
+------------------+     +------------------+     +------------------+
|  1. Screenshot   |---->|  2. Analyze Page |---->|  3. Decide Action|
|     current page |     |     via Vision   |     |     via LLM      |
+------------------+     +------------------+     +--------+---------+
                                                          |
         +------------------------------------------------+
         |
         v
+------------------+     +------------------+     +------------------+
|  4. Execute      |---->|  5. Check Goal   |---->|  6. Loop or Done |
|     Action       |     |     Achieved?    |     |                  |
+------------------+     +------------------+     +------------------+
```

### Two-Stage LLM Decision Process

**Stage 1: Page Analysis** (`page-analyzer.ts`)
- Screenshots the current page
- Extracts interactive elements (links, buttons, inputs, selects)
- Asks the LLM to describe the page type and suggest relevant actions
- Returns structured perception with element indices

**Stage 2: Action Decision** (`action-decider.ts`)
- Takes the page perception and goal context
- Reviews recent step history to avoid loops
- Asks the LLM for the next action with confidence score
- Parses response into executable action

### Supported Actions

| Action Type | Description |
|-------------|-------------|
| `click`     | Click an element by selector |
| `fill`      | Enter text into an input field |
| `select`    | Choose option from dropdown |
| `hover`     | Hover over an element |
| `scroll`    | Scroll up or down by pixels |
| `keyboard`  | Press a key (Enter, Escape, etc.) |
| `wait`      | Wait for milliseconds |
| `done`      | Goal achieved, terminate successfully |
| `stuck`     | Cannot proceed, terminate with failure |

### Configuration

```json
{
  "goal": {
    "objective": "Add a product to the shopping cart",
    "successCriteria": [
      { "type": "element-visible", "value": "[data-testid='cart-count']" },
      { "type": "url-contains", "value": "/cart" }
    ],
    "maxSteps": 15,
    "maxWallTimeMs": 180000,
    "inputValues": {
      "email": "test@example.com",
      "quantity": "2"
    }
  },
  "vision": {
    "model": "openrouter/openai/gpt-4-vision-preview",
    "maxTokens": 1024,
    "timeoutMs": 45000,
    "retry": { "maxAttempts": 3, "baseDelayMs": 1000 }
  }
}
```

### Success Criteria Types

| Type | Description |
|------|-------------|
| `url-contains` | Final URL contains substring |
| `url-matches` | Final URL matches regex |
| `element-visible` | Element matching selector is visible |
| `text-visible` | Text content is visible on page |
| `custom` | Reserved for future use |

### Safety Features

- **Loop Detection**: If the same URL appears 3+ times in the last 6 steps, the agent terminates as stuck
- **Wall Time Budget**: Hard timeout prevents runaway execution
- **Step Budget**: Maximum steps prevents infinite loops
- **Image Compression**: Screenshots compressed to stay under token limits
- **Concurrency Control**: Semaphore limits parallel LLM requests

### Output Artifacts

The AI Navigator produces:

1. **ai-trace.json** - Complete step-by-step trace:
```json
{
  "success": false,
  "goal": { "objective": "Complete checkout" },
  "startUrl": "https://example.com",
  "finalUrl": "https://example.com/cart",
  "totalSteps": 8,
  "totalDurationMs": 45230,
  "stuckReason": "Payment form not found",
  "steps": [
    {
      "stepNumber": 1,
      "url": "https://example.com",
      "action": { "type": "click", "selector": "#add-to-cart" },
      "reasoning": "Add item to cart to begin checkout flow",
      "success": true,
      "screenshotPath": "screenshots/ai-step-001.png"
    }
  ]
}
```

2. **screenshots/** - Per-step screenshots for debugging

---

## Provenance System

The provenance file defines which pages to scan, how to wait for them, and what actions to perform before scanning. It's the contract between job submission and scanner execution.

### Provenance Schema

```typescript
interface Provenance {
  version: string;                      // Schema version (e.g., "1.0.0")
  job_id: string;                       // Job identifier
  base_url?: string;                    // Base URL for resolving relative paths
  mode?: "static" | "spa" | "live";     // Scanning mode (default: "live")
  default_wait_for?: WaitStrategy;      // Default wait strategy for all pages
  default_viewport?: Viewport;          // Default viewport dimensions
  pages: PageEntry[];                   // Pages to scan
}

interface PageEntry {
  id: string;                           // Unique page identifier
  path?: string;                        // Path relative to base_url
  url?: string;                         // Full URL (overrides base_url + path)
  wait_for?: WaitStrategy;              // Page-specific wait strategy
  pre_scan_actions?: PreScanAction[];   // Actions before scanning
  viewport?: Viewport;                  // Page-specific viewport
  skip?: boolean;                       // Skip this page
  metadata?: Record<string, unknown>;   // Custom metadata
}

interface Viewport {
  width: number;
  height: number;
}
```

### Wait Strategies

Control how scanners determine when a page is "ready":

| Strategy | Use Case |
|----------|----------|
| `{ "type": "load" }` | Default. Waits for `load` event. Good for static sites. |
| `{ "type": "domcontentloaded" }` | Faster. DOM ready but resources may still load. |
| `{ "type": "networkidle" }` | No network requests for 500ms. Caution: may hang on WebSocket sites. |
| `{ "type": "selector", "selector": ".content", "timeout": 30000 }` | Wait for specific element. Best for SPAs. |
| `{ "type": "timeout", "ms": 5000 }` | Fixed delay. Last resort for unpredictable pages. |

### Scanning Modes

| Mode | Description |
|------|-------------|
| `static` | ZIP upload of HTML files. No network requests during scan. |
| `spa` | Single-page application. Expects client-side routing and dynamic content. |
| `live` | Live website. Full network access, real server responses. |

### Dynamic Provenance Generation

For URL-based jobs without a provenance file, one is generated from the `SCAN_URLS` environment variable:

```bash
# Environment variable
SCAN_URLS='["https://example.com", "https://example.com/about"]'

# Generated provenance
{
  "version": "1.0.0",
  "job_id": "abc123",
  "base_url": "https://example.com",
  "pages": [
    { "id": "page-0", "url": "https://example.com" },
    { "id": "page-1", "url": "https://example.com/about" }
  ]
}
```

---

## Pre-Scan Actions

Pre-scan actions allow page manipulation before scanning begins. Use them to dismiss cookie banners, authenticate, fill forms, or set up any required page state.

### Supported Actions

| Action | Parameters | Description |
|--------|------------|-------------|
| `click` | `selector` | Click an element |
| `fill` | `selector`, `value` | Enter text into input/textarea |
| `select` | `selector`, `value` | Choose dropdown option by value |
| `hover` | `selector` | Hover over element |
| `scroll` | `direction`, `pixels` | Scroll up/down by pixels |
| `keyboard` | `key` | Press keyboard key (Enter, Escape, Tab, etc.) |
| `wait` | `ms` | Wait for milliseconds |

### Action Schema

```typescript
type PreScanAction =
  | { type: "click"; selector: string }
  | { type: "fill"; selector: string; value: string }
  | { type: "select"; selector: string; value: string }
  | { type: "hover"; selector: string }
  | { type: "scroll"; direction: "up" | "down"; pixels: number }
  | { type: "keyboard"; key: string }
  | { type: "wait"; ms: number };
```

### Example: Login Flow

```json
{
  "pages": [{
    "id": "admin-dashboard",
    "url": "https://example.com/admin",
    "pre_scan_actions": [
      { "type": "click", "selector": "#cookie-consent-accept" },
      { "type": "fill", "selector": "#email", "value": "admin@example.com" },
      { "type": "fill", "selector": "#password", "value": "secure-password" },
      { "type": "click", "selector": "button[type='submit']" },
      { "type": "wait", "ms": 2000 },
      { "type": "click", "selector": "[data-dismiss='modal']" }
    ],
    "wait_for": { "type": "selector", "selector": ".dashboard-content" }
  }]
}
```

### Execution Order

```
1. Navigate to page URL
2. Wait for initial page load (wait_for strategy)
3. Execute pre_scan_actions sequentially
4. Call scanner's scanPage() method
5. Publish completion event
```

### Error Handling

- Missing selector: Action fails after timeout (default 30s)
- Action failure: Logged but scan continues
- All actions have implicit timeouts based on Playwright defaults

---

## Real-Time Progress (SSE)

The API provides Server-Sent Events for real-time job progress updates, eliminating the need for polling.

### Endpoint

```
GET /api/v1/jobs/{jobID}/stream
Accept: text/event-stream
```

### Event Types

| Event | When Sent | Payload |
|-------|-----------|---------|
| `status` | Immediately on connect | Full job status |
| `update` | On state change | Updated job status |
| `done` | Job terminal state | Final status with report URLs |

### Event Format

```
event: status
data: {"jobId":"abc123","state":"SCANNING","currentPage":3,"totalPages":10}

event: update
data: {"jobId":"abc123","state":"SCANNING","currentPage":4,"totalPages":10}

event: done
data: {"jobId":"abc123","state":"DONE","reportUrl":"/api/v1/jobs/abc123/results"}
```

### Connection Management

- **Heartbeat**: `:keepalive` comment sent every 15 seconds to prevent proxy/firewall timeout
- **Reconnection**: Clients can reconnect and receive current state immediately
- **Terminal Detection**: Server closes stream after sending `done` event
- **Graceful Shutdown**: Connection drains on server shutdown

### Implementation Details

```go
// SSE handler flow
func handleJobStream(w http.ResponseWriter, r *http.Request) {
    // 1. Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // 2. Send initial status immediately
    sendEvent(w, "status", currentJobStatus)

    // 3. Subscribe to job updates via channel
    updates := hub.Subscribe(jobID)
    defer hub.Unsubscribe(jobID, updates)

    // 4. Stream updates until done
    ticker := time.NewTicker(15 * time.Second)
    for {
        select {
        case <-ticker.C:
            fmt.Fprintf(w, ":keepalive\n\n")
        case update := <-updates:
            sendEvent(w, "update", update)
            if isTerminal(update.State) {
                sendEvent(w, "done", update)
                return
            }
        case <-r.Context().Done():
            return
        }
    }
}
```

### Client Example

```javascript
const eventSource = new EventSource(`/api/v1/jobs/${jobId}/stream`);

eventSource.addEventListener('status', (e) => {
  const data = JSON.parse(e.data);
  console.log('Initial status:', data.state);
});

eventSource.addEventListener('update', (e) => {
  const data = JSON.parse(e.data);
  updateProgressBar(data.currentPage, data.totalPages);
});

eventSource.addEventListener('done', (e) => {
  const data = JSON.parse(e.data);
  eventSource.close();
  window.location.href = data.reportUrl;
});

eventSource.onerror = () => {
  // Reconnect automatically handled by browser
  // Or implement manual reconnection with backoff
};
```

---

## Event-Driven Architecture

StageFlow uses NATS JetStream for all inter-service communication. This provides durability, replay capability, and clean decoupling between services.

### Why Event-Driven?

| Benefit | How It Helps |
|---------|--------------|
| **Durability** | Events persist in JetStream. If the orchestrator crashes mid-job, it resumes from the last acknowledged event. |
| **Decoupling** | API publishes `job.created` without knowing who consumes it. New services can subscribe without API changes. |
| **Audit Trail** | Every event is logged to `job_events` table with delivery metadata. Debug failures by replaying the event stream. |
| **Scalability** | Multiple orchestrator instances can share a consumer group. Work distributes automatically. |
| **Resilience** | Failed handlers are retried (up to MaxDeliver). Transient errors don't lose jobs. |

### NATS Streams

| Stream       | Subjects                            | Purpose                    |
|--------------|-------------------------------------|----------------------------|
| `jobs`       | `jobs.events.*`                     | Job lifecycle events       |
| `extraction` | `extraction.events.*`               | ZIP extraction events      |
| `scan`       | `scan.events.*`                     | Scanner events             |

### Event Types

```
jobs.events.created      -> Orchestrator (start job)
jobs.events.completed    -> API (update status)
jobs.events.failed       -> API (update status)

extraction.events.ready  -> Orchestrator (start scanning)
extraction.events.failed -> Orchestrator (fail job)

scan.events.page.completed -> API (progress update)
scan.events.completed      -> Orchestrator (scanner done)
scan.events.failed         -> Orchestrator (scanner failed)
```

### Event Envelope

```typescript
interface EventEnvelope<T> {
  event: string;           // Event type
  job_id: string;          // Job identifier
  request_id?: string;     // Correlation ID
  run_id?: string;         // Execution run ID
  timestamp: string;       // ISO 8601
  producer: string;        // Service name
  payload: T;              // Event-specific data
}
```

### Consumer Configuration

- **AckPolicy**: Explicit (must acknowledge)
- **MaxDeliver**: 3 (retry failed handlers up to 3 times)
- **AckWait**: 30 seconds (redeliver if not acknowledged)
- **Durable**: Named consumers for persistence

---

## ZIP Extractor

The extractor service safely handles ZIP file uploads, validating and extracting them for static site scanning.

### Security Features

The extractor implements multiple layers of protection against malicious archives:

**ZIP Bomb Protection:**
| Limit | Value | Purpose |
|-------|-------|---------|
| Max entries | 5,000 | Prevents million-file bombs |
| Max compression ratio | 100:1 | Detects highly compressed bombs |
| Max uncompressed size | 1 GB | Total extraction limit |
| Max single file | 250 MB | Per-entry limit |
| Max filename length | 4,096 chars | Prevents path-based attacks |

**Path Traversal Prevention:**
```go
// Rejected patterns:
// - Absolute paths: /etc/passwd
// - Windows drives: C:/Windows
// - Parent traversal: ../../../etc/passwd
// - Backslash variants: ..\..\..\etc\passwd
// - NUL bytes in names: file\x00.txt
```

### Extraction Flow

```
1. Download ZIP from MinIO staging bucket
2. Validate archive structure:
   - Check entry count
   - Verify compression ratios
   - Validate filenames
3. Extract to workspace volume:
   - Sanitize each entry path
   - Verify path stays within destination
   - Create directories with 0750 permissions
   - Create files with 0600 permissions
4. Generate provenance.json from directory structure
5. Publish extraction.ready event
6. Delete temporary ZIP file
```

### Supported Archive Structure

```
site.zip
├── index.html           # Required: entry point
├── about/
│   └── index.html
├── products/
│   ├── index.html
│   └── widget.html
├── css/
│   └── styles.css
├── js/
│   └── app.js
└── images/
    └── logo.png
```

The extractor discovers all `.html` files and generates a provenance file with page entries for each.

---

## Stage Logging

Every scan execution is logged with detailed metrics and events. Stage logs provide a complete audit trail for debugging and compliance.

### Stage Log Schema

```typescript
interface ScanStageLog {
  schema: "stageflow.stages.scan.v1";
  job_id: string;
  stage: "scan";
  scanner_type: string;
  attempt: number;                    // Retry attempt number
  status: "succeeded" | "failed";
  started_at: string;                 // ISO 8601
  completed_at: string;
  duration_ms: number;

  recipe_ref: {
    bucket: string;
    path: string;
    sha256: string;                   // Content hash for integrity
  };

  metrics: {
    pages_total?: number;
    pages_scanned?: number;
    total_issues?: number;
  };

  events: StageEvent[];               // Timestamped event log

  artifacts: {
    provenance_key?: string;          // MinIO key
    results_key?: string;
  };

  failure?: {
    stage: string;
    message: string;
    details?: string;
  };
}

interface StageEvent {
  ts: string;                         // ISO 8601 timestamp
  type: string;                       // Event type
  details?: Record<string, unknown>;
}
```

### Event Types

| Event Type | When Logged |
|------------|-------------|
| `stage_started` | Scanner begins execution |
| `provenance_loaded` | Provenance file parsed |
| `page_started` | Page scan begins |
| `page_completed` | Page scan finishes |
| `results_written` | Results saved to disk |
| `artifacts_uploaded` | Files uploaded to MinIO |
| `stage_completed` | Scanner finishes successfully |
| `stage_failed` | Scanner encounters fatal error |

### Recipe Schema

Recipes capture the immutable configuration used for a scan:

```typescript
interface ScanRecipe {
  schema: "stageflow.recipes.scan.v1";
  job_id: string;
  stage: "scan";
  scanner_type: string;
  generated_at: string;

  input: {
    provenance_path: string;
    scan_urls: boolean;               // Was SCAN_URLS env var used?
  };

  results_dir: string;

  environment: {
    nats_url: string;
    minio_endpoint: string;
    minio_use_ssl: boolean;
    artifacts_bucket: string;
  };
}
```

### Storage Location

Stage logs and recipes are uploaded to MinIO:

```
scanner-artifacts/
└── {job_id}/
    ├── scan-stage-log.json      # Execution log
    ├── scan-recipe.json         # Configuration snapshot
    ├── results.json             # Scanner results
    └── screenshots/             # Captured images
```

### Debugging with Stage Logs

```bash
# Download stage log for a failed job
mc cp minio/scanner-artifacts/{job_id}/scan-stage-log.json .

# View the event timeline
jq '.events[] | "\(.ts) \(.type)"' scan-stage-log.json

# Check failure details
jq '.failure' scan-stage-log.json

# Compare with recipe to verify configuration
mc cp minio/scanner-artifacts/{job_id}/scan-recipe.json .
jq '.environment' scan-recipe.json
```

---

## Database Schema

### Orchestrator Database (PostgreSQL)

```sql
-- Jobs table: Core job tracking
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    state TEXT NOT NULL,                    -- PENDING, EXTRACTING, etc.
    input_type TEXT NOT NULL,               -- "zip" or "urls"
    input_path TEXT NOT NULL,               -- MinIO staging path
    urls TEXT,                              -- JSON array for URL jobs
    pod_id TEXT,                            -- Podman pod ID
    config_json TEXT NOT NULL,              -- Serialized JobConfig
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT,
    provenance_path TEXT,                   -- Local path
    provenance_key TEXT,                    -- MinIO key
    expected_scanners TEXT,                 -- JSON array
    completed_scanners TEXT,                -- JSON array
    scanner_results TEXT,                   -- JSON map
    extraction_started_at TIMESTAMP,
    extraction_completed_at TIMESTAMP,
    scan_started_at TIMESTAMP,
    scan_completed_at TIMESTAMP,
    pages_scanned INTEGER DEFAULT 0,
    total_issues INTEGER DEFAULT 0,
    critical_issues INTEGER DEFAULT 0,
    serious_issues INTEGER DEFAULT 0,
    moderate_issues INTEGER DEFAULT 0,
    minor_issues INTEGER DEFAULT 0
);

CREATE INDEX idx_jobs_state ON jobs(state);
CREATE INDEX idx_jobs_created_at ON jobs(created_at);

-- Job events table: Audit log
CREATE TABLE job_events (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES jobs(id),
    event TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload_json TEXT,
    request_id TEXT,
    run_id TEXT,
    producer TEXT,
    nats_subject TEXT,
    nats_stream TEXT,
    nats_consumer TEXT,
    nats_stream_seq INTEGER,
    nats_consumer_seq INTEGER,
    nats_deliveries INTEGER,
    nats_stored_at TIMESTAMP,
    handler_status TEXT,
    handler_error TEXT,
    duration_ms INTEGER
);

CREATE INDEX idx_job_events_job_id ON job_events(job_id);
```

### Platform API Status Source

`platform/api` is stateless for job status reads. It now fetches status snapshots directly from
the orchestrator admin API (`GET /api/v1/jobs/{id}`) and keeps only a short-lived in-memory
pending cache immediately after `job.created` publish.

### Entity Relationships

```
+------------------+
|      jobs        |
+------------------+
| id (PK)          |
| state            |
| input_type       |
| ...              |
+--------+---------+
         |
         | 1:N
         v
+------------------+
|   job_events     |
+------------------+
| id (PK)          |
| job_id (FK)      |
| event            |
| timestamp        |
| ...              |
+------------------+
```

---

## Report Schema

### Unified Report v2

The report schema is defined in JSON Schema and generates TypeScript and Go types.

**Source**: `packages/contracts/report/schema/unified-report.v2.schema.json`

```
UnifiedReportV2
+-- version: string (semver "2.x.x")
+-- meta: ReportMeta
|   +-- jobId: string
|   +-- baseUrl: string
|   +-- scannedAt: ISO datetime
|   +-- completedAt: ISO datetime
|   +-- durationMs: number
|
+-- summary: ReportSummary
|   +-- score: 0-100
|   +-- scoreGrade: "A+" to "F"
|   +-- totalIssues: number
|   +-- bySeverity: SeverityCounts
|   +-- byScanner: { [scanner]: number }
|   +-- pagesScanned: number
|   +-- pagesWithIssues: number
|   +-- lighthouseCategories?: [...]
|
+-- scanners: ScannerSummary[]
|   +-- id: string
|   +-- name: string
|   +-- status: "success" | "failed" | "skipped"
|   +-- issueCount: number
|   +-- durationMs: number
|   +-- resultsPath?: string
|   +-- reportPath?: string
|
+-- pages: PageSummary[]
|   +-- id: string
|   +-- url: string
|   +-- path?: string
|   +-- issueCount: number
|   +-- bySeverity: SeverityCounts
|   +-- overview?: PageOverview
|
+-- issues: IssueDetail[]
|   +-- id: string
|   +-- scanner: string
|   +-- severity: IssueSeverity
|   +-- category: string
|   +-- title: string
|   +-- description: string
|   +-- helpUrl?: string
|   +-- occurrences: Occurrence[]
|   +-- userImpact?: UserImpact
|   +-- metadata?: object
|
+-- artifacts?: ReportArtifact[]
+-- errors?: ReportError[]
```

### Scoring Algorithm

```go
// Calculate issue penalty
penalty := float64(counts.Critical*10 + counts.Serious*5 + counts.Moderate*2 + counts.Minor)

// Apply logarithmic scaling for diminishing returns
scaled := 20*math.Log10(penalty+1) + penalty*0.3

// Calculate score (0-100)
score := int(100 - scaled + 0.5)

// Map to grade
// A+: 97+, A: 93-96, A-: 90-92
// B+: 87-89, B: 83-86, B-: 80-82
// C+: 77-79, C: 73-76, C-: 70-72
// D+: 67-69, D: 63-66, D-: 60-62
// F: <60
```

### User Impact Model

Each issue can include human-readable impact information explaining who is affected and how:

```typescript
interface UserImpact {
  statement?: string;                // Narrative description of impact
  affectedGroups?: UserGroup[];      // Which user groups are affected
  severity?: UserImpactSeverity;     // How severely they're affected
  userStory?: string;                // First-person user perspective
}

type UserGroup =
  | "blind"        // Screen reader users
  | "low-vision"   // Users with visual impairments
  | "motor"        // Users with motor control difficulties
  | "cognitive"    // Users with cognitive disabilities
  | "deaf"         // Users with hearing impairments
  | "vestibular"   // Users sensitive to motion/animation
  | "all";         // Affects all users

type UserImpactSeverity =
  | "blocking"      // Cannot complete task at all
  | "degraded"      // Can complete but with significant difficulty
  | "inconvenient"; // Minor friction but manageable
```

**Example Issue with User Impact:**

```json
{
  "id": "color-contrast-001",
  "title": "Text has insufficient color contrast",
  "severity": "serious",
  "userImpact": {
    "statement": "Users with low vision cannot read the navigation text against the background color",
    "affectedGroups": ["low-vision", "cognitive"],
    "severity": "degraded",
    "userStory": "As a user with low vision, I struggle to read the menu items because they blend into the background"
  }
}
```

**Affected Group Guidelines:**

| Group | Common Issues |
|-------|---------------|
| `blind` | Missing alt text, unlabeled buttons, broken skip links, focus management |
| `low-vision` | Color contrast, text sizing, zoom support, focus indicators |
| `motor` | Keyboard access, target sizes, timeout issues, drag-only interactions |
| `cognitive` | Complex language, inconsistent navigation, error recovery, distractions |
| `deaf` | Missing captions, audio-only content, visual-only alerts |
| `vestibular` | Auto-playing animations, parallax effects, rapid flashing |

### Issue Deduplication

When multiple scanners flag the same issue (e.g., both axe and lighthouse flag `image-alt`), the orchestrator deduplicates by scanner priority:

```go
var scannerPriority = map[string]int{
    "axe":              100,  // Accessibility specialist (highest)
    "security-headers": 90,   // Domain specialist
    "lighthouse":       80,   // Broad coverage
    "link-checker":     70,
    "seo":              60,   // Lowest priority
}
```

The higher-priority scanner's issue is kept, with metadata noting which other scanners also detected it.

**Cross-Scanner Rule Mappings** (`rule_deduplication.go`)

The orchestrator maintains equivalence mappings for rules that detect the same underlying problem:

```go
// Example mappings (40+ total)
var ruleEquivalences = map[string][]string{
    // Image alt text: axe, lighthouse, and seo all check this
    "image-alt":           {"axe:image-alt", "lighthouse:image-alt", "seo:images-missing-alt"},

    // Document title
    "document-title":      {"axe:document-title", "lighthouse:document-title", "seo:missing-title"},

    // Meta description
    "meta-description":    {"lighthouse:meta-description", "seo:missing-meta-description"},

    // Link text quality
    "link-name":           {"axe:link-name", "lighthouse:link-name"},

    // Heading structure
    "heading-order":       {"axe:heading-order", "seo:heading-order"},

    // Color contrast
    "color-contrast":      {"axe:color-contrast", "lighthouse:color-contrast"},

    // Form labels
    "label":               {"axe:label", "lighthouse:label"},

    // Language attribute
    "html-has-lang":       {"axe:html-has-lang", "lighthouse:html-has-lang"},

    // Viewport meta
    "viewport":            {"axe:meta-viewport", "lighthouse:viewport", "seo:viewport"},
}
```

**Deduplication Process:**

1. Group all issues by normalized rule ID (using equivalence map)
2. For each group, select the issue from the highest-priority scanner
3. Add `alsoDetectedBy` metadata listing other scanners that found it
4. Return deduplicated issue list

**Example Result:**

```json
{
  "id": "image-alt",
  "scanner": "axe",
  "title": "Images must have alternate text",
  "metadata": {
    "alsoDetectedBy": ["lighthouse", "seo"],
    "originalRuleIds": {
      "axe": "image-alt",
      "lighthouse": "image-alt",
      "seo": "images-missing-alt"
    }
  }
}
```

This approach reduces report noise while preserving evidence that multiple tools validated the finding.

---

## Security Architecture

### API Authentication

```
+-------------+
|   Client    |
+------+------+
       | X-API-Key: <token>
       | or
       | Authorization: Bearer <token>
       v
+------+------+
| apiKeyMiddleware |
| (Platform API)   |
+------+------+
       | Validates against PLATFORM_API_TOKEN env
       v
+------+------+
|   Handler   |
+-------------+
```

- **Optional auth**: If `PLATFORM_API_TOKEN` is not set, requests pass through
- **Proxy passthrough**: Caddy forwards auth headers to Platform API

### SSRF Protection

URL-based scans are validated to prevent Server-Side Request Forgery:

```go
// Blocked patterns:
// - Private IPs: 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
// - Link-local: 169.254.0.0/16
// - Cloud metadata: 169.254.169.254
// - IPv6 loopback, link-local, unique-local (fc00::/7)

// Validation steps:
// 1. Parse URL
// 2. Check scheme (http/https only)
// 3. Resolve hostname to IPs
// 4. Check each IP against blocked ranges
// 5. Reject if any IP is private/blocked
```

### Rate Limiting

StageFlow does not include a separate in-repo API gateway rate-limiter service.
Apply rate limiting at the edge proxy (Caddy, CDN, or load balancer) per deployment.

### Request Limits

| Limit           | Value    | Applied To             |
|-----------------|----------|------------------------|
| Max body size   | 105 MB   | All POST requests      |
| Request timeout | 60s      | Standard requests      |
| Upload timeout  | 5 min    | ZIP uploads            |
| Max URLs        | 100      | URL-based scans        |
| Max URL length  | 2048     | Each URL               |

---

## Infrastructure

### Container Images

| Image                          | Base                     | Size   |
|--------------------------------|--------------------------|--------|
| `stageflow/platform-api`       | distroless               | ~15 MB |
| `stageflow/orchestrator`       | distroless               | ~20 MB |
| `stageflow/extractor`          | alpine                   | ~10 MB |
| `stageflow/scanner-runner`     | playwright (chromium)    | ~1.5 GB|
| `stageflow/frontend`           | nginx:alpine             | ~25 MB |

### Network Topology

```
stageflow_net (Podman network)
|
+-- stageflow-pod (Production)
|   +-- nats (nats:2.12.2-alpine)
|   +-- minio (minio/minio:RELEASE.2025-09-07T16-13-09Z)
|   +-- postgres (postgres:17-alpine)
|   +-- grafana (grafana/grafana:12.2.0)
|   +-- platform-api
|   +-- orchestrator
|   +-- frontend
|
+-- Dynamic pods (created per job)
    +-- job-{jobID}
        +-- extractor (ephemeral)
        +-- scanner-axe-{jobID}
        +-- scanner-lighthouse-{jobID}
        +-- ...
```

### Volume Mounts

| Volume             | Purpose                          | Access      |
|--------------------|----------------------------------|-------------|
| `nats_data`        | JetStream persistence            | Read/Write  |
| `minio_data`       | Object storage                   | Read/Write  |
| `postgres_data`    | Orchestrator PostgreSQL data     | Read/Write  |
| `grafana_data`     | Dashboards, plugins              | Read/Write  |
| `workspace-{job}`  | Extracted files / provenance     | Read-only*  |
| `results-{job}`    | Scanner output                   | Read/Write  |

*Scanners mount workspace as read-only; only extractor writes to it.

### Production Deployment (Quadlets)

```
/home/user/.config/containers/systemd/
+-- stageflow.pod             # Pod definition
+-- stageflow.target          # Orchestrates all services
+-- stageflow-nats.container
+-- stageflow-minio.container
+-- stageflow-postgres.container
+-- stageflow-orchestrator.container
+-- stageflow-platform-api.container
+-- stageflow-frontend.container
+-- stageflow-grafana.container
```

Service dependencies:
```
stageflow.target
+-- stageflow-pod.service
+-- stageflow-nats.service
+-- stageflow-minio.service
+-- stageflow-postgres.service
+-- stageflow-orchestrator.service (after: nats, minio, postgres)
+-- stageflow-platform-api.service (after: orchestrator)
+-- stageflow-frontend.service (after: platform-api)
+-- stageflow-grafana.service (after: orchestrator)
```

### Caddy Reverse Proxy

```
{$STAGEFLOW_PUBLIC_DOMAIN} {
    # API routes -> Platform API
    handle /api/* {
        reverse_proxy 127.0.0.1:8100
    }

    # MinIO presigned URLs with CORS
    handle /scanner-artifacts/* {
        header Access-Control-Allow-Origin *
        reverse_proxy 127.0.0.1:9100
    }

    # Monitoring -> Grafana
    handle /monitoring* {
        reverse_proxy 127.0.0.1:3101
    }

    # Everything else -> Frontend SPA
    handle {
        reverse_proxy 127.0.0.1:3100
    }
}
```

---

## Appendix: Key File Locations

| Component               | Primary Files                                           |
|-------------------------|---------------------------------------------------------|
| Platform API            | `platform/api/internal/api/router.go`                   |
| Orchestrator            | `platform/orchestrator/internal/orchestrator/orchestrator.go` |
| Scanner Runner          | `platform/scanner-runner/src/worker.ts`                 |
| Scanner Base            | `platform/scanner-runner/src/core/scanner-base.ts`      |
| Report Schema           | `packages/contracts/report/schema/unified-report.v2.schema.json` |
| Scanner Manifest Schema | `packages/contracts/scanner-manifest/schema/scanner-manifest.schema.json` |
| Shared Go Models        | `packages/shared-go/models/job.go`                      |
| NATS Events             | `packages/shared-go/events/types.go`                    |
| Orchestrator DB Schema  | `platform/orchestrator/internal/db/schema.sql`          |
| Compose Files           | `infra/compose/podman-compose*.yml`                     |
| Quadlet Templates       | `infra/quadlets/templates/*.in`                         |
| Task Runner             | `justfile`                                              |

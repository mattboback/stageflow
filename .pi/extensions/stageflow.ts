import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import {
	DEFAULT_MAX_BYTES,
	DEFAULT_MAX_LINES,
	formatSize,
	truncateTail,
	withFileMutationQueue,
} from "@mariozechner/pi-coding-agent";
import { Type } from "@sinclair/typebox";

const DEFAULT_SCAN_TIMEOUT_SECONDS = 600;
const DEFAULT_PROJECT_TIMEOUT_SECONDS = 900;
const DEFAULT_DOCTOR_TIMEOUT_SECONDS = 180;
const DEFAULT_MAX_ISSUES = 50;
const ISSUE_PREVIEW_LIMIT = 8;
const SCANNER_PREVIEW_LIMIT = 10;
const REMOTE_PROJECT_PREVIEW_LIMIT = 20;

interface CommandResult {
	command: string;
	args: string[];
	cwd: string;
	stdout: string;
	stderr: string;
	code: number | null;
	signal: NodeJS.Signals | null;
	timedOut: boolean;
}

interface StageflowReportEnvelope {
	schema?: string;
	job?: {
		id?: string;
		state?: string;
		error?: string;
	};
	links?: {
		job?: string;
		results?: string;
	};
	urls?: string[];
	filters?: {
		max_issues?: number;
		issues_returned?: number;
		issues_total?: number;
		truncated?: boolean;
	};
	report?: {
		summary?: {
			score?: number;
			scoreGrade?: string;
			totalIssues?: number;
			pagesScanned?: number;
			pagesWithIssues?: number;
			bySeverity?: Record<string, number>;
		};
		issues?: Array<{
			id?: string;
			scanner?: string;
			ruleId?: string;
			severity?: string;
			title?: string;
			pageUrl?: string;
			category?: string;
			howToFix?: string;
		}>;
		scanners?: Array<{
			id?: string;
			name?: string;
			status?: string;
			issueCount?: number;
			durationMs?: number;
		}>;
		pages?: Array<{
			id?: string;
			url?: string;
			issueCount?: number;
			durationMs?: number;
		}>;
	};
}

interface StageflowDiffEnvelope {
	schema?: string;
	baseline?: {
		file?: string;
		score?: number;
		totalIssues?: number;
	};
	current?: {
		jobId?: string;
		file?: string;
		score?: number;
		totalIssues?: number;
	};
	delta?: {
		scoreDelta?: number;
		newIssues?: number;
		fixedIssues?: number;
		unchangedIssues?: number;
	};
	new?: Array<{
		id?: string;
		scanner?: string;
		ruleId?: string;
		severity?: string;
		title?: string;
		pageUrl?: string;
	}>;
	fixed?: Array<{
		id?: string;
		scanner?: string;
		ruleId?: string;
		severity?: string;
		title?: string;
		pageUrl?: string;
	}>;
	regressed?: boolean;
}

interface StageflowScannersResponse {
	total?: number;
	enabled?: number;
	categories?: string[];
	scanners?: Array<{
		id?: string;
		name?: string;
		version?: string;
		description?: string;
		categories?: string[];
		aliases?: string[];
		enabled?: boolean;
		builtIn?: boolean;
		capabilities?: {
			supportsScreenshots?: boolean;
			requiresBrowser?: boolean;
			supportsOffline?: boolean;
			maxConcurrency?: number;
			estimatedTimePerPage?: number;
		};
	}>;
}

interface StageflowRemoteProject {
	id?: string;
	slug?: string;
	name?: string;
	urls?: string[];
	scanners?: string[];
	baseline_job_id?: string;
	created_at?: string;
	updated_at?: string;
}

function isURL(value: string): boolean {
	return /^https?:\/\//i.test(value.trim());
}

function cleanPathArg(value: string): string {
	return value.startsWith("@") ? value.slice(1) : value;
}

function resolvePathFrom(baseDir: string, pathValue: string): string {
	return resolve(baseDir, cleanPathArg(pathValue));
}

function resolveReportTarget(baseDir: string, value: string): string {
	const trimmed = value.trim();
	return isURL(trimmed) ? trimmed : resolvePathFrom(baseDir, trimmed);
}

function formatDurationMs(durationMs: number | undefined): string | undefined {
	if (typeof durationMs !== "number" || Number.isNaN(durationMs)) return undefined;
	if (durationMs < 1000) return `${durationMs}ms`;
	if (durationMs < 60_000) return `${(durationMs / 1000).toFixed(1)}s`;
	return `${(durationMs / 60_000).toFixed(1)}m`;
}

function formatSignedNumber(value: number | undefined): string | undefined {
	if (typeof value !== "number" || Number.isNaN(value)) return undefined;
	return value > 0 ? `+${value}` : `${value}`;
}

function appendAPIFlags(args: string[], api?: string, apiKey?: string): string[] {
	if (api?.trim()) args.push("--api", api.trim());
	if (apiKey?.trim()) args.push("--api-key", apiKey.trim());
	return args;
}

function safeJsonParse<T>(text: string): T | undefined {
	try {
		return JSON.parse(text) as T;
	} catch {
		return undefined;
	}
}

function extractTopLevelJSONObjectStrings(text: string): string[] {
	const docs: string[] = [];
	let depth = 0;
	let inString = false;
	let escaped = false;
	let start = -1;

	for (let i = 0; i < text.length; i += 1) {
		const ch = text[i];

		if (inString) {
			if (escaped) {
				escaped = false;
				continue;
			}
			if (ch === "\\") {
				escaped = true;
				continue;
			}
			if (ch === '"') {
				inString = false;
			}
			continue;
		}

		if (ch === '"') {
			inString = true;
			continue;
		}

		if (ch === "{") {
			if (depth === 0) start = i;
			depth += 1;
			continue;
		}

		if (ch === "}") {
			if (depth === 0) continue;
			depth -= 1;
			if (depth === 0 && start >= 0) {
				docs.push(text.slice(start, i + 1));
				start = -1;
			}
		}
	}

	return docs;
}

function parseJSONDocuments<T = unknown>(text: string): T[] {
	return extractTopLevelJSONObjectStrings(text)
		.map((doc) => safeJsonParse<T>(doc))
		.filter((doc): doc is T => doc !== undefined);
}

function buildSeveritySummary(summary: Record<string, number> | undefined): string {
	const sev = summary ?? {};
	return [
		`critical=${sev.critical ?? 0}`,
		`serious=${sev.serious ?? 0}`,
		`moderate=${sev.moderate ?? 0}`,
		`minor=${sev.minor ?? 0}`,
		`info=${sev.info ?? 0}`,
	].join(" ");
}

function buildIssuePreview(
	issues: Array<{
		severity?: string;
		scanner?: string;
		title?: string;
		ruleId?: string;
		pageUrl?: string;
	}> | undefined,
	limit = ISSUE_PREVIEW_LIMIT,
): string[] {
	if (!issues || issues.length === 0) return [];

	return issues.slice(0, limit).map((issue) => {
		const parts = [
			`[${issue.severity ?? "unknown"}]`,
			issue.scanner ? `[${issue.scanner}]` : undefined,
			issue.title ?? issue.ruleId ?? "Untitled issue",
			issue.pageUrl ? `— ${issue.pageUrl}` : undefined,
		].filter(Boolean);

		return `- ${parts.join(" ")}`;
	});
}

function buildReportSummary(envelope: StageflowReportEnvelope, outputPath: string): string {
	const lines: string[] = [];
	const summary = envelope.report?.summary;
	const filters = envelope.filters;
	const issues = envelope.report?.issues ?? [];
	const scanners = envelope.report?.scanners ?? [];
	const pages = envelope.report?.pages ?? [];

	lines.push(`StageFlow report (${envelope.schema ?? "unknown schema"})`);

	if (envelope.job?.id || envelope.job?.state) {
		lines.push(`Job: ${envelope.job?.id ?? "unknown"} (${envelope.job?.state ?? "unknown state"})`);
	}

	if (typeof summary?.score === "number") {
		lines.push(
			`Score: ${summary.score}${summary.scoreGrade ? ` (${summary.scoreGrade})` : ""}`,
		);
	}

	if (typeof summary?.totalIssues === "number") {
		lines.push(`Issues: ${summary.totalIssues} total • ${buildSeveritySummary(summary.bySeverity)}`);
	}

	if (typeof summary?.pagesScanned === "number") {
		lines.push(
			`Pages: ${summary.pagesScanned} scanned${typeof summary.pagesWithIssues === "number" ? `, ${summary.pagesWithIssues} with issues` : ""}`,
		);
	}

	if (filters) {
		const returned = filters.issues_returned ?? issues.length;
		const total = filters.issues_total ?? returned;
		lines.push(
			`Returned findings: ${returned}/${total}${filters.truncated ? " (truncated by CLI filter)" : ""}`,
		);
	}

	if (envelope.urls && envelope.urls.length > 0) {
		lines.push(`URLs: ${envelope.urls.join(", ")}`);
	}

	if (scanners.length > 0) {
		lines.push("Scanners:");
		for (const scanner of scanners.slice(0, SCANNER_PREVIEW_LIMIT)) {
			const duration = formatDurationMs(scanner.durationMs);
			lines.push(
				`- ${scanner.id ?? scanner.name ?? "unknown"}: ${scanner.status ?? "unknown"}${typeof scanner.issueCount === "number" ? `, issues=${scanner.issueCount}` : ""}${duration ? `, duration=${duration}` : ""}`,
			);
		}

		if (scanners.length > SCANNER_PREVIEW_LIMIT) {
			lines.push(`- … ${scanners.length - SCANNER_PREVIEW_LIMIT} more scanner entries`);
		}
	}

	if (pages.length > 0) {
		lines.push("Pages:");
		for (const page of pages.slice(0, 5)) {
			const duration = formatDurationMs(page.durationMs);
			lines.push(
				`- ${page.url ?? page.id ?? "unknown page"}${typeof page.issueCount === "number" ? `, issues=${page.issueCount}` : ""}${duration ? `, duration=${duration}` : ""}`,
			);
		}
		if (pages.length > 5) {
			lines.push(`- … ${pages.length - 5} more pages`);
		}
	}

	const issuePreview = buildIssuePreview(issues);
	if (issuePreview.length > 0) {
		lines.push("Top findings:");
		lines.push(...issuePreview);
	}

	if (envelope.links?.results) {
		lines.push(`Results URL: ${envelope.links.results}`);
	}

	lines.push(`Full JSON: ${outputPath}`);
	lines.push("Use the read tool on the saved JSON path if you need the raw report.");

	return lines.join("\n");
}

function buildDiffSummary(envelope: StageflowDiffEnvelope, outputPath: string): string {
	const lines: string[] = [];
	const delta = envelope.delta;

	lines.push(`StageFlow diff (${envelope.schema ?? "unknown schema"})`);
	lines.push(`Regression detected: ${envelope.regressed ? "yes" : "no"}`);

	if (envelope.baseline?.file || typeof envelope.baseline?.score === "number") {
		lines.push(
			`Baseline: ${envelope.baseline?.file ?? "(live)"}${typeof envelope.baseline?.score === "number" ? `, score=${envelope.baseline.score}` : ""}${typeof envelope.baseline?.totalIssues === "number" ? `, issues=${envelope.baseline.totalIssues}` : ""}`,
		);
	}

	if (envelope.current?.jobId || envelope.current?.file || typeof envelope.current?.score === "number") {
		lines.push(
			`Current: ${envelope.current?.file ?? envelope.current?.jobId ?? "(live)"}${typeof envelope.current?.score === "number" ? `, score=${envelope.current.score}` : ""}${typeof envelope.current?.totalIssues === "number" ? `, issues=${envelope.current.totalIssues}` : ""}`,
		);
	}

	if (delta) {
		lines.push(
			`Delta: score=${formatSignedNumber(delta.scoreDelta) ?? "n/a"}, new=${delta.newIssues ?? 0}, fixed=${delta.fixedIssues ?? 0}, unchanged=${delta.unchangedIssues ?? 0}`,
		);
	}

	const newIssues = buildIssuePreview(envelope.new, 6);
	if (newIssues.length > 0) {
		lines.push("New issues:");
		lines.push(...newIssues);
	}

	const fixedIssues = buildIssuePreview(envelope.fixed, 4);
	if (fixedIssues.length > 0) {
		lines.push("Fixed issues:");
		lines.push(...fixedIssues);
	}

	lines.push(`Full JSON: ${outputPath}`);
	lines.push("Use the read tool on the saved JSON path if you need the raw diff envelope.");

	return lines.join("\n");
}

function buildScannersSummary(response: StageflowScannersResponse, outputPath: string): string {
	const lines: string[] = [];
	const scanners = response.scanners ?? [];

	lines.push("StageFlow scanners");
	lines.push(`Enabled: ${response.enabled ?? 0}/${response.total ?? scanners.length}`);
	if (response.categories && response.categories.length > 0) {
		lines.push(`Categories: ${response.categories.join(", ")}`);
	}

	for (const scanner of scanners.slice(0, SCANNER_PREVIEW_LIMIT)) {
		const capabilities: string[] = [];
		if (scanner.capabilities?.supportsScreenshots) capabilities.push("screenshots");
		if (scanner.capabilities?.requiresBrowser) capabilities.push("browser");
		if (scanner.capabilities?.supportsOffline) capabilities.push("offline");
		if (typeof scanner.capabilities?.maxConcurrency === "number" && scanner.capabilities.maxConcurrency > 0) {
			capabilities.push(`maxConcurrency=${scanner.capabilities.maxConcurrency}`);
		}

		lines.push(
			`- ${scanner.id ?? scanner.name ?? "unknown"}: ${scanner.enabled === false ? "disabled" : "enabled"}${scanner.version ? `, v${scanner.version}` : ""}${scanner.categories && scanner.categories.length > 0 ? `, categories=${scanner.categories.join("/")}` : ""}${capabilities.length > 0 ? `, ${capabilities.join(", ")}` : ""}`,
		);
	}

	if (scanners.length > SCANNER_PREVIEW_LIMIT) {
		lines.push(`- … ${scanners.length - SCANNER_PREVIEW_LIMIT} more scanners`);
	}

	lines.push(`Full JSON: ${outputPath}`);
	lines.push("Use the read tool on the saved JSON path if you need the full scanner manifest list.");

	return lines.join("\n");
}

function buildRemoteProjectSummary(
	project: StageflowRemoteProject,
	outputPath: string,
	heading = "StageFlow remote project",
): string {
	const lines: string[] = [];

	lines.push(heading);
	lines.push(`Slug: ${project.slug ?? "unknown"}`);
	lines.push(`Name: ${project.name ?? project.slug ?? "unknown"}`);
	if (project.id) lines.push(`ID: ${project.id}`);
	if (project.urls && project.urls.length > 0) {
		lines.push("URLs:");
		for (const url of project.urls) lines.push(`- ${url}`);
	}
	if (project.scanners && project.scanners.length > 0) {
		lines.push(`Scanners: ${project.scanners.join(", ")}`);
	}
	if (project.baseline_job_id) {
		lines.push(`Baseline job: ${project.baseline_job_id}`);
	}
	if (project.created_at) lines.push(`Created: ${project.created_at}`);
	if (project.updated_at) lines.push(`Updated: ${project.updated_at}`);
	lines.push(`Full JSON: ${outputPath}`);
	lines.push("Use the read tool on the saved JSON path if you need the raw project object.");

	return lines.join("\n");
}

function buildRemoteProjectListSummary(projects: StageflowRemoteProject[], outputPath: string): string {
	const lines: string[] = [];

	lines.push(`StageFlow remote projects (${projects.length})`);
	for (const project of projects.slice(0, REMOTE_PROJECT_PREVIEW_LIMIT)) {
		const scanners = project.scanners && project.scanners.length > 0 ? `, scanners=${project.scanners.join("/")}` : "";
		const baseline = project.baseline_job_id ? `, baseline=${project.baseline_job_id}` : "";
		lines.push(
			`- ${project.slug ?? "unknown"}: ${project.name ?? project.slug ?? "unknown"}, urls=${project.urls?.length ?? 0}${scanners}${baseline}`,
		);
	}
	if (projects.length > REMOTE_PROJECT_PREVIEW_LIMIT) {
		lines.push(`- … ${projects.length - REMOTE_PROJECT_PREVIEW_LIMIT} more projects`);
	}
	lines.push(`Full JSON: ${outputPath}`);
	lines.push("Use the read tool on the saved JSON path if you need the full project list.");

	return lines.join("\n");
}

function buildRemoteProjectScanSummary(
	reportEnvelope: StageflowReportEnvelope,
	diffEnvelope: StageflowDiffEnvelope | undefined,
	outputPath: string,
): string {
	const parts = [buildReportSummary(reportEnvelope, outputPath)];
	if (diffEnvelope) {
		parts.push(buildDiffSummary(diffEnvelope, outputPath));
	}
	return parts.join("\n\n");
}

function buildTextSummary(title: string, output: string, outputPath?: string): string {
	const truncation = truncateTail(output.trim() || "(no output)", {
		maxLines: Math.min(DEFAULT_MAX_LINES, 200),
		maxBytes: Math.min(DEFAULT_MAX_BYTES, 12 * 1024),
	});

	let text = `${title}\n${truncation.content}`;
	if (truncation.truncated && outputPath) {
		text += `\n\n[Output truncated: showing ${truncation.outputLines} of ${truncation.totalLines} lines (${formatSize(truncation.outputBytes)} of ${formatSize(truncation.totalBytes)}). Full output saved to: ${outputPath}]`;
	} else if (outputPath) {
		text += `\n\nFull output: ${outputPath}`;
	}

	return text;
}

function findStageflowRepoRoot(startDir: string): string | undefined {
	let current = resolve(startDir);

	for (;;) {
		if (existsSync(join(current, "clients", "cli", "main.go"))) {
			return current;
		}

		const parent = resolve(current, "..");
		if (parent === current) {
			return undefined;
		}
		current = parent;
	}
}

async function saveTempFile(prefix: string, extension: string, content: string): Promise<string> {
	const dir = await mkdtemp(join(tmpdir(), "pi-stageflow-"));
	const filePath = join(dir, `${prefix}.${extension}`);
	await withFileMutationQueue(filePath, async () => {
		await writeFile(filePath, content, "utf8");
	});
	return filePath;
}

async function runStageflowCommand(
	repoRoot: string,
	args: string[],
	timeoutSeconds: number,
	signal: AbortSignal | undefined,
	onUpdate: ((partial: { content: Array<{ type: "text"; text: string }>; details?: Record<string, unknown> }) => void) | undefined,
): Promise<CommandResult> {
	const command = "go";
	const commandArgs = ["run", "./clients/cli", ...args];

	return new Promise<CommandResult>((resolvePromise, rejectPromise) => {
		let stdout = "";
		let stderr = "";
		let timedOut = false;
		let finished = false;
		let lastProgressAt = 0;

		const child = spawn(command, commandArgs, {
			cwd: repoRoot,
			env: process.env,
			stdio: ["ignore", "pipe", "pipe"],
		});

		onUpdate?.({
			content: [{ type: "text", text: `Running: ${command} ${commandArgs.join(" ")}` }],
			details: { cwd: repoRoot },
		});

		const emitProgress = (chunk: string) => {
			const now = Date.now();
			if (now - lastProgressAt < 800) return;

			const lines = chunk
				.split(/\r?\n/)
				.map((line) => line.trim())
				.filter(Boolean);
			const lastLine = lines.at(-1);
			if (!lastLine) return;

			lastProgressAt = now;
			onUpdate?.({
				content: [{ type: "text", text: lastLine }],
				details: { cwd: repoRoot, running: true },
			});
		};

		const timeoutId = setTimeout(() => {
			timedOut = true;
			child.kill("SIGTERM");
			const killTimer = setTimeout(() => child.kill("SIGKILL"), 2000);
			killTimer.unref?.();
		}, Math.max(1, timeoutSeconds) * 1000);
		timeoutId.unref?.();

		const abortHandler = () => {
			child.kill("SIGTERM");
			const killTimer = setTimeout(() => child.kill("SIGKILL"), 2000);
			killTimer.unref?.();
		};

		signal?.addEventListener("abort", abortHandler, { once: true });

		child.stdout.on("data", (chunk: Buffer | string) => {
			stdout += chunk.toString();
		});

		child.stderr.on("data", (chunk: Buffer | string) => {
			const text = chunk.toString();
			stderr += text;
			emitProgress(text);
		});

		child.on("error", (error) => {
			clearTimeout(timeoutId);
			signal?.removeEventListener("abort", abortHandler);
			if (finished) return;
			finished = true;
			rejectPromise(new Error(`Failed to start StageFlow CLI via ${command}: ${error.message}`));
		});

		child.on("close", (code, exitSignal) => {
			clearTimeout(timeoutId);
			signal?.removeEventListener("abort", abortHandler);
			if (finished) return;
			finished = true;

			if (signal?.aborted) {
				rejectPromise(new Error("StageFlow command aborted"));
				return;
			}

			resolvePromise({
				command,
				args: commandArgs,
				cwd: repoRoot,
				stdout,
				stderr,
				code,
				signal: exitSignal,
				timedOut,
			});
		});
	});
}

async function throwCommandFailure(action: string, result: CommandResult, extraHint?: string): Promise<never> {
	const combined = [result.stderr.trim(), result.stdout.trim()].filter(Boolean).join("\n\n");
	const outputPath = await saveTempFile("stageflow-error", "log", combined || "(no output)");
	const truncation = truncateTail(combined || "(no output)", {
		maxLines: 120,
		maxBytes: 12 * 1024,
	});

	let message = `${action} failed`;
	if (result.code !== null) message += ` with exit code ${result.code}`;
	if (result.timedOut) message += " (timed out)";
	if (result.signal) message += ` (signal ${result.signal})`;
	message += `.\n\n${truncation.content}`;

	if (truncation.truncated) {
		message += `\n\n[Output truncated: ${truncation.outputLines} of ${truncation.totalLines} lines, ${formatSize(truncation.outputBytes)} of ${formatSize(truncation.totalBytes)}. Full output: ${outputPath}]`;
	} else {
		message += `\n\nFull output: ${outputPath}`;
	}

	if (extraHint) {
		message += `\n\n${extraHint}`;
	}

	throw new Error(message);
}

function projectConfigHint(projectPath: string): string {
	return `If this project has not been bootstrapped yet, run stageflow_project_init with path ${projectPath}.`;
}

export default function stageflowExtension(pi: ExtensionAPI) {
	const commonPathDescription = "Path to the target project. Relative paths resolve from the current working directory. Defaults to the current working directory.";
	const commonApiDescription = "Optional StageFlow API base URL. If omitted, the CLI/environment defaults are used.";
	const commonApiKeyDescription = "Optional StageFlow API key. If omitted, the CLI/environment defaults are used.";
	const commonSlugDescription = "Remote StageFlow project slug.";

	pi.registerTool({
		name: "stageflow_scanners",
		label: "StageFlow Scanners",
		description:
			"List scanners available from the current StageFlow API. Saves the full JSON response to a temp file and returns a concise summary.",
		promptSnippet: "List available StageFlow scanners and their capabilities.",
		promptGuidelines: [
			"Use this before choosing non-default StageFlow scanner IDs.",
		],
		parameters: Type.Object({
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags(["scanners", "--format", "json"], params.api, params.apiKey);

			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? 60,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure("stageflow scanners", result);
			}

			const response = safeJsonParse<StageflowScannersResponse>(result.stdout);
			if (!response) {
				await throwCommandFailure("stageflow scanners (parse JSON)", result);
			}

			const outputPath = await saveTempFile("stageflow-scanners", "json", result.stdout);
			return {
				content: [{ type: "text", text: buildScannersSummary(response, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					outputPath,
					total: response.total,
					enabled: response.enabled,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_scan",
		label: "StageFlow Scan",
		description:
			"Run a one-off StageFlow URL scan through the local StageFlow CLI in this repository. Saves the full JSON report to a temp file and returns a concise summary.",
		promptSnippet: "Run a StageFlow scan for one or more explicit URLs.",
		promptGuidelines: [
			"Use this when the user gives concrete URLs to scan.",
			"Use stageflow_project_run instead when the repository already has .stageflow/config.yaml and should be scanned through Project Mode.",
		],
		parameters: Type.Object({
			urls: Type.Array(Type.String({ description: "URL to scan." }), {
				description: "One or more URLs to scan.",
				minItems: 1,
			}),
			scanners: Type.Optional(
				Type.Array(Type.String({ description: "Scanner ID such as axe, lighthouse, seo, link-checker, ai-navigator." }), {
					description: "Optional scanner IDs. Omit to use the CLI defaults.",
				}),
			),
			screenshot: Type.Optional(Type.Boolean({ description: "Capture screenshots during the scan." })),
			allowPrivateTargets: Type.Optional(
				Type.Boolean({ description: "Allow private or loopback targets such as localhost or 127.0.0.1." }),
			),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			maxIssues: Type.Optional(
				Type.Number({
					description: "Maximum issues for the CLI to include in the JSON payload. Defaults to 50.",
					minimum: 0,
					maximum: 5000,
				}),
			),
			summaryOnly: Type.Optional(
				Type.Boolean({ description: "Ask the CLI to omit individual findings and return only summary sections." }),
			),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 7200,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags(["scan", ...params.urls.map((url) => url.trim()), "--format", "json"], params.api, params.apiKey);
			if (params.scanners && params.scanners.length > 0) {
				args.push("--scanners", params.scanners.join(","));
			}
			if (params.screenshot) args.push("--screenshot");
			if (params.allowPrivateTargets) args.push("--allow-private-targets");
			args.push("--max-issues", String(params.maxIssues ?? DEFAULT_MAX_ISSUES));
			if (params.summaryOnly) args.push("--summary-only");
			args.push("--timeout", `${Math.max(1, Math.floor(params.timeoutSeconds ?? DEFAULT_SCAN_TIMEOUT_SECONDS))}s`);

			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? DEFAULT_SCAN_TIMEOUT_SECONDS,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure("stageflow scan", result);
			}

			const envelope = safeJsonParse<StageflowReportEnvelope>(result.stdout);
			if (!envelope) {
				await throwCommandFailure("stageflow scan (parse JSON)", result);
			}

			const outputPath = await saveTempFile("stageflow-scan", "json", result.stdout);
			return {
				content: [{ type: "text", text: buildReportSummary(envelope, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					outputPath,
					jobId: envelope.job?.id,
					state: envelope.job?.state,
					score: envelope.report?.summary?.score,
					totalIssues: envelope.report?.summary?.totalIssues,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_project_init",
		label: "StageFlow Project Init",
		description:
			"Scaffold .stageflow/config.yaml and .stageflow/README.md for Project Mode in the selected repository path.",
		promptSnippet: "Initialize StageFlow Project Mode in a repository.",
		promptGuidelines: [
			"Use this before stageflow_project_doctor or stageflow_project_run when .stageflow/config.yaml does not exist yet.",
		],
		parameters: Type.Object({
			path: Type.Optional(Type.String({ description: commonPathDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const targetPath = params.path ? resolvePathFrom(ctx.cwd, params.path) : ctx.cwd;
			const args = ["project", "init", targetPath];
			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? 60,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure("stageflow project init", result);
			}

			const combinedOutput = [result.stderr.trim(), result.stdout.trim()].filter(Boolean).join("\n\n");
			const outputPath = await saveTempFile("stageflow-project-init", "log", combinedOutput || "(no output)");
			const configPath = join(targetPath, ".stageflow", "config.yaml");
			const guidePath = join(targetPath, ".stageflow", "README.md");

			return {
				content: [
					{
						type: "text",
						text: buildTextSummary(
							`Initialized StageFlow Project Mode in ${targetPath}\nConfig: ${configPath}\nGuide: ${guidePath}\nNext step: run stageflow_project_doctor on the same path.`,
							combinedOutput,
							outputPath,
						),
					},
				],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					projectPath: targetPath,
					configPath,
					guidePath,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_project_doctor",
		label: "StageFlow Project Doctor",
		description:
			"Validate local StageFlow Project Mode wiring (.stageflow/config.yaml, dev readiness, StageFlow API reachability) without running a scan.",
		promptSnippet: "Validate StageFlow Project Mode setup for a repository.",
		promptGuidelines: [
			"Use this before a first project scan or when Project Mode fails.",
		],
		parameters: Type.Object({
			path: Type.Optional(Type.String({ description: commonPathDescription })),
			skipDev: Type.Optional(
				Type.Boolean({ description: "Skip starting the dev server and only validate static config and API checks." }),
			),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 7200,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const targetPath = params.path ? resolvePathFrom(ctx.cwd, params.path) : ctx.cwd;
			const args = ["project", "doctor", targetPath, "--timeout", `${Math.max(1, Math.floor(params.timeoutSeconds ?? DEFAULT_DOCTOR_TIMEOUT_SECONDS))}s`];
			if (params.skipDev) args.push("--skip-dev");

			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? DEFAULT_DOCTOR_TIMEOUT_SECONDS,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure(
					"stageflow project doctor",
					result,
					projectConfigHint(targetPath),
				);
			}

			const combinedOutput = [result.stderr.trim(), result.stdout.trim()].filter(Boolean).join("\n\n");
			const outputPath = await saveTempFile("stageflow-project-doctor", "log", combinedOutput || "(no output)");
			return {
				content: [
					{
						type: "text",
						text: buildTextSummary(`StageFlow project doctor passed for ${targetPath}`, combinedOutput, outputPath),
					},
				],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					projectPath: targetPath,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_project_run",
		label: "StageFlow Project Run",
		description:
			"Run StageFlow Project Mode for a repository with .stageflow/config.yaml: start the dev server, wait for readiness, scan, and stop the app. Saves the full JSON report to a temp file.",
		promptSnippet: "Run StageFlow Project Mode using the repository's .stageflow/config.yaml.",
		promptGuidelines: [
			"Use this for repository-local web app scans after project setup exists.",
			"If project setup is missing or uncertain, use stageflow_project_doctor or stageflow_project_init first.",
		],
		parameters: Type.Object({
			path: Type.Optional(Type.String({ description: commonPathDescription })),
			maxIssues: Type.Optional(
				Type.Number({
					description: "Maximum issues for the CLI to include in the JSON payload. Defaults to 50.",
					minimum: 0,
					maximum: 5000,
				}),
			),
			summaryOnly: Type.Optional(
				Type.Boolean({ description: "Ask the CLI to omit individual findings and return only summary sections." }),
			),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 7200,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const targetPath = params.path ? resolvePathFrom(ctx.cwd, params.path) : ctx.cwd;
			const args = [
				"project",
				targetPath,
				"--format",
				"json",
				"--max-issues",
				String(params.maxIssues ?? DEFAULT_MAX_ISSUES),
				"--timeout",
				`${Math.max(1, Math.floor(params.timeoutSeconds ?? DEFAULT_PROJECT_TIMEOUT_SECONDS))}s`,
			];
			if (params.summaryOnly) args.push("--summary-only");

			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? DEFAULT_PROJECT_TIMEOUT_SECONDS,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure(
					"stageflow project",
					result,
					projectConfigHint(targetPath),
				);
			}

			const envelope = safeJsonParse<StageflowReportEnvelope>(result.stdout);
			if (!envelope) {
				await throwCommandFailure("stageflow project (parse JSON)", result);
			}

			const outputPath = await saveTempFile("stageflow-project-report", "json", result.stdout);
			return {
				content: [{ type: "text", text: buildReportSummary(envelope, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					projectPath: targetPath,
					outputPath,
					jobId: envelope.job?.id,
					state: envelope.job?.state,
					score: envelope.report?.summary?.score,
					totalIssues: envelope.report?.summary?.totalIssues,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_list",
		label: "StageFlow Remote Project List",
		description:
			"List remote StageFlow projects from the configured API. Saves the full JSON response to a temp file and returns a concise summary.",
		promptSnippet: "List remote StageFlow projects registered on the API.",
		parameters: Type.Object({
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags(["project", "list", "--format", "json"], params.api, params.apiKey);
			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure("stageflow project list", result);
			}

			const projects = safeJsonParse<StageflowRemoteProject[]>(result.stdout);
			if (!projects) {
				await throwCommandFailure("stageflow project list (parse JSON)", result);
			}

			const outputPath = await saveTempFile("stageflow-remote-projects", "json", result.stdout);
			return {
				content: [{ type: "text", text: buildRemoteProjectListSummary(projects, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					count: projects.length,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_show",
		label: "StageFlow Remote Project Show",
		description:
			"Show a single remote StageFlow project by slug. Saves the full JSON response to a temp file and returns a concise summary.",
		promptSnippet: "Show one remote StageFlow project by slug.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags(["project", "show", params.slug, "--format", "json"], params.api, params.apiKey);
			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure(`stageflow project show ${params.slug}`, result);
			}

			const project = safeJsonParse<StageflowRemoteProject>(result.stdout);
			if (!project) {
				await throwCommandFailure(`stageflow project show ${params.slug} (parse JSON)`, result);
			}

			const outputPath = await saveTempFile(`stageflow-remote-project-${params.slug}`, "json", result.stdout);
			return {
				content: [{ type: "text", text: buildRemoteProjectSummary(project, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: project.slug ?? params.slug,
					name: project.name,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_create",
		label: "StageFlow Remote Project Create",
		description:
			"Create a remote StageFlow project on the configured API. Saves the created project JSON to a temp file and returns a concise summary.",
		promptSnippet: "Create a remote StageFlow project.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			name: Type.Optional(Type.String({ description: "Optional display name. Defaults to the slug if omitted." })),
			urls: Type.Array(Type.String({ description: "Target URL to include in the remote project." }), {
				description: "One or more project URLs.",
				minItems: 1,
			}),
			scanners: Type.Optional(
				Type.Array(Type.String({ description: "Optional scanner module ID." }), {
					description: "Optional scanner allowlist for the remote project.",
				}),
			),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}
			if (params.name !== undefined && !params.name.trim()) {
				throw new Error("Project name, if provided, must not be empty.");
			}

			const args = appendAPIFlags(["project", "create", params.slug, "--format", "json"], params.api, params.apiKey);
			if (params.name?.trim()) args.push("--name", params.name.trim());
			for (const url of params.urls) args.push("--url", url.trim());
			for (const scanner of params.scanners ?? []) args.push("--scanner", scanner.trim());

			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure(`stageflow project create ${params.slug}`, result);
			}

			const project = safeJsonParse<StageflowRemoteProject>(result.stdout);
			if (!project) {
				await throwCommandFailure(`stageflow project create ${params.slug} (parse JSON)`, result);
			}

			const outputPath = await saveTempFile(`stageflow-remote-project-create-${params.slug}`, "json", result.stdout);
			return {
				content: [{ type: "text", text: buildRemoteProjectSummary(project, outputPath, "Created StageFlow remote project") }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: project.slug ?? params.slug,
					name: project.name,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_update",
		label: "StageFlow Remote Project Update",
		description:
			"Update a remote StageFlow project on the configured API. Replaces URLs or scanners when provided. Saves the updated project JSON to a temp file and returns a concise summary.",
		promptSnippet: "Update a remote StageFlow project.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			name: Type.Optional(Type.String({ description: "Optional new display name." })),
			urls: Type.Optional(
				Type.Array(Type.String({ description: "Replacement project URL." }), {
					description: "Replacement URL list. If provided, it replaces the existing URLs.",
				}),
			),
			scanners: Type.Optional(
				Type.Array(Type.String({ description: "Replacement scanner module ID." }), {
					description: "Replacement scanner list. If provided, it replaces the existing scanners.",
				}),
			),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const hasName = params.name !== undefined;
			const hasURLs = Array.isArray(params.urls);
			const hasScanners = Array.isArray(params.scanners);
			if (!hasName && !hasURLs && !hasScanners) {
				throw new Error("At least one of name, urls, or scanners must be provided.");
			}
			if (hasName && !params.name?.trim()) {
				throw new Error("Project name, if provided, must not be empty.");
			}
			if (hasURLs && params.urls!.length === 0) {
				throw new Error("Replacing URLs with an empty list is not supported by this wrapper. Use the CLI manually if you need to clear all URLs.");
			}
			if (hasScanners && params.scanners!.length === 0) {
				throw new Error("Replacing scanners with an empty list is not supported by this wrapper. Use the CLI manually if you need to clear all scanners.");
			}

			const args = appendAPIFlags(["project", "update", params.slug, "--format", "json"], params.api, params.apiKey);
			if (hasName && params.name?.trim()) args.push("--name", params.name.trim());
			for (const url of params.urls ?? []) args.push("--url", url.trim());
			for (const scanner of params.scanners ?? []) args.push("--scanner", scanner.trim());

			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure(`stageflow project update ${params.slug}`, result);
			}

			const project = safeJsonParse<StageflowRemoteProject>(result.stdout);
			if (!project) {
				await throwCommandFailure(`stageflow project update ${params.slug} (parse JSON)`, result);
			}

			const outputPath = await saveTempFile(`stageflow-remote-project-update-${params.slug}`, "json", result.stdout);
			return {
				content: [{ type: "text", text: buildRemoteProjectSummary(project, outputPath, "Updated StageFlow remote project") }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: project.slug ?? params.slug,
					name: project.name,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_delete",
		label: "StageFlow Remote Project Delete",
		description:
			"Delete a remote StageFlow project by slug. In interactive mode it asks for confirmation unless force is true.",
		promptSnippet: "Delete a remote StageFlow project.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			force: Type.Optional(Type.Boolean({ description: "Skip the interactive confirmation prompt." })),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			if (!params.force) {
				if (!ctx.hasUI) {
					throw new Error("Deleting a remote project in non-interactive mode requires force=true.");
				}
				const ok = await ctx.ui.confirm(
					"Delete remote StageFlow project?",
					`Delete remote project ${params.slug}? This cannot be undone.`,
				);
				if (!ok) {
					throw new Error(`Deletion cancelled for remote project ${params.slug}.`);
				}
			}

			const args = appendAPIFlags(["project", "delete", params.slug], params.api, params.apiKey);
			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure(`stageflow project delete ${params.slug}`, result);
			}

			const combinedOutput = [result.stderr.trim(), result.stdout.trim()].filter(Boolean).join("\n\n");
			const outputPath = await saveTempFile(`stageflow-remote-project-delete-${params.slug}`, "log", combinedOutput || "(no output)");
			return {
				content: [
					{
						type: "text",
						text: buildTextSummary(`Deleted StageFlow remote project ${params.slug}`, combinedOutput || `Deleted project ${params.slug}.`, outputPath),
					},
				],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: params.slug,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_promote",
		label: "StageFlow Remote Project Promote",
		description:
			"Promote a job ID to become the saved baseline for a remote StageFlow project.",
		promptSnippet: "Promote a StageFlow job as the baseline for a remote project.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			jobId: Type.String({ description: "StageFlow job ID to promote as the baseline." }),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 3600,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags(["project", "promote", params.slug, "--job-id", params.jobId], params.api, params.apiKey);
			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? 60, signal, onUpdate);
			if (result.code !== 0) {
				await throwCommandFailure(`stageflow project promote ${params.slug}`, result);
			}

			const combinedOutput = [result.stderr.trim(), result.stdout.trim()].filter(Boolean).join("\n\n");
			const outputPath = await saveTempFile(`stageflow-remote-project-promote-${params.slug}`, "log", combinedOutput || "(no output)");
			return {
				content: [
					{
						type: "text",
						text: buildTextSummary(`Promoted job ${params.jobId} as the baseline for remote project ${params.slug}`, combinedOutput || `Promoted baseline for ${params.slug}.`, outputPath),
					},
				],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: params.slug,
					jobId: params.jobId,
					outputPath,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_remote_project_scan",
		label: "StageFlow Remote Project Scan",
		description:
			"Run a scan using a saved remote StageFlow project slug. Saves the full JSON output to a temp file and returns a concise summary, including diff details when a baseline exists.",
		promptSnippet: "Run a scan for a saved remote StageFlow project.",
		parameters: Type.Object({
			slug: Type.String({ description: commonSlugDescription }),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			maxIssues: Type.Optional(
				Type.Number({
					description: "Maximum issues for the CLI to include in the JSON payload. Defaults to 50.",
					minimum: 0,
					maximum: 5000,
				}),
			),
			summaryOnly: Type.Optional(
				Type.Boolean({ description: "Ask the CLI to omit individual findings and return only summary sections." }),
			),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds.",
					minimum: 1,
					maximum: 7200,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const args = appendAPIFlags([
				"scan",
				"--project",
				params.slug,
				"--format",
				"json",
				"--max-issues",
				String(params.maxIssues ?? DEFAULT_MAX_ISSUES),
				"--timeout",
				`${Math.max(1, Math.floor(params.timeoutSeconds ?? DEFAULT_PROJECT_TIMEOUT_SECONDS))}s`,
			], params.api, params.apiKey);
			if (params.summaryOnly) args.push("--summary-only");

			const result = await runStageflowCommand(repoRoot, args, params.timeoutSeconds ?? DEFAULT_PROJECT_TIMEOUT_SECONDS, signal, onUpdate);
			if (result.code !== 0 && result.code !== 1) {
				await throwCommandFailure(`stageflow scan --project ${params.slug}`, result);
			}

			const docs = parseJSONDocuments<StageflowReportEnvelope | StageflowDiffEnvelope>(result.stdout);
			const reportEnvelope = docs.find((doc) => "report" in doc) as StageflowReportEnvelope | undefined;
			const diffEnvelope = docs.find((doc) => "delta" in doc) as StageflowDiffEnvelope | undefined;
			if (!reportEnvelope) {
				await throwCommandFailure(`stageflow scan --project ${params.slug} (parse JSON)`, result);
			}

			const outputPath = await saveTempFile(`stageflow-remote-project-scan-${params.slug}`, "json", result.stdout);
			return {
				content: [{ type: "text", text: buildRemoteProjectScanSummary(reportEnvelope, diffEnvelope, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					slug: params.slug,
					outputPath,
					jobId: reportEnvelope.job?.id,
					state: reportEnvelope.job?.state,
					score: reportEnvelope.report?.summary?.score,
					totalIssues: reportEnvelope.report?.summary?.totalIssues,
					regressed: diffEnvelope?.regressed,
					scoreDelta: diffEnvelope?.delta?.scoreDelta,
					newIssues: diffEnvelope?.delta?.newIssues,
				},
			};
		},
	});

	pi.registerTool({
		name: "stageflow_diff",
		label: "StageFlow Diff",
		description:
			"Compare a saved StageFlow baseline JSON report against another saved report or a live URL. Saves the full diff envelope to a temp file and returns a concise summary.",
		promptSnippet: "Diff StageFlow reports or compare a baseline report against a live URL.",
		promptGuidelines: [
			"Use this for regression comparisons after a baseline JSON report already exists.",
		],
		parameters: Type.Object({
			baselinePath: Type.String({ description: "Path to a saved StageFlow JSON report file to use as the baseline." }),
			current: Type.String({
				description: "Either another StageFlow JSON report path or a live URL to scan for comparison.",
			}),
			api: Type.Optional(Type.String({ description: commonApiDescription })),
			apiKey: Type.Optional(Type.String({ description: commonApiKeyDescription })),
			timeoutSeconds: Type.Optional(
				Type.Number({
					description: "Command timeout in seconds when the current target is a live URL.",
					minimum: 1,
					maximum: 7200,
				}),
			),
		}),
		async execute(_toolCallId, params, signal, onUpdate, ctx) {
			const repoRoot = findStageflowRepoRoot(ctx.cwd);
			if (!repoRoot) {
				throw new Error("Could not locate the StageFlow repository root (expected clients/cli/main.go in a parent directory).");
			}

			const baselinePath = resolvePathFrom(ctx.cwd, params.baselinePath);
			const currentTarget = resolveReportTarget(ctx.cwd, params.current);
			const args = appendAPIFlags([
				"diff",
				baselinePath,
				currentTarget,
				"--format",
				"json",
				"--timeout",
				`${Math.max(1, Math.floor(params.timeoutSeconds ?? DEFAULT_SCAN_TIMEOUT_SECONDS))}s`,
			], params.api, params.apiKey);

			const result = await runStageflowCommand(
				repoRoot,
				args,
				params.timeoutSeconds ?? DEFAULT_SCAN_TIMEOUT_SECONDS,
				signal,
				onUpdate,
			);
			if (result.code !== 0) {
				await throwCommandFailure("stageflow diff", result);
			}

			const envelope = safeJsonParse<StageflowDiffEnvelope>(result.stdout);
			if (!envelope) {
				await throwCommandFailure("stageflow diff (parse JSON)", result);
			}

			const outputPath = await saveTempFile("stageflow-diff", "json", result.stdout);
			return {
				content: [{ type: "text", text: buildDiffSummary(envelope, outputPath) }],
				details: {
					command: `${result.command} ${result.args.join(" ")}`,
					cwd: result.cwd,
					baselinePath,
					currentTarget,
					outputPath,
					regressed: envelope.regressed,
					scoreDelta: envelope.delta?.scoreDelta,
					newIssues: envelope.delta?.newIssues,
					fixedIssues: envelope.delta?.fixedIssues,
				},
			};
		},
	});

	pi.registerCommand("stageflow-reload", {
		description: "Reload pi extensions, including this StageFlow extension",
		handler: async (_args, ctx) => {
			await ctx.reload();
			return;
		},
	});

	pi.registerTool({
		name: "stageflow_reload",
		label: "StageFlow Reload",
		description: "Reload pi so changes to .pi/extensions/stageflow.ts are picked up.",
		parameters: Type.Object({}),
		async execute() {
			pi.sendUserMessage("/stageflow-reload", { deliverAs: "followUp" });
			return {
				content: [{ type: "text", text: "Queued /stageflow-reload as a follow-up command." }],
				details: {},
			};
		},
	});

}

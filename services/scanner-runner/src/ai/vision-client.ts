import { setTimeout as delay } from "node:timers/promises";
import sharp from "sharp";

import type { VisionConfig, VisionMessage, VisionResponse } from "./types";

import { isPlainObject } from "./json";

const OPENROUTER_API_URL = "https://openrouter.ai/api/v1/chat/completions";

interface RetryConfig {
	maxAttempts: number;
	baseDelayMs: number;
}

class Semaphore {
	private inFlight = 0;
	private readonly queue: (() => void)[] = [];

	constructor(private readonly max: number) {}

	async acquire(): Promise<() => void> {
		if (this.inFlight < this.max) {
			this.inFlight += 1;
			return (): void => {
				this.release();
			};
		}

		await new Promise<void>((resolve) => {
			this.queue.push(resolve);
		});

		this.inFlight += 1;
		return (): void => {
			this.release();
		};
	}

	private release(): void {
		this.inFlight -= 1;
		const next = this.queue.shift();
		if (next) {
			next();
		}
	}
}

export class VisionClient {
	private readonly semaphore: Semaphore | null;
	private readonly retry: RetryConfig;

	constructor(private readonly config: VisionConfig) {
		const maxConcurrency = config.maxConcurrency ?? 1;
		this.semaphore = maxConcurrency > 1 ? new Semaphore(maxConcurrency) : null;

		const retryConfig = config.retry;
		this.retry = {
			maxAttempts: retryConfig?.maxAttempts ?? 1,
			baseDelayMs: retryConfig?.baseDelayMs ?? 500,
		};
	}

	async analyze(image: Buffer, prompt: string): Promise<VisionResponse> {
		const { dataUrl } = await this.prepareImageDataUrl(image);

		const messages: VisionMessage[] = [
			{
				role: "user",
				content: [
					{ type: "text", text: prompt },
					{ type: "image_url", image_url: { url: dataUrl } },
				],
			},
		];

		return this.chat(messages);
	}

	async chat(messages: VisionMessage[]): Promise<VisionResponse> {
		const release = this.semaphore ? await this.semaphore.acquire() : null;
		try {
			return await this.chatWithRetry(messages);
		} finally {
			release?.();
		}
	}

	private async chatWithRetry(
		messages: VisionMessage[],
	): Promise<VisionResponse> {
		const attempts = Math.max(1, this.retry.maxAttempts);

		for (let attempt = 1; attempt <= attempts; attempt += 1) {
			try {
				return await this.chatOnce(messages);
			} catch (err) {
				if (attempt >= attempts) {
					throw err;
				}

				if (!isRetryableError(err)) {
					throw err;
				}

				const backoffMs = this.retry.baseDelayMs * 2 ** (attempt - 1);
				await delay(backoffMs);
			}
		}

		throw new Error("unreachable");
	}

	private async chatOnce(messages: VisionMessage[]): Promise<VisionResponse> {
		const controller = new AbortController();
		const timeoutMs = this.config.timeoutMs ?? 30_000;
		const timeoutId = setTimeout(() => {
			controller.abort();
		}, timeoutMs);

		try {
			const response = await fetch(OPENROUTER_API_URL, {
				method: "POST",
				headers: this.buildHeaders(),
				body: JSON.stringify({
					model: this.config.model,
					messages,
					max_tokens: this.config.maxTokens,
				}),
				signal: controller.signal,
			});

			const responseText = await response.text();
			const parsed = safeJsonParse(responseText);

			if (!response.ok) {
				const details = isPlainObject(parsed)
					? JSON.stringify(parsed)
					: responseText;
				throw new Error(
					`OpenRouter request failed (${response.status}): ${details}`,
				);
			}

			const content = extractContent(parsed);
			if (!content) {
				throw new Error("OpenRouter response missing assistant content");
			}

			const usage = extractUsage(parsed);
			return usage ? { content, usage } : { content };
		} finally {
			clearTimeout(timeoutId);
		}
	}

	private buildHeaders(): Record<string, string> {
		const headers: Record<string, string> = {
			Authorization: `Bearer ${this.config.apiKey}`,
			"Content-Type": "application/json",
			Accept: "application/json",
		};

		if (this.config.appReferer) {
			headers["HTTP-Referer"] = this.config.appReferer;
		}

		if (this.config.appTitle) {
			headers["X-Title"] = this.config.appTitle;
		}

		return headers;
	}

	private async prepareImageDataUrl(
		image: Buffer,
	): Promise<{ mimeType: string; dataUrl: string }> {
		const maxBytes = this.config.maxImageBytes;
		if (!maxBytes || image.byteLength <= maxBytes) {
			return {
				mimeType: "image/png",
				dataUrl: `data:image/png;base64,${image.toString("base64")}`,
			};
		}

		const prepared = await this.compressToMaxBytes(image, maxBytes);
		return {
			mimeType: prepared.mimeType,
			dataUrl: `data:${prepared.mimeType};base64,${prepared.buffer.toString("base64")}`,
		};
	}

	private async compressToMaxBytes(
		input: Buffer,
		maxBytes: number,
	): Promise<{ buffer: Buffer; mimeType: string }> {
		let buffer = input;

		for (let pass = 0; pass < 6; pass += 1) {
			const quality = Math.max(30, 80 - pass * 10);
			const jpeg = await sharp(buffer).jpeg({ quality }).toBuffer();
			if (jpeg.byteLength <= maxBytes) {
				return { buffer: jpeg, mimeType: "image/jpeg" };
			}

			const metadata = await sharp(jpeg).metadata();
			const width = metadata.width;
			if (!width || width < 320) {
				buffer = jpeg;
				continue;
			}

			const resized = await sharp(jpeg)
				.resize({ width: Math.round(width * 0.85) })
				.jpeg({ quality })
				.toBuffer();

			if (resized.byteLength <= maxBytes) {
				return { buffer: resized, mimeType: "image/jpeg" };
			}

			buffer = resized;
		}

		return { buffer, mimeType: "image/jpeg" };
	}
}

function safeJsonParse(raw: string): unknown {
	const trimmed = raw.trim();
	if (!trimmed) {
		return undefined;
	}

	try {
		return JSON.parse(trimmed) as unknown;
	} catch {
		return undefined;
	}
}

function extractContent(response: unknown): string | undefined {
	if (!isPlainObject(response)) {
		return undefined;
	}

	const choices = response.choices;
	if (!Array.isArray(choices) || choices.length === 0) {
		return undefined;
	}

	const first: unknown = choices[0];
	if (!isPlainObject(first)) {
		return undefined;
	}

	const message = first.message;
	if (!isPlainObject(message)) {
		return undefined;
	}

	const content = message.content;
	if (typeof content !== "string") {
		return undefined;
	}

	return content;
}

function extractUsage(response: unknown): VisionResponse["usage"] | undefined {
	if (!isPlainObject(response)) {
		return undefined;
	}

	const usage = response.usage;
	if (!isPlainObject(usage)) {
		return undefined;
	}

	const promptTokens = usage.prompt_tokens;
	const completionTokens = usage.completion_tokens;
	const totalTokens = usage.total_tokens;

	if (
		typeof promptTokens !== "number" ||
		typeof completionTokens !== "number" ||
		typeof totalTokens !== "number"
	) {
		return undefined;
	}

	return { promptTokens, completionTokens, totalTokens };
}

function isRetryableError(err: unknown): boolean {
	if (!(err instanceof Error)) {
		return false;
	}

	const message = err.message.toLowerCase();
	return (
		message.includes("429") ||
		message.includes("rate") ||
		message.includes("timeout") ||
		message.includes("econnreset") ||
		message.includes("socket") ||
		message.includes("aborterror")
	);
}

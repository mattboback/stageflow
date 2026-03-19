import { describe, expect, it } from "vitest";

import {
	BlockedTargetError,
	type TargetAddressResolver,
	shouldEnforceRuntimeTargetValidation,
	validateRuntimeTargetURL,
} from "../../src/core/target-validation";

function staticResolver(
	records: Record<string, string[]>,
): TargetAddressResolver {
	return {
		resolve(hostname: string): Promise<string[]> {
			return Promise.resolve(records[hostname] ?? []);
		},
	};
}

describe("target validation", () => {
	it("allows public IP targets", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("https://8.8.8.8"),
		).resolves.toBeUndefined();
	});

	it("blocks private IP targets", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("https://10.0.0.1"),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("allows private targets when explicitly enabled", async () => {
		const previous = process.env.ALLOW_PRIVATE_TARGETS;
		process.env.ALLOW_PRIVATE_TARGETS = "true";

		try {
			await expect(
				validateRuntimeTargetURL("https://10.0.0.1"),
			).resolves.toBeUndefined();
			await expect(
				validateRuntimeTargetURL("https://127.0.0.1"),
			).resolves.toBeUndefined();
			await expect(
				validateRuntimeTargetURL("https://[::1]"),
			).resolves.toBeUndefined();
		} finally {
			if (previous === undefined) {
				delete process.env.ALLOW_PRIVATE_TARGETS;
			} else {
				process.env.ALLOW_PRIVATE_TARGETS = previous;
			}
		}
	});

	it("still blocks link-local targets when explicitly enabled", async () => {
		const previous = process.env.ALLOW_PRIVATE_TARGETS;
		process.env.ALLOW_PRIVATE_TARGETS = "true";

		try {
			await expect(
				validateRuntimeTargetURL("https://169.254.169.254"),
			).rejects.toBeInstanceOf(BlockedTargetError);
		} finally {
			if (previous === undefined) {
				delete process.env.ALLOW_PRIVATE_TARGETS;
			} else {
				process.env.ALLOW_PRIVATE_TARGETS = previous;
			}
		}
	});

	it("allows hostnames resolving to loopback when explicitly enabled", async () => {
		const previous = process.env.ALLOW_PRIVATE_TARGETS;
		process.env.ALLOW_PRIVATE_TARGETS = "true";

		const resolver = staticResolver({
			"localhost.test": ["127.0.0.1"],
		});

		try {
			await expect(
				validateRuntimeTargetURL("https://localhost.test", resolver),
			).resolves.toBeUndefined();
		} finally {
			if (previous === undefined) {
				delete process.env.ALLOW_PRIVATE_TARGETS;
			} else {
				process.env.ALLOW_PRIVATE_TARGETS = previous;
			}
		}
	});

	it("blocks unsupported schemes", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("ftp://example.com"),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("blocks hostnames resolving to blocked ranges", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		const resolver = staticResolver({
			"metadata.test": ["169.254.169.254"],
		});

		await expect(
			validateRuntimeTargetURL("https://metadata.test", resolver),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("allows hostnames resolving to public addresses", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		const resolver = staticResolver({
			"public.example": ["93.184.216.34"],
		});

		await expect(
			validateRuntimeTargetURL("https://public.example", resolver),
		).resolves.toBeUndefined();
	});

	it("blocks carrier-grade NAT addresses", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("https://100.64.0.1"),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("blocks benchmark network addresses", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("https://198.18.0.1"),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("blocks IPv6 docs addresses", async () => {
		delete process.env.ALLOW_PRIVATE_TARGETS;
		await expect(
			validateRuntimeTargetURL("https://[2001:db8::1]"),
		).rejects.toBeInstanceOf(BlockedTargetError);
	});

	it("enforces validation only for URL jobs", () => {
		process.env.SCAN_URLS = '["https://example.com"]';
		expect(shouldEnforceRuntimeTargetValidation()).toBe(true);

		delete process.env.SCAN_URLS;
		expect(shouldEnforceRuntimeTargetValidation()).toBe(false);
	});
});

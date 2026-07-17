const REDACTED = '[REDACTED]';

/**
 * Redact secret values in their raw, URI-component, and HTML form-encoded
 * representations. Browsers can expose any of these forms in URLs and errors.
 */
export function redactSecretValues(value: string, secretValues: Iterable<string>): string {
	const variants = new Set<string>();

	for (const secret of secretValues) {
		if (!secret) {
			continue;
		}

		variants.add(secret);
		addEncodedVariants(variants, encodeURIComponent(secret));

		const formEncoded = new URLSearchParams([['value', secret]]).toString().slice('value='.length);
		addEncodedVariants(variants, formEncoded);
	}

	return [...variants]
		.sort((left, right) => right.length - left.length)
		.reduce((redacted, secret) => redacted.replaceAll(secret, REDACTED), value);
}

/**
 * Recursively redact string values in arrays and plain JSON-like objects.
 *
 * Object keys are deliberately preserved: callers commonly pass typed values,
 * and a short secret such as `s` must not rename structural fields like
 * `issues` or `success`.
 */
export function redactStringValues<T>(value: T, redact: (input: string) => string): T {
	return transformStringValues(value, redact, false);
}

/**
 * Redact values and keys in schema-free metadata.
 *
 * Keep this restricted to records whose keys are data, rather than typed
 * application objects. Keys that collapse to the same redacted value receive
 * stable suffixes so redaction never drops data.
 */
export function redactDynamicStringValues<T>(value: T, redact: (input: string) => string): T {
	return transformStringValues(value, redact, true);
}

function transformStringValues<T>(
	value: T,
	redact: (input: string) => string,
	redactKeys: boolean
): T {
	const seen = new WeakMap<object, unknown>();

	function visit(input: unknown): unknown {
		if (typeof input === 'string') {
			return redact(input);
		}

		if (Array.isArray(input)) {
			const existing = seen.get(input);
			if (existing !== undefined) {
				return existing;
			}
			const output: unknown[] = [];
			seen.set(input, output);
			for (const entry of input) {
				output.push(visit(entry));
			}
			return output;
		}

		if (input !== null && typeof input === 'object') {
			const prototype = Object.getPrototypeOf(input) as unknown;
			if (prototype !== Object.prototype && prototype !== null) {
				return input;
			}

			const existing = seen.get(input);
			if (existing !== undefined) {
				return existing;
			}
			const output: Record<string, unknown> = {};
			seen.set(input, output);
			for (const [key, entry] of Object.entries(input)) {
				const candidateKey = redactKeys ? redact(key) : key;
				const outputKey = uniqueKey(output, candidateKey);
				Object.defineProperty(output, outputKey, {
					value: visit(entry),
					enumerable: true,
					configurable: true,
					writable: true
				});
			}
			return output;
		}

		return input;
	}

	return visit(value) as T;
}

function uniqueKey(output: Record<string, unknown>, candidate: string): string {
	if (!Object.hasOwn(output, candidate)) {
		return candidate;
	}

	let suffix = 2;
	while (Object.hasOwn(output, `${candidate}#${suffix}`)) {
		suffix += 1;
	}
	return `${candidate}#${suffix}`;
}

function addEncodedVariants(variants: Set<string>, encoded: string): void {
	variants.add(encoded);
	variants.add(encoded.replace(/%[0-9A-F]{2}/g, (triplet) => triplet.toLowerCase()));
}

/**
 * Resolves typed `{ from_env: NAME }` references in PreScanAction values
 * against an explicit allow-list derived from the recipe.
 *
 * Two invariants:
 *   1. Only env vars referenced by the recipe (auth steps + per-page actions)
 *      are ever resolved. Arbitrary env passthrough is impossible.
 *   2. Resolved credentials live only in memory inside this resolver. They
 *      never appear in stored Provenance, the unified report, the scan stage
 *      log, or any NATS event.
 */

import type {
	FromEnvReference,
	PreScanAction,
	PreScanActionValue,
	Provenance
} from './types';

import { isFromEnvReference } from './types';

export class SecretsResolutionError extends Error {
	readonly reference: string;

	constructor(message: string, reference: string) {
		super(message);
		this.name = 'SecretsResolutionError';
		this.reference = reference;
	}
}

export interface SecretsResolver {
	/**
	 * Returns the resolved value for the given reference. Throws if the
	 * reference is outside the allow-list or the env var is unset.
	 */
	resolve(reference: FromEnvReference): string;

	/**
	 * Resolves a PreScanActionValue. Literal strings pass through unchanged.
	 */
	resolveValue(value: PreScanActionValue): string;

	/** Sorted, frozen view of the allow-listed env var names. */
	readonly allowList: readonly string[];
}

export interface CreateSecretsResolverOptions {
	/** Env var names this resolver may read. Anything else is rejected. */
	allowList: Iterable<string>;
	/**
	 * Source environment. Defaults to `process.env`. Tests pass an explicit
	 * mapping to avoid relying on real process state.
	 */
	env?: Readonly<Record<string, string | undefined>>;
}

/**
 * Builds a resolver bound to the given allow-list. The allow-list should be
 * derived from {@link collectFromEnvReferences} on the same Provenance.
 */
export function createSecretsResolver(options: CreateSecretsResolverOptions): SecretsResolver {
	const allowList = Object.freeze([...new Set(options.allowList)].sort());
	const allowSet = new Set(allowList);
	const env = options.env ?? process.env;

	function resolve(reference: FromEnvReference): string {
		const name = reference.from_env;
		if (!allowSet.has(name)) {
			throw new SecretsResolutionError(
				`from_env reference "${name}" is not in the recipe allow-list. ` +
					`The allow-list is derived from the recipe at execution time and may not be expanded by callers.`,
				name
			);
		}

		const value = env[name];
		if (typeof value !== 'string' || value.length === 0) {
			throw new SecretsResolutionError(
				`from_env reference "${name}" is unset in the scanner-runner environment.`,
				name
			);
		}

		return value;
	}

	function resolveValue(value: PreScanActionValue): string {
		if (typeof value === 'string') {
			return value;
		}
		return resolve(value);
	}

	return { resolve, resolveValue, allowList };
}

/**
 * Walks Provenance and collects every env var name referenced via
 * `{ from_env: NAME }`. Used to build the resolver allow-list and, in the
 * orchestrator, the set of env vars to forward into the scanner-runner pod.
 */
export function collectFromEnvReferences(provenance: Provenance): string[] {
	const names = new Set<string>();

	collectFromActions(provenance.auth?.mode === 'form' ? provenance.auth.steps : [], names);

	for (const page of provenance.pages) {
		collectFromActions(page.pre_scan_actions ?? [], names);
	}

	return [...names].sort();
}

function collectFromActions(actions: readonly PreScanAction[], out: Set<string>): void {
	for (const action of actions) {
		if (action.type !== 'fill' && action.type !== 'select') {
			continue;
		}
		if (isFromEnvReference(action.value)) {
			out.add(action.value.from_env);
		}
	}
}

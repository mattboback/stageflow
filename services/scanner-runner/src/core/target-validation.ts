import { lookup } from 'node:dns/promises';
import { BlockList, isIP } from 'node:net';

import type { Provenance } from './types';

export interface TargetAddressResolver {
	resolve(hostname: string): Promise<string[]>;
}

export interface TargetValidationPolicy {
	allowedOrigins: readonly string[];
}

const defaultResolver: TargetAddressResolver = {
	async resolve(hostname: string): Promise<string[]> {
		const records = await lookup(hostname, { all: true, verbatim: true });
		return records.map((record) => record.address);
	}
};

const DISALLOWED_IPV4_SUBNETS: readonly { network: string; prefix: number }[] = [
	{ network: '0.0.0.0', prefix: 8 },
	{ network: '10.0.0.0', prefix: 8 },
	{ network: '100.64.0.0', prefix: 10 },
	{ network: '127.0.0.0', prefix: 8 },
	{ network: '169.254.0.0', prefix: 16 },
	{ network: '172.16.0.0', prefix: 12 },
	{ network: '192.0.0.0', prefix: 24 },
	{ network: '192.0.2.0', prefix: 24 },
	{ network: '192.168.0.0', prefix: 16 },
	{ network: '198.18.0.0', prefix: 15 },
	{ network: '198.51.100.0', prefix: 24 },
	{ network: '203.0.113.0', prefix: 24 },
	{ network: '224.0.0.0', prefix: 4 },
	{ network: '240.0.0.0', prefix: 4 }
];

const DISALLOWED_IPV6_SUBNETS: readonly { network: string; prefix: number }[] = [
	{ network: '::', prefix: 128 },
	{ network: '::1', prefix: 128 },
	{ network: '100::', prefix: 64 },
	{ network: '2001:db8::', prefix: 32 },
	{ network: 'fc00::', prefix: 7 },
	{ network: 'fe80::', prefix: 10 },
	{ network: 'fec0::', prefix: 10 },
	{ network: 'ff00::', prefix: 8 }
];

const ALLOWED_PRIVATE_IPV4_SUBNETS = new Set<string>([
	'10.0.0.0/8',
	'127.0.0.0/8',
	'172.16.0.0/12',
	'192.168.0.0/16'
]);

const ALLOWED_PRIVATE_IPV6_SUBNETS = new Set<string>(['::1/128']);

function createDisallowedBlockList(allowPrivateTargets: boolean): BlockList {
	const blockList = new BlockList();

	for (const subnet of DISALLOWED_IPV4_SUBNETS) {
		if (
			allowPrivateTargets &&
			ALLOWED_PRIVATE_IPV4_SUBNETS.has(`${subnet.network}/${subnet.prefix}`)
		) {
			continue;
		}

		blockList.addSubnet(subnet.network, subnet.prefix, 'ipv4');
	}

	for (const subnet of DISALLOWED_IPV6_SUBNETS) {
		if (
			allowPrivateTargets &&
			ALLOWED_PRIVATE_IPV6_SUBNETS.has(`${subnet.network}/${subnet.prefix}`)
		) {
			continue;
		}

		blockList.addSubnet(subnet.network, subnet.prefix, 'ipv6');
	}

	return blockList;
}

const disallowedBlockListPublic = createDisallowedBlockList(false);
const disallowedBlockListPrivate = createDisallowedBlockList(true);

export class BlockedTargetError extends Error {
	readonly targetURL: string;
	readonly reason: string;

	constructor(targetURL: string, reason: string) {
		super(`Blocked target URL "${targetURL}": ${reason}`);
		this.name = 'BlockedTargetError';
		this.targetURL = targetURL;
		this.reason = reason;
	}
}

function normalizeIPAddress(address: string): string {
	const trimmed = address.trim();
	const lower = trimmed.toLowerCase();

	if (lower.startsWith('::ffff:')) {
		const mapped = trimmed.slice('::ffff:'.length);
		if (isIP(mapped) === 4) {
			return mapped;
		}
	}

	return trimmed;
}

function shouldAllowPrivateTargets(): boolean {
	return process.env.ALLOW_PRIVATE_TARGETS === 'true';
}

function runtimeDisallowedBlockList(): BlockList {
	return shouldAllowPrivateTargets() ? disallowedBlockListPrivate : disallowedBlockListPublic;
}

function isDisallowedIPAddress(address: string, blockList: BlockList): boolean {
	const normalized = normalizeIPAddress(address);
	const ipVersion = isIP(normalized);
	if (ipVersion === 0) {
		return false;
	}

	if (ipVersion === 4) {
		return blockList.check(normalized, 'ipv4');
	}

	return blockList.check(normalized, 'ipv6');
}

function parseTargetURL(rawURL: string): URL {
	let target: URL;
	try {
		target = new URL(rawURL);
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		throw new Error(`Invalid target URL "${rawURL}": ${message}`);
	}

	if (target.protocol !== 'http:' && target.protocol !== 'https:') {
		throw new BlockedTargetError(rawURL, 'only http/https URLs are allowed');
	}

	if (!target.hostname) {
		throw new Error(`Invalid target URL "${rawURL}": missing hostname`);
	}

	return target;
}

export function shouldEnforceRuntimeTargetValidation(): boolean {
	return Boolean(process.env.SCAN_URLS);
}

function normalizeOrigin(rawURL: string): string | null {
	try {
		const target = parseTargetURL(rawURL);
		return target.origin;
	} catch {
		return null;
	}
}

function isLoopbackHostname(hostname: string): boolean {
	const host = hostname.replace(/^\[/, '').replace(/\]$/, '').toLowerCase();
	if (host === 'localhost' || host.endsWith('.localhost')) {
		return true;
	}

	if (host === '::1') {
		return true;
	}

	if (host.startsWith('127.')) {
		return true;
	}

	return false;
}

function allowedStaticOrigin(provenance: Provenance): string | null {
	if (provenance.mode !== 'static' && provenance.mode !== 'spa') {
		return null;
	}

	if (!provenance.base_url) {
		return null;
	}

	let baseURL: URL;
	try {
		baseURL = parseTargetURL(provenance.base_url);
	} catch {
		return null;
	}

	if (!isLoopbackHostname(baseURL.hostname)) {
		return null;
	}

	return baseURL.origin;
}

export function buildTargetValidationPolicy(provenance: Provenance): TargetValidationPolicy {
	const origins = new Set<string>();
	const staticOrigin = allowedStaticOrigin(provenance);
	if (staticOrigin !== null) {
		origins.add(staticOrigin);
	}

	return {
		allowedOrigins: [...origins].sort()
	};
}

export async function validateTargetURLForPolicy(
	rawURL: string,
	policy: TargetValidationPolicy,
	resolver: TargetAddressResolver = defaultResolver
): Promise<void> {
	const origin = normalizeOrigin(rawURL);
	if (origin !== null && policy.allowedOrigins.includes(origin)) {
		return;
	}

	await validateRuntimeTargetURL(rawURL, resolver);
}

export async function validateRuntimeTargetURL(
	rawURL: string,
	resolver: TargetAddressResolver = defaultResolver
): Promise<void> {
	const target = parseTargetURL(rawURL);
	const host = target.hostname.replace(/^\[/, '').replace(/\]$/, '');
	const blockList = runtimeDisallowedBlockList();

	if (isIP(host) !== 0) {
		if (isDisallowedIPAddress(host, blockList)) {
			throw new BlockedTargetError(rawURL, `IP ${host} is in a blocked network range`);
		}

		return;
	}

	let addresses: string[];
	try {
		addresses = await resolver.resolve(host);
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		throw new Error(`Failed to resolve host "${host}": ${message}`);
	}

	if (addresses.length === 0) {
		throw new Error(`Failed to resolve host "${host}": no addresses returned`);
	}

	for (const address of addresses) {
		if (isDisallowedIPAddress(address, blockList)) {
			throw new BlockedTargetError(
				rawURL,
				`hostname "${host}" resolves to blocked address ${address}`
			);
		}
	}
}

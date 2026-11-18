import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../../', import.meta.url));
const workspaces = ['clients/web', 'services/scanner-runner'];
const kinds = new Set([
	'files',
	'dependencies',
	'devDependencies',
	'optionalPeerDependencies',
	'unlisted',
	'unresolved',
	'exports',
	'types',
	'enumMembers',
	'namespaceMembers',
	'classMembers',
	'duplicates',
	'binaries'
]);

export function findingKey(finding) {
	return JSON.stringify([finding.workspace, finding.file, finding.kind, finding.symbol]);
}

export function normalizeReport(workspace, report) {
	if (!report || !Array.isArray(report.issues)) throw new Error('Invalid Knip JSON report');
	const findings = [];
	for (const issue of report.issues) {
		if (typeof issue.file !== 'string') throw new Error('Missing Knip issue file');
		for (const [kind, entries] of Object.entries(issue)) {
			if (kind === 'file') continue;
			if (!kinds.has(kind) || !Array.isArray(entries))
				throw new Error(`Unknown Knip issue kind: ${kind}`);
			for (const entry of entries) {
				const symbol = kind === 'files' ? '' : typeof entry === 'string' ? entry : entry.name;
				if (typeof symbol !== 'string') throw new Error(`Missing Knip symbol: ${kind}`);
				findings.push({ workspace, file: issue.file, kind, symbol });
			}
		}
	}
	return findings;
}

export function compareFindings(findings, allowances) {
	const actual = new Map(findings.map((finding) => [findingKey(finding), finding]));
	const allowed = new Map();
	for (const allowance of allowances) {
		if (
			!workspaces.includes(allowance.workspace) ||
			typeof allowance.owner !== 'string' ||
			!/^\S+$/.test(allowance.owner) ||
			!kinds.has(allowance.kind) ||
			typeof allowance.file !== 'string' ||
			typeof allowance.symbol !== 'string' ||
			typeof allowance.reason !== 'string' ||
			!allowance.reason.trim() ||
			/[*?{}]/.test(allowance.file)
		) {
			throw new Error('Knip allowances require an exact finding, a cleanup owner and a reason');
		}
		const key = findingKey(allowance);
		if (allowed.has(key)) throw new Error(`Duplicate Knip allowance: ${key}`);
		allowed.set(key, allowance);
	}
	return {
		unexpected: [...actual].filter(([key]) => !allowed.has(key)).map(([, value]) => value),
		stale: [...allowed].filter(([key]) => !actual.has(key)).map(([, value]) => value)
	};
}

export function runKnip(cwd, binary = path.join(cwd, 'node_modules/knip/bin/knip.js')) {
	const result = spawnSync(process.execPath, [binary, '--production', '--reporter', 'json'], {
		cwd,
		encoding: 'utf8',
		maxBuffer: 16 * 1024 * 1024
	});
	if (result.error || ![0, 1].includes(result.status)) {
		throw new Error(`Knip failed in ${cwd}: ${result.error?.message ?? result.stderr}`);
	}
	let report;
	try {
		report = JSON.parse(result.stdout);
	} catch {
		throw new Error(`Knip returned invalid JSON in ${cwd}: ${result.stderr}`);
	}
	if (result.status === 1 && (!Array.isArray(report.issues) || report.issues.length === 0)) {
		throw new Error(`Knip failed without findings in ${cwd}: ${result.stderr}`);
	}
	return report;
}

function main() {
	const selected = process.argv.slice(2);
	if (selected.some((workspace) => !workspaces.includes(workspace)))
		throw new Error('Expected clients/web or services/scanner-runner');
	const targets = selected.length ? selected : workspaces;
	const allowances = JSON.parse(
		readFileSync(path.join(root, 'devtools/scripts/knip-allowlist.json'), 'utf8')
	);
	// Validate the whole ledger even when checking one workspace.
	compareFindings([], allowances);
	const findings = targets.flatMap((workspace) =>
		normalizeReport(workspace, runKnip(path.join(root, workspace)))
	);
	const { unexpected, stale } = compareFindings(
		findings,
		allowances.filter((item) => targets.includes(item.workspace))
	);
	for (const finding of unexpected) console.error(`New Knip finding: ${findingKey(finding)}`);
	for (const finding of stale)
		console.error(`Remove stale Knip allowance (${finding.owner}): ${findingKey(finding)}`);
	if (unexpected.length || stale.length) process.exitCode = 1;
	else
		console.log(
			`Production Knip passed: ${targets.join(', ')} (${findings.length} exact findings assigned to cleanup owners)`
		);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
	try {
		main();
	} catch (error) {
		console.error(error.message);
		process.exitCode = 1;
	}
}

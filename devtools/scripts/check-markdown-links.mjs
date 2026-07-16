#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, isAbsolute, resolve } from 'node:path';
import process from 'node:process';

function markdownFiles() {
	if (process.argv.length > 2) {
		return process.argv.slice(2);
	}

	const output = execFileSync(
		'git',
		['ls-files', '--cached', '--others', '--exclude-standard', '--', '*.md'],
		{ encoding: 'utf8' }
	);

	return output.split('\n').filter(Boolean);
}

function localTarget(rawTarget) {
	let target = rawTarget.trim();
	if (target.startsWith('<') && target.endsWith('>')) {
		target = target.slice(1, -1);
	}

	if (/^(?:[a-z][a-z0-9+.-]*:|#)/i.test(target)) {
		return null;
	}

	const withoutTitle = target.match(/^(\S+)(?:\s+["'][^"']*["'])?$/)?.[1] ?? target;
	const path = withoutTitle.split('#', 1)[0].split('?', 1)[0];
	if (!path) {
		return null;
	}

	try {
		return decodeURIComponent(path);
	} catch {
		return path;
	}
}

function backtickedPath(rawTarget, file) {
	// Changelog entries intentionally preserve paths from the release in which
	// they existed; later removals must not rewrite that historical record.
	if (file.endsWith('/CHANGELOG.md') || file === 'CHANGELOG.md') {
		return null;
	}

	const target = rawTarget.trim().replace(/[.,;:]$/, '');
	if (
		target !== rawTarget ||
		target.length === 0 ||
		/\s/.test(target) ||
		/[?*{}[\]]/.test(target) ||
		/^(?:https?:|\/|#|--|\$)/.test(target) ||
		!target.includes('/')
	) {
		return null;
	}

	const path = target.split('#', 1)[0];
	const rootCandidate = resolve(process.cwd(), path);
	const relativeCandidate = resolve(dirname(file), path);
	const firstSegment = path.replace(/^\.\//, '').split('/', 1)[0];
	const looksRootRelative = existsSync(resolve(process.cwd(), firstSegment));
	const looksFileRelative = existsSync(resolve(dirname(file), firstSegment));
	const explicitlyFileRelative = path.startsWith('./') || path.startsWith('../');

	if (!looksRootRelative && !looksFileRelative) {
		return null;
	}

	return explicitlyFileRelative || !looksRootRelative ? relativeCandidate : rootCandidate;
}

const failures = [];

for (const file of markdownFiles()) {
	if (!existsSync(file)) {
		continue;
	}

	const source = readFileSync(file, 'utf8');
	const targets = [];

	for (const match of source.matchAll(/!?\[[^\]]*\]\(([^)]+)\)/g)) {
		targets.push(match[1]);
	}
	for (const match of source.matchAll(/^\s*\[[^\]]+\]:\s*(\S+)/gm)) {
		targets.push(match[1]);
	}

	for (const rawTarget of targets) {
		const target = localTarget(rawTarget);
		if (target === null) {
			continue;
		}

		const candidate = isAbsolute(target)
			? resolve(process.cwd(), `.${target}`)
			: resolve(dirname(file), target);
		if (!existsSync(candidate)) {
			failures.push(`${file}: ${rawTarget}`);
		}
	}

	const prose = source.replace(/```[\s\S]*?```/g, '');
	for (const match of prose.matchAll(/`([^`\n]+)`/g)) {
		const candidate = backtickedPath(match[1], file);
		if (candidate !== null && !existsSync(candidate)) {
			failures.push(`${file}: \`${match[1]}\``);
		}
	}
}

if (failures.length > 0) {
	console.error('Broken internal Markdown links:');
	for (const failure of failures) {
		console.error(`  ${failure}`);
	}
	process.exit(1);
}

console.log('Internal Markdown links are valid.');

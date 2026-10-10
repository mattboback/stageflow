import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { compareFindings, normalizeReport, runKnip } from '../check-knip.mjs';

const root = fileURLToPath(new URL('../../../', import.meta.url));
const finding = {
	workspace: 'services/scanner-runner',
	file: 'src/dead.ts',
	kind: 'files',
	symbol: ''
};
const allowance = { ...finding, owner: 'scanner-evidence-cleanup', reason: 'Scheduled evidence cleanup' };

test('exact allowances reject new findings and become stale after removal', () => {
	assert.deepEqual(compareFindings([finding], [allowance]), { unexpected: [], stale: [] });
	assert.deepEqual(compareFindings([], [allowance]).stale, [allowance]);
	const added = { ...finding, file: 'src/another.ts' };
	assert.deepEqual(compareFindings([finding, added], [allowance]).unexpected, [added]);
	const newSymbol = { ...finding, kind: 'exports', symbol: 'newDeadExport' };
	assert.deepEqual(compareFindings([finding, newSymbol], [allowance]).unexpected, [newSymbol]);
});

test('allowances require exact findings, cleanup ownership and a reason', () => {
	assert.throws(() => compareFindings([], [{ ...allowance, file: 'src/**' }]), /exact finding/);
	assert.throws(() => compareFindings([], [{ ...allowance, owner: '  ' }]), /cleanup owner/);
	assert.throws(() => compareFindings([], [{ ...allowance, owner: null }]), /cleanup owner/);
	assert.throws(() => compareFindings([], [{ ...allowance, reason: '  ' }]), /reason/);
	assert.throws(() => compareFindings([], [allowance, allowance]), /Duplicate/);
	assert.throws(
		() => normalizeReport('clients/web', { issues: [{ file: 'a.ts', unknown: [] }] }),
		/Unknown/
	);
});

for (const workspace of ['clients/web', 'services/scanner-runner']) {
	test(`${workspace}: installed production Knip detects an orphan and an unused export`, () => {
		const fixture = mkdtempSync(path.join(tmpdir(), 'stageflow-knip-'));
		try {
			mkdirSync(path.join(fixture, 'src'));
			writeFileSync(
				path.join(fixture, 'package.json'),
				JSON.stringify({ name: 'knip-gate-probe', private: true })
			);
			writeFileSync(
				path.join(fixture, 'knip.json'),
				JSON.stringify({
					entry: ['src/index.ts!'],
					project: ['src/**/*.ts!'],
					includeEntryExports: true
				})
			);
			writeFileSync(
				path.join(fixture, 'src/index.ts'),
				"import { used } from './live'; console.log(used);\n"
			);
			writeFileSync(
				path.join(fixture, 'src/live.ts'),
				'export const used = 1; export const unused = 2;\n'
			);
			writeFileSync(path.join(fixture, 'src/orphan.ts'), 'export const orphan = 3;\n');
			const binary = path.join(root, workspace, 'node_modules/knip/bin/knip.js');
			const findings = normalizeReport(workspace, runKnip(fixture, binary));
			assert.ok(findings.some((item) => item.file === 'src/orphan.ts' && item.kind === 'files'));
			assert.ok(
				findings.some(
					(item) =>
						item.file === 'src/live.ts' && item.kind === 'exports' && item.symbol === 'unused'
				)
			);
			assert.equal(compareFindings(findings, []).unexpected.length, 2);
			writeFileSync(path.join(fixture, 'src/live.ts'), 'export const used = 1;\n');
			rmSync(path.join(fixture, 'src/orphan.ts'));
			assert.deepEqual(normalizeReport(workspace, runKnip(fixture, binary)), []);
		} finally {
			rmSync(fixture, { recursive: true, force: true });
		}
	});
}

for (const workspace of ['clients/web', 'services/scanner-runner']) {
	test(`${workspace}: actual configuration includes production sources`, () => {
		const cwd = path.join(root, workspace);
		const source = workspace === 'clients/web' ? 'app' : 'src';
		const probeDir = mkdtempSync(path.join(cwd, source, 'knip-gate-probe-'));
		const probeFile = path.join(probeDir, 'orphan.ts');
		try {
			writeFileSync(probeFile, 'export const productionGraphCanary = true;\n');
			const findings = normalizeReport(workspace, runKnip(cwd));
			const canary = { workspace, file: path.relative(cwd, probeFile), kind: 'files', symbol: '' };
			const allowances = JSON.parse(
				readFileSync(path.join(root, 'devtools/scripts/knip-allowlist.json'), 'utf8')
			).filter((item) => item.workspace === workspace);
			const result = compareFindings(findings, allowances);
			assert.deepEqual(
				result.unexpected,
				[canary],
				'A missing/empty production graph must not pass'
			);
			assert.deepEqual(result.stale, []);
		} finally {
			rmSync(probeDir, { recursive: true, force: true });
		}
	});
}

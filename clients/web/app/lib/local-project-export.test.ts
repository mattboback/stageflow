import { describe, expect, it } from 'vitest';

import { loadAllScansFixture } from '../test/load-fixture';
import {
	LOCAL_PROJECT_EXPORT_SCHEMA,
	buildLocalProjectExport,
	parseLocalProjectExport
} from './local-project-export';
import { LOCAL_PROJECT_SCHEMA_VERSION, type LocalProject, type LocalRun } from './projects';

const project: LocalProject = {
	id: 'project-1',
	schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
	name: 'Example',
	configuration: {
		urls: ['https://example.com'],
		scanners: [{ id: 'axe', enabled: true }],
		browser: 'chromium',
		highlightStyle: 'solid'
	},
	createdAt: '2026-08-13T00:00:00.000Z',
	updatedAt: '2026-08-13T00:00:00.000Z'
};

describe('local project export', () => {
	it('round-trips a project with a stored report', () => {
		const report = loadAllScansFixture();
		const run: LocalRun = {
			jobId: report.meta.jobId,
			projectId: project.id,
			configFingerprint: 'abc',
			status: 'complete',
			createdAt: '2026-08-13T00:00:00.000Z',
			report
		};
		const exported = buildLocalProjectExport(project, null, [run]);
		expect(exported.schema).toBe(LOCAL_PROJECT_EXPORT_SCHEMA);
		expect(parseLocalProjectExport(exported)?.runs[0]?.report?.meta.jobId).toBe(report.meta.jobId);
	});

	it('rejects a foreign payload', () => {
		expect(parseLocalProjectExport({ schema: 'nope' })).toBeNull();
	});

	it('rejects a schema-matching payload with no configuration', () => {
		expect(
			parseLocalProjectExport({
				schema: LOCAL_PROJECT_EXPORT_SCHEMA,
				project: { id: project.id, name: project.name },
				runs: []
			})
		).toBeNull();
	});

	it('rejects a run that is missing a jobId', () => {
		expect(
			parseLocalProjectExport({
				schema: LOCAL_PROJECT_EXPORT_SCHEMA,
				project,
				runs: [
					{
						projectId: project.id,
						configFingerprint: 'abc',
						status: 'complete',
						createdAt: '2026-08-13T00:00:00.000Z'
					}
				]
			})
		).toBeNull();
	});
});

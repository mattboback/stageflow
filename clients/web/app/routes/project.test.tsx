import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Project from './project';
import { loadAllScansFixture } from '../test/load-fixture';
import {
	getLocalProject,
	getLocalBaseline,
	listLocalProjectRuns,
	importCliReport
} from '../lib/local-project-store';

vi.mock('../lib/local-project-store', () => ({
	getLocalProject: vi.fn(),
	getLocalBaseline: vi.fn(),
	listLocalProjectRuns: vi.fn(),
	importCliReport: vi.fn(),
	exportLocalProject: vi.fn()
}));
const project = {
	id: 'site',
	schemaVersion: 1 as const,
	name: 'Preview site',
	configuration: {
		urls: ['https://preview.example.com/', 'https://preview.example.com/about'],
		scanners: [{ id: 'axe', enabled: true }],
		browser: 'chromium' as const,
		highlightStyle: 'solid' as const,
		profile: 'staging' as const
	},
	createdAt: '2026-10-13',
	updatedAt: '2026-10-13'
};
function mount() {
	const Stub = createRoutesStub([{ path: '/projects/:projectId', Component: Project }]);
	render(<Stub initialEntries={['/projects/site']} />);
}
beforeEach(() => {
	vi.resetAllMocks();
	vi.mocked(getLocalProject).mockResolvedValue(project);
	vi.mocked(getLocalBaseline).mockResolvedValue(null);
	vi.mocked(listLocalProjectRuns).mockResolvedValue([]);
});
describe('project workspace', () => {
	it('does not show disabled scanners as selected', async () => {
		vi.mocked(getLocalProject).mockResolvedValue({
			...project,
			configuration: { ...project.configuration, scanners: [{ id: 'axe', enabled: false }] }
		});
		mount();
		await screen.findByText(/No scanners enabled/);
	});
	it('does not compare legacy targeted verification reports as full scans', async () => {
		const report = loadAllScansFixture();
		report.meta.scanContext = {
			expectedPages: [],
			identityScope: 'path',
			scannerConfigFingerprints: {},
			purpose: 'verification',
			authenticated: false,
			inputType: 'urls'
		};
		vi.mocked(listLocalProjectRuns).mockResolvedValue([
			{
				jobId: report.meta.jobId,
				projectId: 'site',
				configFingerprint: 'same',
				status: 'complete',
				createdAt: '2026-10-13',
				report
			}
		]);
		mount();
		await screen.findByRole('heading', { name: 'Preview site' });
		expect(screen.queryByRole('link', { name: 'Review findings' })).toBeNull();
		expect(screen.getByRole('link', { name: 'Open report' })).toBeDefined();
	});

	it('shows frozen pages, profile and explicit next actions', async () => {
		mount();
		await screen.findByRole('heading', { name: 'Preview site' });
		expect(screen.getByText('Staging')).toBeDefined();
		expect(screen.getByRole('heading', { name: '2 saved pages' })).toBeDefined();
		expect(screen.getByRole('link', { name: 'Run scan' }).getAttribute('href')).toBe(
			'/playground?project=site'
		);
		expect(screen.getByText('No baseline yet')).toBeDefined();
		expect(screen.getByText(/Browser data can be cleared/)).toBeDefined();
	});
	it('retains links to submitted and failed runs after reopening', async () => {
		vi.mocked(listLocalProjectRuns).mockResolvedValue([
			{
				jobId: 'pending',
				projectId: 'site',
				configFingerprint: 'same',
				status: 'submitted',
				createdAt: '2026-10-13'
			},
			{
				jobId: 'failed',
				projectId: 'site',
				configFingerprint: 'same',
				status: 'failed',
				createdAt: '2026-10-12'
			}
		]);
		mount();
		await screen.findByText(/Submitted — open to follow progress/);
		expect(screen.getByText(/Failed — review scan details/)).toBeDefined();
		expect(
			screen
				.getAllByRole('link', { name: 'View progress' })
				.map((link) => link.getAttribute('href'))
		).toEqual(['/scan/pending?project=site', '/scan/failed?project=site']);
	});
	it('surfaces storage errors without claiming the project is empty', async () => {
		vi.mocked(getLocalProject).mockRejectedValue(new Error('Storage quota exceeded'));
		mount();
		expect((await screen.findByRole('alert')).textContent).toContain('Storage quota exceeded');
	});
	it('explains a missing browser-local project', async () => {
		vi.mocked(getLocalProject).mockResolvedValue(null);
		mount();
		await screen.findByRole('heading', { name: 'Project unavailable' });
		expect(screen.queryByRole('link', { name: 'Run scan' })).toBeNull();
	});
	it('imports a CLI file into the selected project and surfaces validation failures', async () => {
		vi.mocked(importCliReport).mockRejectedValue(new Error('Scan configuration differs'));
		mount();
		await screen.findByRole('heading', { name: 'Preview site' });
		const file = new File(['{}'], 'report.json', { type: 'application/json' });
		fireEvent.change(screen.getByLabelText('CLI report file'), { target: { files: [file] } });
		await waitFor(() => expect(importCliReport).toHaveBeenCalledWith('site', {}));
		expect((await screen.findByRole('alert')).textContent).toContain('Scan configuration differs');
	});
});

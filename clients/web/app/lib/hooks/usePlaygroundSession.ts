import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router';

import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '../api/client';
import { getLocalProject, saveLocalProject, saveLocalRun } from '../local-project-store';
import {
	buildAiNavigatorConfig,
	buildFormAuthConfig,
	DEFAULT_AI_CONFIG,
	estimateScanRuntime,
	isAuthConfigComplete,
	validatePlaygroundConfiguration,
	validateZipUploadFile,
	type AiConfigState,
	type AuthFormConfig
} from '../playground/playground-utils';
import {
	fingerprintProjectConfiguration,
	type LocalProject,
	type LocalProjectConfiguration
} from '../projects';
import type { ScannerDefinition, ScannerSelection } from '../types/scan';
import { normalizeUrlInput, validateHttpUrl } from '../url';

export type PlaygroundMode = 'url' | 'zip';

/** Alias kept so the extracted body below reads unchanged. */
type Mode = PlaygroundMode;

export interface PlaygroundSessionOptions {
	projectId: string | null;
	seedUrl: string | null;
}

function mergeProjectSelections(
	catalog: ScannerDefinition[],
	stored: ScannerSelection[]
): ScannerSelection[] {
	if (stored.length === 0) return getDefaultScannerSelections(catalog);
	const storedById = new Map(stored.map((selection) => [selection.id, selection]));
	return catalog.map((scanner) => storedById.get(scanner.id) ?? { id: scanner.id, enabled: false });
}

function preserveUnavailableProjectSelections(
	catalog: ScannerDefinition[],
	current: ScannerSelection[],
	stored: ScannerSelection[]
): ScannerSelection[] {
	const availableIds = new Set(catalog.map((scanner) => scanner.id));
	return [...current, ...stored.filter((selection) => !availableIds.has(selection.id))];
}

function aiStateFromSelections(selections: ScannerSelection[]): AiConfigState {
	const config = selections.find((selection) => selection.id === 'ai-navigator')?.config;
	const goal = config?.goal as Record<string, unknown> | undefined;
	const vision = config?.vision as Record<string, unknown> | undefined;
	const criteria = Array.isArray(goal?.successCriteria)
		? goal.successCriteria.flatMap((criterion) => {
				if (!criterion || typeof criterion !== 'object') return [];
				const candidate = criterion as Record<string, unknown>;
				return typeof candidate.type === 'string' && typeof candidate.value === 'string'
					? [{ type: candidate.type, value: candidate.value }]
					: [];
			})
		: [];

	return {
		objective: typeof goal?.objective === 'string' ? goal.objective : DEFAULT_AI_CONFIG.objective,
		maxSteps: typeof goal?.maxSteps === 'number' ? goal.maxSteps : DEFAULT_AI_CONFIG.maxSteps,
		maxWallTimeMs:
			typeof goal?.maxWallTimeMs === 'number'
				? goal.maxWallTimeMs
				: DEFAULT_AI_CONFIG.maxWallTimeMs,
		model: typeof vision?.model === 'string' ? vision.model : DEFAULT_AI_CONFIG.model,
		inputValues: [],
		successCriteria: criteria
	};
}

/**
 * All configure-page state: the scanner catalog, the target list, auth and AI
 * settings, local-project persistence, and submission.
 *
 * Extracted from routes/playground.tsx, where a single component held seventeen
 * useState calls and roughly four hundred lines of JSX. The route now renders; this
 * decides. Same pattern as useScanMonitor.
 *
 * The caller is responsible for keying the component on the project query so a
 * project switch discards credentials and draft values — see the route.
 */
export function usePlaygroundSession({ projectId, seedUrl }: PlaygroundSessionOptions) {
	const navigate = useNavigate();

	const [catalog, setCatalog] = useState<ScannerDefinition[]>([]);
	const [selections, setSelections] = useState<ScannerSelection[]>([]);
	const [catalogError, setCatalogError] = useState<string | null>(null);
	const [catalogLoading, setCatalogLoading] = useState(true);
	const [project, setProject] = useState<LocalProject | null>(null);
	const [projectName, setProjectName] = useState('');
	const [savingProject, setSavingProject] = useState(false);
	const [projectNotice, setProjectNotice] = useState<string | null>(null);

	const [mode, setMode] = useState<Mode>('url');
	const [urls, setUrls] = useState<string[]>(() => {
		return seedUrl ? [seedUrl, ''] : ['https://example.com', ''];
	});
	const [file, setFile] = useState<File | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);

	const [error, setError] = useState<string | null>(null);
	const [urlRowErrors, setUrlRowErrors] = useState<Record<number, string>>({});
	const [submitting, setSubmitting] = useState(false);

	const [authConfig, setAuthConfig] = useState<AuthFormConfig>({
		enabled: false,
		loginUrl: '',
		username: '',
		password: '',
		usernameSelector: '',
		passwordSelector: '',
		submitSelector: '',
		successStrategy: 'auto',
		successSelector: ''
	});
	const isAuthValid = isAuthConfigComplete(authConfig);

	const [aiConfig, setAiConfig] = useState<AiConfigState>(DEFAULT_AI_CONFIG);
	const isAiNavigatorEnabled = selections.some((s) => s.id === 'ai-navigator' && s.enabled);

	useEffect(() => {
		const controller = new AbortController();
		Promise.all([
			fetchScanners(controller.signal),
			projectId ? getLocalProject(projectId) : Promise.resolve(null)
		])
			.then(([res, storedProject]) => {
				if (controller.signal.aborted) return;
				setCatalog(res.scanners);
				if (projectId && !storedProject) {
					setProject(null);
					setSelections(getDefaultScannerSelections(res.scanners));
					setError('That local project was not found. It may have been deleted in this browser.');
				} else if (storedProject) {
					setProject(storedProject);
					setProjectName(storedProject.name);
					setMode('url');
					setUrls([...storedProject.configuration.urls, '']);
					setSelections(mergeProjectSelections(res.scanners, storedProject.configuration.scanners));
					setAiConfig(aiStateFromSelections(storedProject.configuration.scanners));
				} else {
					setSelections(getDefaultScannerSelections(res.scanners));
				}
				setCatalogError(null);
			})
			.catch((err: unknown) => {
				if (controller.signal.aborted) return;
				setCatalogError(err instanceof Error ? err.message : 'Failed to load scanners.');
			})
			.finally(() => {
				if (!controller.signal.aborted) setCatalogLoading(false);
			});
		return () => controller.abort();
	}, [projectId]);

	const enabledById = useMemo(() => {
		const map = new Map<string, boolean>();
		for (const s of selections) map.set(s.id, s.enabled);
		return map;
	}, [selections]);

	const armed = selections.filter((s) => s.enabled).length;
	const total = selections.length;

	function toggleScanner(id: string) {
		setSelections((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s)));
	}

	function setAllScanners(enabled: boolean) {
		setSelections((prev) => prev.map((s) => ({ ...s, enabled })));
	}

	function updateUrl(index: number, value: string) {
		setUrls((prev) => prev.map((u, i) => (i === index ? value : u)));
		setUrlRowErrors((prev) => {
			if (!(index in prev)) return prev;
			const next = { ...prev };
			delete next[index];
			return next;
		});
	}

	function addUrlRow() {
		setUrls((prev) => [...prev, '']);
	}

	function removeUrlRow(index: number) {
		setUrls((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
		setUrlRowErrors({});
	}

	function onFilePicked(f: File | null) {
		if (!f) {
			setFile(null);
			return;
		}
		const problem = validateZipUploadFile(f);
		if (problem) {
			setError(problem);
			setFile(null);
			return;
		}
		setError(null);
		setFile(f);
	}

	function selectionsForCurrentConfiguration(): ScannerSelection[] {
		return isAiNavigatorEnabled
			? selections.map((selection) =>
					selection.id === 'ai-navigator'
						? { ...selection, config: buildAiNavigatorConfig(aiConfig) }
						: selection
				)
			: selections;
	}

	function selectionsForProjectPersistence(current: ScannerSelection[]): ScannerSelection[] {
		return project
			? preserveUnavailableProjectSelections(catalog, current, project.configuration.scanners)
			: current;
	}

	function collectValidUrls(): { valid: string[]; rowErrors: Record<number, string> } {
		const rowErrors: Record<number, string> = {};
		const valid: string[] = [];
		urls.forEach((raw, index) => {
			const normalized = normalizeUrlInput(raw);
			if (!normalized) return;
			const result = validateHttpUrl(normalized);
			if (result.ok) {
				valid.push(result.url);
			} else {
				rowErrors[index] = result.reason;
			}
		});
		return { valid, rowErrors };
	}

	function currentProjectConfiguration(
		validUrls: string[],
		scanners: ScannerSelection[]
	): LocalProjectConfiguration {
		return {
			urls: validUrls,
			scanners,
			browser: project?.configuration.browser ?? 'chromium',
			highlightStyle: project?.configuration.highlightStyle ?? 'solid'
		};
	}

	async function saveProjectConfiguration() {
		if (!project) return;
		setProjectNotice(null);
		setError(null);
		const { valid, rowErrors } = collectValidUrls();
		setUrlRowErrors(rowErrors);
		if (Object.keys(rowErrors).length > 0) return;
		if (valid.length === 0) {
			setError('Enter at least one URL before saving this project.');
			return;
		}
		if (!projectName.trim()) {
			setError('Give this project a name before saving.');
			return;
		}

		setSavingProject(true);
		try {
			const currentSelections = selectionsForCurrentConfiguration();
			const saved = await saveLocalProject({
				...project,
				name: projectName,
				configuration: currentProjectConfiguration(
					valid,
					selectionsForProjectPersistence(currentSelections)
				)
			});
			setProject(saved);
			setProjectName(saved.name);
			setProjectNotice('Project configuration saved locally.');
		} catch (saveError) {
			setError(saveError instanceof Error ? saveError.message : 'Could not save this project.');
		} finally {
			setSavingProject(false);
		}
	}

	async function runScan() {
		setError(null);
		setUrlRowErrors(validation.urlRowErrors);
		if (!validation.ready) {
			setError(validation.error);
			if (validation.focusId) {
				requestAnimationFrame(() => document.getElementById(validation.focusId ?? '')?.focus());
			}
			return;
		}

		const validUrls = validation.validUrls;
		const auth = mode === 'url' ? buildFormAuthConfig(authConfig) : null;
		const scannersForSubmit = selectionsForCurrentConfiguration();

		setSubmitting(true);
		try {
			let configFingerprint: string | null = null;
			if (project) {
				const runConfiguration = currentProjectConfiguration(validUrls, scannersForSubmit);
				const savedConfiguration = currentProjectConfiguration(
					validUrls,
					selectionsForProjectPersistence(scannersForSubmit)
				);
				const saved = await saveLocalProject({
					...project,
					name: projectName,
					configuration: savedConfiguration
				});
				setProject(saved);
				configFingerprint = await fingerprintProjectConfiguration(runConfiguration);
			}
			const { job_id } = await submitScanJob({
				mode,
				file,
				urls: validUrls,
				scanners: scannersForSubmit,
				highlightStyle: project?.configuration.highlightStyle ?? 'solid',
				engine: project?.configuration.browser ?? 'chromium',
				auth
			});
			if (project && configFingerprint) {
				try {
					await saveLocalRun({
						jobId: job_id,
						projectId: project.id,
						configFingerprint,
						status: 'submitted',
						createdAt: new Date().toISOString()
					});
				} catch {
					// The scan is already running. Preserve project context in the URL so the
					// report can still offer a recovery path if browser storage was exhausted.
				}
			}
			const projectQuery = project ? `?project=${encodeURIComponent(project.id)}` : '';
			void navigate(`/scan/${job_id}${projectQuery}`);
		} catch (err: unknown) {
			setError(err instanceof Error ? err.message : 'Scan failed. Please try again.');
			setSubmitting(false);
		}
	}

	const targetCount = mode === 'url' ? urls.filter((u) => u.trim()).length : file ? 1 : 0;
	const validation = validatePlaygroundConfiguration({
		mode,
		urls,
		file,
		selections,
		auth: authConfig,
		ai: aiConfig,
		aiEnabled: isAiNavigatorEnabled,
		catalogLoading,
		catalogError,
		projectName: project ? projectName : undefined
	});
	const ready = validation.ready;
	const readyDetail = validation.message;
	const runtimeEstimate = estimateScanRuntime(catalog, selections, targetCount, mode);
	return {
		// Scanner catalog
		catalog,
		selections,
		setSelections,
		catalogError,
		catalogLoading,
		enabledById,
		toggleScanner,
		setAllScanners,
		isAiNavigatorEnabled,
		/** Enabled scanner count and catalog size, for the "3 of 8 armed" summary. */
		armed,
		total,

		// Targets
		mode,
		setMode,
		urls,
		updateUrl,
		addUrlRow,
		removeUrlRow,
		targetCount,
		file,
		fileInputRef,
		onFilePicked,

		// Auth and AI configuration
		authConfig,
		setAuthConfig,
		isAuthValid,
		aiConfig,
		setAiConfig,

		// Local project
		project,
		projectName,
		setProjectName,
		savingProject,
		projectNotice,
		saveProjectConfiguration,

		// Submission and validation
		error,
		urlRowErrors,
		submitting,
		runScan,
		ready,
		readyDetail,
		runtimeEstimate
	};
}

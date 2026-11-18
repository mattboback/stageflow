import { parsePortableAudit } from './portable-audit';
import { buildImportedCliRun, parseCliReport } from './cli-report-import';
import {
	createReviewVerdictStore,
	mergeReviewChanges,
	type ReviewVerdictStore
} from './report/review-verdict-store';
import {
	normalizeReviewVerdicts,
	REVIEW_VERDICTS_STORAGE_KEY,
	LEGACY_CONTRAST_VERDICTS_STORAGE_KEY,
	type ReviewVerdictsByJob
} from './report/review-verdict';
import {
	buildLocalProjectExport,
	parseLocalProjectExport,
	type LocalProjectExport
} from './local-project-export';
import {
	LOCAL_PROJECT_SCHEMA_VERSION,
	sanitizeProjectConfiguration,
	fingerprintProjectConfiguration,
	type LocalBaseline,
	type LocalProject,
	type LocalProjectConfiguration,
	type LocalRun
} from './projects';

const DATABASE_NAME = 'stageflow-local-projects';
const DATABASE_VERSION = 2;
const PROJECTS_STORE = 'projects';
const BASELINES_STORE = 'baselines';
const RUNS_STORE = 'projectRuns';
const REVIEWS_STORE = 'reviews';
const MAX_RECENT_RUNS = 10;

export class LocalProjectStoreUnavailableError extends Error {
	constructor(message = 'Local projects are unavailable in this browser.') {
		super(message);
		this.name = 'LocalProjectStoreUnavailableError';
	}
}

function requireIndexedDb(): IDBFactory {
	if (typeof indexedDB === 'undefined') throw new LocalProjectStoreUnavailableError();
	return indexedDB;
}

function requestResult<T>(request: IDBRequest<T>): Promise<T> {
	return new Promise((resolve, reject) => {
		request.onsuccess = () => resolve(request.result);
		request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed.'));
	});
}

function transactionComplete(transaction: IDBTransaction): Promise<void> {
	return new Promise((resolve, reject) => {
		transaction.oncomplete = () => resolve();
		transaction.onabort = () =>
			reject(transaction.error ?? new Error('IndexedDB transaction aborted.'));
		transaction.onerror = () =>
			reject(transaction.error ?? new Error('IndexedDB transaction failed.'));
	});
}

async function openDatabase(): Promise<IDBDatabase> {
	const request = requireIndexedDb().open(DATABASE_NAME, DATABASE_VERSION);
	request.onupgradeneeded = () => {
		const database = request.result;
		if (!database.objectStoreNames.contains(PROJECTS_STORE)) {
			const projects = database.createObjectStore(PROJECTS_STORE, { keyPath: 'id' });
			projects.createIndex('updatedAt', 'updatedAt');
		}
		if (!database.objectStoreNames.contains(BASELINES_STORE)) {
			database.createObjectStore(BASELINES_STORE, { keyPath: 'projectId' });
		}
		if (!database.objectStoreNames.contains(RUNS_STORE)) {
			const runs = database.createObjectStore(RUNS_STORE, { keyPath: ['projectId', 'jobId'] });
			runs.createIndex('jobId', 'jobId');
			runs.createIndex('projectId', 'projectId');
			runs.createIndex('createdAt', 'createdAt');
		}
		if (!database.objectStoreNames.contains(REVIEWS_STORE))
			database.createObjectStore(REVIEWS_STORE);
		if (database.objectStoreNames.contains('runs') && request.transaction) {
			const transaction = request.transaction;
			const cursor = transaction.objectStore('runs').openCursor();
			cursor.onsuccess = () => {
				if (!cursor.result) {
					database.deleteObjectStore('runs');
					return;
				}
				transaction.objectStore(RUNS_STORE).put(cursor.result.value);
				cursor.result.continue();
			};
		}
	};
	return requestResult(request);
}

async function withDatabase<T>(callback: (database: IDBDatabase) => Promise<T>): Promise<T> {
	let database: IDBDatabase | null = null;
	try {
		database = await openDatabase();
		return await callback(database);
	} catch (error) {
		if (error instanceof LocalProjectStoreUnavailableError) throw error;
		if (error instanceof DOMException && error.name === 'QuotaExceededError') {
			throw new LocalProjectStoreUnavailableError(
				'This browser has no storage space available for local projects.'
			);
		}
		throw new LocalProjectStoreUnavailableError(
			error instanceof Error ? error.message : 'Local project storage failed.'
		);
	} finally {
		database?.close();
	}
}

function newProjectId(): string {
	return crypto.randomUUID();
}

export async function listLocalProjects(): Promise<LocalProject[]> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(PROJECTS_STORE, 'readonly');
		const projects = await requestResult(
			transaction.objectStore(PROJECTS_STORE).getAll() as IDBRequest<LocalProject[]>
		);
		await transactionComplete(transaction);
		return projects.sort((left, right) => right.updatedAt.localeCompare(left.updatedAt));
	});
}

export async function getLocalProject(projectId: string): Promise<LocalProject | null> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(PROJECTS_STORE, 'readonly');
		const project = await requestResult(
			transaction.objectStore(PROJECTS_STORE).get(projectId) as IDBRequest<LocalProject | undefined>
		);
		await transactionComplete(transaction);
		return project ?? null;
	});
}

export async function createLocalProject(
	name: string,
	configuration: LocalProjectConfiguration
): Promise<LocalProject> {
	const now = new Date().toISOString();
	const project: LocalProject = {
		id: newProjectId(),
		schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
		name: name.trim(),
		configuration: sanitizeProjectConfiguration(configuration),
		createdAt: now,
		updatedAt: now
	};

	await withDatabase(async (database) => {
		const transaction = database.transaction(PROJECTS_STORE, 'readwrite');
		transaction.objectStore(PROJECTS_STORE).add(project);
		await transactionComplete(transaction);
	});
	return project;
}

export async function saveLocalProject(project: LocalProject): Promise<LocalProject> {
	const safeProject: LocalProject = {
		...project,
		name: project.name.trim(),
		configuration: sanitizeProjectConfiguration(project.configuration),
		updatedAt: new Date().toISOString()
	};

	await withDatabase(async (database) => {
		const transaction = database.transaction(PROJECTS_STORE, 'readwrite');
		transaction.objectStore(PROJECTS_STORE).put(safeProject);
		await transactionComplete(transaction);
	});
	return safeProject;
}

export async function deleteLocalProject(projectId: string): Promise<void> {
	await withDatabase(async (database) => {
		const transaction = database.transaction(
			[PROJECTS_STORE, BASELINES_STORE, RUNS_STORE, REVIEWS_STORE],
			'readwrite'
		);
		transaction.objectStore(PROJECTS_STORE).delete(projectId);
		transaction.objectStore(BASELINES_STORE).delete(projectId);
		transaction.objectStore(REVIEWS_STORE).delete(projectId);
		projectReviewStores.delete(projectId);
		projectReviewSubscriptions.get(projectId)?.();
		projectReviewSubscriptions.delete(projectId);

		const runIndex = transaction.objectStore(RUNS_STORE).index('projectId');
		const cursorRequest = runIndex.openKeyCursor(IDBKeyRange.only(projectId));
		cursorRequest.onsuccess = () => {
			const cursor = cursorRequest.result;
			if (!cursor) return;
			transaction.objectStore(RUNS_STORE).delete(cursor.primaryKey);
			cursor.continue();
		};
		await transactionComplete(transaction);
	});
}

export async function getLocalBaseline(projectId: string): Promise<LocalBaseline | null> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(BASELINES_STORE, 'readonly');
		const baseline = await requestResult(
			transaction.objectStore(BASELINES_STORE).get(projectId) as IDBRequest<
				LocalBaseline | undefined
			>
		);
		await transactionComplete(transaction);
		return baseline ?? null;
	});
}

export async function saveLocalBaseline(baseline: LocalBaseline): Promise<void> {
	await withDatabase(async (database) => {
		const transaction = database.transaction(BASELINES_STORE, 'readwrite');
		transaction.objectStore(BASELINES_STORE).put(baseline);
		await transactionComplete(transaction);
	});
}

export async function getLocalRun(jobId: string, projectId?: string): Promise<LocalRun | null> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(RUNS_STORE, 'readonly');
		const run = await requestResult(
			(projectId
				? transaction.objectStore(RUNS_STORE).get([projectId, jobId])
				: transaction.objectStore(RUNS_STORE).index('jobId').get(jobId)) as IDBRequest<
				LocalRun | undefined
			>
		);
		await transactionComplete(transaction);
		return run ?? null;
	});
}

export async function listLocalProjectRuns(projectId: string): Promise<LocalRun[]> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(RUNS_STORE, 'readonly');
		const runs = await requestResult(
			transaction.objectStore(RUNS_STORE).index('projectId').getAll(projectId) as IDBRequest<
				LocalRun[]
			>
		);
		await transactionComplete(transaction);
		return runs.sort((left, right) => right.createdAt.localeCompare(left.createdAt));
	});
}

export async function saveLocalRun(run: LocalRun): Promise<void> {
	await withDatabase(async (database) => {
		const transaction = database.transaction([RUNS_STORE, PROJECTS_STORE], 'readwrite');
		const runStore = transaction.objectStore(RUNS_STORE);
		runStore.put(run);

		const existing = await requestResult(
			runStore.index('projectId').getAll(run.projectId) as IDBRequest<LocalRun[]>
		);
		existing
			.filter((candidate) => candidate.jobId !== run.jobId)
			.sort((left, right) => right.createdAt.localeCompare(left.createdAt))
			.slice(MAX_RECENT_RUNS - 1)
			.forEach((candidate) => runStore.delete([candidate.projectId, candidate.jobId]));

		const projectStore = transaction.objectStore(PROJECTS_STORE);
		const project = await requestResult(
			projectStore.get(run.projectId) as IDBRequest<LocalProject | undefined>
		);
		if (project) {
			projectStore.put({ ...project, updatedAt: new Date().toISOString() });
		}
		await transactionComplete(transaction);
	});
}

export async function exportLocalProject(projectId: string): Promise<LocalProjectExport> {
	const project = await getLocalProject(projectId);
	if (!project) {
		throw new LocalProjectStoreUnavailableError('That local project is no longer in this browser.');
	}

	const [baseline, runs, reviews] = await Promise.all([
		getLocalBaseline(projectId),
		listLocalProjectRuns(projectId),
		getLocalProjectReviews(projectId)
	]);
	return buildLocalProjectExport(project, baseline, runs, reviews);
}

/** Keep CLI imports local: envelope API/artifact URLs are never fetched. */
export async function importCliReport(projectId: string, value: unknown): Promise<LocalRun> {
	const report = parseCliReport(value);
	const project = await getLocalProject(projectId);
	if (!project)
		throw new LocalProjectStoreUnavailableError('That project is no longer in this browser.');
	const run = await buildImportedCliRun(project, report);
	return withDatabase(async (database) => {
		// Wait for the transaction, not just put() success: quota/abort failures must be visible.
		// https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Using_IndexedDB#adding_data_to_the_database
		const transaction = database.transaction([PROJECTS_STORE, RUNS_STORE], 'readwrite');
		const completed = transactionComplete(transaction);
		const projects = transaction.objectStore(PROJECTS_STORE);
		const runs = transaction.objectStore(RUNS_STORE);
		try {
			const current = await requestResult(
				projects.get(projectId) as IDBRequest<LocalProject | undefined>
			);
			if (
				!current ||
				JSON.stringify(current.configuration) !== JSON.stringify(project.configuration)
			)
				throw new Error('The project settings changed during import. Try again.');
			const existing = await requestResult(
				runs.get([projectId, run.jobId]) as IDBRequest<LocalRun | undefined>
			);
			if (existing?.report) {
				await completed;
				return existing;
			}
			runs.put(run);
			const history = await requestResult(
				runs.index('projectId').getAll(projectId) as IDBRequest<LocalRun[]>
			);
			history
				.sort((left, right) => right.createdAt.localeCompare(left.createdAt))
				.slice(MAX_RECENT_RUNS)
				.forEach((old) => runs.delete([old.projectId, old.jobId]));
			projects.put({ ...current, updatedAt: new Date().toISOString() });
			await completed;
			return run;
		} catch (error) {
			try {
				transaction.abort();
			} catch {
				/* It may have already aborted. */
			}
			await completed.catch(() => undefined);
			throw error;
		}
	});
}

export async function importLocalProject(value: unknown): Promise<LocalProject> {
	let parsed = parseLocalProjectExport(value);
	const audit = parsed ? null : parsePortableAudit(value);
	if (audit) {
		if (audit.baseline?.meta.scanContext?.purpose === 'verification')
			throw new LocalProjectStoreUnavailableError(
				'Targeted verification cannot become a complete baseline.'
			);
		const now = new Date().toISOString();
		const configuration = audit.configuration ?? {
			urls: audit.report.pages.map((page) => page.url),
			scanners: audit.report.scanners.map((scanner) => ({ id: scanner.id, enabled: true })),
			browser: 'chromium' as const,
			highlightStyle: 'solid' as const,
			profile: audit.report.meta.profile ?? 'production'
		};
		const project: LocalProject = {
			id: 'import',
			schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
			name: `Imported audit: ${audit.report.meta.baseUrl ?? audit.jobId}`,
			configuration,
			createdAt: now,
			updatedAt: now
		};
		const configFingerprint = await fingerprintProjectConfiguration(configuration);
		const importedReports =
			audit.baseline && audit.baseline.meta.jobId !== audit.report.meta.jobId
				? [audit.baseline, audit.report]
				: [audit.report];
		const runs: LocalRun[] = importedReports.map((report) => ({
			projectId: project.id,
			jobId: report.meta.jobId,
			configFingerprint,
			status: 'complete',
			createdAt: report.meta.scannedAt ?? now,
			report,
			archived: true,
			...(report === audit.report
				? {
						screenshots: audit.screenshots,
						...(audit.verification?.jobId === report.meta.jobId
							? { verification: audit.verification }
							: {})
					}
				: {})
		}));
		parsed = buildLocalProjectExport(
			project,
			audit.baseline
				? {
						projectId: project.id,
						jobId: audit.baseline.meta.jobId,
						configFingerprint,
						report: audit.baseline,
						createdAt: now
					}
				: null,
			runs,
			audit.reviews
		);
	}
	if (!parsed) {
		throw new LocalProjectStoreUnavailableError(
			'That file is not a StageFlow project or portable audit JSON.'
		);
	}

	const now = new Date().toISOString();
	const project: LocalProject = {
		...parsed.project,
		id: newProjectId(),
		schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
		configuration: sanitizeProjectConfiguration(parsed.project.configuration),
		updatedAt: now
	};

	await withDatabase(async (database) => {
		const transaction = database.transaction(
			[PROJECTS_STORE, BASELINES_STORE, RUNS_STORE, REVIEWS_STORE],
			'readwrite'
		);
		// A fresh project ID preserves existing projects and their run history.
		transaction.objectStore(REVIEWS_STORE).put(parsed.reviews, project.id);
		const baselines = transaction.objectStore(BASELINES_STORE);
		const runs = transaction.objectStore(RUNS_STORE);
		transaction.objectStore(PROJECTS_STORE).put(project);
		if (parsed.baseline) baselines.put({ ...parsed.baseline, projectId: project.id });
		for (const run of parsed.runs) runs.put({ ...run, projectId: project.id, archived: true });
		await transactionComplete(transaction);
	});

	return project;
}

export async function getLocalProjectReviews(projectId: string): Promise<ReviewVerdictsByJob> {
	const cached = projectReviewStores.get(projectId);
	if (cached) return cached.getSnapshot();
	return readProjectReviewsFromDatabase(projectId);
}

async function readProjectReviewsFromDatabase(projectId: string): Promise<ReviewVerdictsByJob> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(REVIEWS_STORE, 'readonly');
		const value = await requestResult(
			transaction.objectStore(REVIEWS_STORE).get(projectId) as IDBRequest<unknown>
		);
		await transactionComplete(transaction);
		return normalizeReviewVerdicts(value);
	});
}

const projectReviewStores = new Map<string, ReviewVerdictStore>();
const projectReviewSubscriptions = new Map<string, () => void>();
let reviewBroadcast: BroadcastChannel | null | undefined;
function getReviewBroadcast(): BroadcastChannel | null {
	if (reviewBroadcast !== undefined) return reviewBroadcast;
	try {
		reviewBroadcast =
			typeof window !== 'undefined' && typeof window.BroadcastChannel === 'function'
				? new window.BroadcastChannel('stageflow-project-reviews')
				: null;
	} catch {
		reviewBroadcast = null;
	}
	return reviewBroadcast;
}
export async function loadProjectReviewStore(
	projectId: string,
	onError?: (error: unknown) => void
): Promise<ReviewVerdictStore> {
	const cached = projectReviewStores.get(projectId);
	if (cached) return cached;
	const [saved, runs, baseline] = await Promise.all([
		getLocalProjectReviews(projectId),
		listLocalProjectRuns(projectId),
		getLocalBaseline(projectId)
	]);
	const existingStore = projectReviewStores.get(projectId);
	if (existingStore) return existingStore;
	let historical: ReviewVerdictsByJob = {};
	try {
		const raw =
			typeof window !== 'undefined'
				? (window.localStorage.getItem(REVIEW_VERDICTS_STORAGE_KEY) ??
					window.localStorage.getItem(LEGACY_CONTRAST_VERDICTS_STORAGE_KEY))
				: null;
		const old = normalizeReviewVerdicts(raw ? JSON.parse(raw) : {});
		const jobs = new Set(runs.map((run) => run.jobId));
		if (baseline) jobs.add(baseline.jobId);
		historical = Object.fromEntries(Object.entries(old).filter(([jobId]) => jobs.has(jobId)));
	} catch {
		/* Existing decisions remain in their original localStorage. */
	}
	const initial = mergeReviewChanges(historical, {}, saved);
	let saving: Promise<void> = Promise.resolve();
	let previous = saved;
	let revision = 0;
	const store = createReviewVerdictStore(
		null,
		() => new Date().toISOString(),
		initial,
		(reviews) => {
			const before = previous;
			previous = reviews;
			const currentRevision = ++revision;
			saving = saving
				.then(async () => {
					const merged = await persistProjectReviewChanges(projectId, before, reviews);
					try {
						getReviewBroadcast()?.postMessage({ projectId });
					} catch {
						/* Persistence does not depend on cross-tab notifications. */
					}
					if (currentRevision === revision) {
						previous = merged;
						store.sync(JSON.stringify(merged));
					}
				})
				.catch((error: unknown) => {
					onError?.(error);
				});
		}
	);
	projectReviewStores.set(projectId, store);
	const channel = getReviewBroadcast();
	if (channel) {
		const refresh = (event: MessageEvent<unknown>) => {
			const payload = event.data;
			if (
				!payload ||
				typeof payload !== 'object' ||
				!('projectId' in payload) ||
				payload.projectId !== projectId ||
				projectReviewStores.get(projectId) !== store
			)
				return;
			void saving
				.then(async () => {
					const currentRevision = revision;
					const currentSnapshot = store.getSnapshot();
					const fresh = await readProjectReviewsFromDatabase(projectId);
					if (revision === currentRevision && store.getSnapshot() === currentSnapshot) {
						previous = fresh;
						store.sync(JSON.stringify(fresh));
					}
				})
				.catch((error: unknown) => onError?.(error));
		};
		channel.addEventListener('message', refresh);
		projectReviewSubscriptions.set(projectId, () =>
			channel.removeEventListener('message', refresh)
		);
	}
	return store;
}

async function persistProjectReviewChanges(
	projectId: string,
	previous: ReviewVerdictsByJob,
	reviews: ReviewVerdictsByJob
): Promise<ReviewVerdictsByJob> {
	return withDatabase(async (database) => {
		const transaction = database.transaction([REVIEWS_STORE, PROJECTS_STORE], 'readwrite');
		const project = await requestResult(
			transaction.objectStore(PROJECTS_STORE).get(projectId) as IDBRequest<unknown>
		);
		if (!project) {
			await transactionComplete(transaction);
			return reviews;
		}
		const persisted = await requestResult(
			transaction.objectStore(REVIEWS_STORE).get(projectId) as IDBRequest<unknown>
		);
		const merged = mergeReviewChanges(normalizeReviewVerdicts(persisted), previous, reviews);
		transaction.objectStore(REVIEWS_STORE).put(merged, projectId);
		await transactionComplete(transaction);
		return merged;
	});
}

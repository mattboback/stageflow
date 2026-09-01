import {
	buildLocalProjectExport,
	parseLocalProjectExport,
	type LocalProjectExport
} from './local-project-export';
import {
	LOCAL_PROJECT_SCHEMA_VERSION,
	sanitizeProjectConfiguration,
	type LocalBaseline,
	type LocalProject,
	type LocalProjectConfiguration,
	type LocalRun
} from './projects';

const DATABASE_NAME = 'stageflow-local-projects';
const DATABASE_VERSION = 1;
const PROJECTS_STORE = 'projects';
const BASELINES_STORE = 'baselines';
const RUNS_STORE = 'runs';
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
			const runs = database.createObjectStore(RUNS_STORE, { keyPath: 'jobId' });
			runs.createIndex('projectId', 'projectId');
			runs.createIndex('createdAt', 'createdAt');
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
			[PROJECTS_STORE, BASELINES_STORE, RUNS_STORE],
			'readwrite'
		);
		transaction.objectStore(PROJECTS_STORE).delete(projectId);
		transaction.objectStore(BASELINES_STORE).delete(projectId);

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

export async function getLocalRun(jobId: string): Promise<LocalRun | null> {
	return withDatabase(async (database) => {
		const transaction = database.transaction(RUNS_STORE, 'readonly');
		const run = await requestResult(
			transaction.objectStore(RUNS_STORE).get(jobId) as IDBRequest<LocalRun | undefined>
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
			.forEach((candidate) => runStore.delete(candidate.jobId));

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

	const [baseline, runs] = await Promise.all([
		getLocalBaseline(projectId),
		listLocalProjectRuns(projectId)
	]);
	return buildLocalProjectExport(project, baseline, runs);
}

export async function importLocalProject(value: unknown): Promise<LocalProject> {
	const parsed = parseLocalProjectExport(value);
	if (!parsed) {
		throw new LocalProjectStoreUnavailableError(
			'That file is not a StageFlow local project export.'
		);
	}

	const now = new Date().toISOString();
	const project: LocalProject = {
		...parsed.project,
		schemaVersion: LOCAL_PROJECT_SCHEMA_VERSION,
		configuration: sanitizeProjectConfiguration(parsed.project.configuration),
		updatedAt: now
	};

	await withDatabase(async (database) => {
		const transaction = database.transaction(
			[PROJECTS_STORE, BASELINES_STORE, RUNS_STORE],
			'readwrite'
		);
		transaction.objectStore(PROJECTS_STORE).put(project);
		if (parsed.baseline) {
			transaction.objectStore(BASELINES_STORE).put({
				...parsed.baseline,
				projectId: project.id
			});
		}
		for (const run of parsed.runs) {
			transaction.objectStore(RUNS_STORE).put({
				...run,
				projectId: project.id
			});
		}
		await transactionComplete(transaction);
	});

	return project;
}

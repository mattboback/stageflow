import { browser } from '$app/environment';

export interface ContrastVerdict {
	verdict: 'pass' | 'fail';
	fg: string;
	bg: string;
	ratio: number | null;
	at: string;
}

export interface ContrastVerdictSummary {
	total: number;
	reviewed: number;
	failed: number;
}

type VerdictsByJob = Record<string, Record<string, ContrastVerdict>>;

const STORAGE_KEY = 'contrast-verdicts';
const MAX_JOBS = 20;

function latestTimestamp(verdicts: Record<string, ContrastVerdict>): string {
	return Object.values(verdicts).reduce((max, v) => (v.at > max ? v.at : max), '');
}

export function createContrastVerdictsStore() {
	let verdicts = $state<VerdictsByJob>({});

	function load() {
		if (!browser) {
			return;
		}
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			verdicts = stored ? (JSON.parse(stored) as VerdictsByJob) : {};
		} catch (e) {
			console.warn('[contrast-verdicts] Failed to load from localStorage:', e);
			verdicts = {};
		}
	}

	function trimToMaxJobs() {
		const jobIds = Object.keys(verdicts);
		if (jobIds.length <= MAX_JOBS) {
			return;
		}
		const byRecency = jobIds.sort((a, b) =>
			latestTimestamp(verdicts[b]).localeCompare(latestTimestamp(verdicts[a]))
		);
		verdicts = Object.fromEntries(byRecency.slice(0, MAX_JOBS).map((id) => [id, verdicts[id]]));
	}

	function save(attempt = 0) {
		if (!browser) {
			return;
		}
		const MAX_SAVE_ATTEMPTS = 3;

		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(verdicts));
		} catch (e) {
			if (e instanceof DOMException && e.name === 'QuotaExceededError') {
				console.warn('[contrast-verdicts] localStorage quota exceeded, trimming oldest jobs');
				const jobCount = Object.keys(verdicts).length;
				if (jobCount > 1 && attempt < MAX_SAVE_ATTEMPTS) {
					const byRecency = Object.keys(verdicts).sort((a, b) =>
						latestTimestamp(verdicts[b]).localeCompare(latestTimestamp(verdicts[a]))
					);
					verdicts = Object.fromEntries(
						byRecency.slice(0, Math.ceil(jobCount / 2)).map((id) => [id, verdicts[id]])
					);
					save(attempt + 1);
				} else if (attempt >= MAX_SAVE_ATTEMPTS) {
					console.error('[contrast-verdicts] Unable to save after trimming, clearing verdicts');
					verdicts = {};
					localStorage.removeItem(STORAGE_KEY);
				}
			} else {
				console.error('[contrast-verdicts] Failed to save:', e);
			}
		}
	}

	function getVerdict(jobId: string, issueId: string): ContrastVerdict | null {
		const forJob: Record<string, ContrastVerdict> | undefined = verdicts[jobId];
		// eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- verdicts[jobId] is undefined for unknown jobs at runtime (noUncheckedIndexedAccess is off)
		return forJob?.[issueId] ?? null;
	}

	function setVerdict(jobId: string, issueId: string, verdict: Omit<ContrastVerdict, 'at'>) {
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- one-shot ISO timestamp string, not reactive Date state
		const at = new Date().toISOString();
		verdicts = {
			...verdicts,
			[jobId]: {
				...verdicts[jobId],
				[issueId]: { ...verdict, at }
			}
		};
		trimToMaxJobs();
		save();
	}

	function clearVerdict(jobId: string, issueId: string) {
		const forJob: Record<string, ContrastVerdict> | undefined = verdicts[jobId];
		// eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- verdicts[jobId] is undefined for unknown jobs at runtime (noUncheckedIndexedAccess is off)
		if (!forJob || !(issueId in forJob)) {
			return;
		}
		const { [issueId]: _removed, ...rest } = forJob;
		verdicts =
			Object.keys(rest).length > 0
				? { ...verdicts, [jobId]: rest }
				: Object.fromEntries(Object.entries(verdicts).filter(([id]) => id !== jobId));
		save();
	}

	function summarize(jobId: string, issueIds: string[]): ContrastVerdictSummary {
		const forJob = verdicts[jobId] ?? {};
		const recorded = issueIds.map((id) => forJob[id]).filter(Boolean);
		return {
			total: issueIds.length,
			reviewed: recorded.length,
			failed: recorded.filter((v) => v.verdict === 'fail').length
		};
	}

	if (browser) {
		load();
	}

	return {
		get verdicts() {
			return verdicts;
		},
		getVerdict,
		setVerdict,
		clearVerdict,
		summarize
	};
}

export const contrastVerdictsStore = createContrastVerdictsStore();

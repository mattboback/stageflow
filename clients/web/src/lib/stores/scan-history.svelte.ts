import { browser } from "$app/environment";

export interface ScanHistoryItem {
	id: string;
	type: "zip" | "url";
	label: string;
	createdAt: string;
	status: "pending" | "complete" | "failed";
}

const STORAGE_KEY = "scan-history";
const MAX_HISTORY_ITEMS = 10;

function createScanHistoryStore() {
	let history = $state<ScanHistoryItem[]>([]);
	let isLoaded = $state(false);

	function load() {
		if (!browser) return;
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			history = stored ? (JSON.parse(stored) as ScanHistoryItem[]) : [];
		} catch (e) {
			console.warn(
				"[scan-history] Failed to load history from localStorage:",
				e,
			);
			history = [];
		}
		isLoaded = true;
	}

	function save(attempt = 0) {
		if (!browser) return;
		const MAX_SAVE_ATTEMPTS = 3;

		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(history));
		} catch (e) {
			if (e instanceof DOMException && e.name === "QuotaExceededError") {
				console.warn(
					"[scan-history] localStorage quota exceeded, trimming history",
				);
				// Try to save with fewer items, but limit recursion depth
				if (history.length > 1 && attempt < MAX_SAVE_ATTEMPTS) {
					history = history.slice(0, Math.ceil(history.length / 2));
					save(attempt + 1);
				} else if (attempt >= MAX_SAVE_ATTEMPTS) {
					console.error(
						"[scan-history] Unable to save after trimming, clearing history",
					);
					history = [];
					localStorage.removeItem(STORAGE_KEY);
				}
			} else {
				console.error("[scan-history] Failed to save:", e);
			}
		}
	}

	function addToHistory(item: Omit<ScanHistoryItem, "createdAt">) {
		const newItem: ScanHistoryItem = {
			...item,
			createdAt: new Date().toISOString(),
		};
		const filtered = history.filter((h) => h.id !== item.id);
		history = [newItem, ...filtered].slice(0, MAX_HISTORY_ITEMS);
		save();
	}

	function updateStatus(id: string, status: ScanHistoryItem["status"]) {
		history = history.map((item) =>
			item.id === id ? { ...item, status } : item,
		);
		save();
	}

	function removeFromHistory(id: string) {
		history = history.filter((item) => item.id !== id);
		save();
	}

	function clearHistory() {
		if (!browser) return;
		try {
			localStorage.removeItem(STORAGE_KEY);
			history = [];
		} catch (e) {
			console.error("[scan-history] Failed to clear:", e);
		}
	}

	// Initialize on load
	if (browser) {
		load();
	}

	return {
		get history() {
			return history;
		},
		get isLoaded() {
			return isLoaded;
		},
		addToHistory,
		updateStatus,
		removeFromHistory,
		clearHistory,
	};
}

export const scanHistoryStore = createScanHistoryStore();

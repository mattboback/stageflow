import { useCallback, useSyncExternalStore } from 'react';

import { createThemeStore, THEME_STORAGE_KEY, type Theme, type ThemeStorage } from '../theme';

const browserStorage: ThemeStorage = {
	getItem(key) {
		return typeof window === 'undefined' ? null : window.localStorage.getItem(key);
	},
	setItem(key, value) {
		if (typeof window !== 'undefined') window.localStorage.setItem(key, value);
	}
};

const store = createThemeStore(
	browserStorage,
	typeof document === 'undefined' ? null : document.documentElement
);

let activeSubscriptions = 0;

function handleStorage(event: StorageEvent) {
	if (event.key !== THEME_STORAGE_KEY && event.key !== null) return;
	try {
		if (event.storageArea && event.storageArea !== window.localStorage) return;
	} catch {
		return;
	}
	store.sync(event.key === null ? null : event.newValue);
}

function subscribe(listener: () => void): () => void {
	const unsubscribe = store.subscribe(listener);
	if (activeSubscriptions === 0 && typeof window !== 'undefined') {
		window.addEventListener('storage', handleStorage);
	}
	activeSubscriptions += 1;
	return () => {
		unsubscribe();
		activeSubscriptions -= 1;
		if (activeSubscriptions === 0 && typeof window !== 'undefined') {
			window.removeEventListener('storage', handleStorage);
		}
	};
}

export function useTheme(): { theme: Theme; setTheme: (next: Theme) => void } {
	// Wrapped rather than passed as bare method references: unbound-method flags
	// those because a detached method could observe the wrong `this`.
	const theme = useSyncExternalStore(
		subscribe,
		() => store.getSnapshot(),
		() => store.getServerSnapshot()
	);

	const setTheme = useCallback((next: Theme) => {
		store.set(next);
	}, []);

	return { theme, setTheme };
}

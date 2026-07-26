/*
 * Theme selection.
 *
 * The palette itself is pure CSS: instrument.css declares every color as
 * light-dark(), and `color-scheme: light dark` on :root means an untouched
 * visitor already gets their OS preference with no JavaScript at all. This
 * module exists only for the visitor who wants something other than their OS
 * setting, and its whole job is to put one attribute on <html>.
 */

export const THEME_STORAGE_KEY = 'stageflow-theme';

export type Theme = 'light' | 'dark' | 'system';

export const THEME_LABELS: Record<Theme, string> = {
	light: 'Light',
	dark: 'Dark',
	system: 'System'
};

export const THEMES: readonly Theme[] = ['light', 'dark', 'system'];

function isTheme(value: unknown): value is Theme {
	return value === 'light' || value === 'dark' || value === 'system';
}

/**
 * Runs in <head> before first paint, so an explicit choice is applied before
 * the browser has anything to repaint. Kept to one line of ES5 on purpose:
 * it is inlined into every prerendered document and the SPA fallback.
 *
 * The try/catch is load-bearing rather than defensive. localStorage throws
 * outright in Safari private mode and in sandboxed iframes, and an uncaught
 * throw here is a pageerror — which the Playwright fixture fails every spec
 * on. 'system' deliberately writes no attribute: absent data-theme, the
 * :root rule leaves color-scheme on both and the OS decides.
 */
export const THEME_INIT_SCRIPT =
	`try{var t=localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});` +
	`if(t==="dark"||t==="light")document.documentElement.dataset.theme=t}catch(e){}`;

export interface ThemeStorage {
	getItem(key: string): string | null;
	setItem(key: string, value: string): void;
}

function applyTheme(theme: Theme, root: HTMLElement): void {
	if (theme === 'system') delete root.dataset.theme;
	else root.dataset.theme = theme;
}

export function createThemeStore(storage: ThemeStorage | null, root: HTMLElement | null) {
	let cache: Theme = 'system';
	let cacheLoaded = false;
	const listeners = new Set<() => void>();

	const readPersisted = (): Theme => {
		if (!storage) return 'system';
		try {
			const raw = storage.getItem(THEME_STORAGE_KEY);
			return isTheme(raw) ? raw : 'system';
		} catch {
			return cacheLoaded ? cache : 'system';
		}
	};

	const notify = () => {
		for (const listener of listeners) listener();
	};

	return {
		subscribe(listener: () => void): () => void {
			listeners.add(listener);
			return () => listeners.delete(listener);
		},

		getSnapshot(): Theme {
			if (!cacheLoaded) {
				cache = readPersisted();
				cacheLoaded = true;
			}
			return cache;
		},

		/*
		 * Always 'system'. The prerendered HTML was built without a visitor, so
		 * the first client render has to agree with it; reading storage here
		 * would produce a hydration mismatch for anyone who has chosen a theme.
		 * The init script has already put the right attribute on <html>, so the
		 * page looks correct while React briefly believes otherwise.
		 */
		getServerSnapshot(): Theme {
			return 'system';
		},

		set(theme: Theme) {
			cache = theme;
			cacheLoaded = true;
			if (root) applyTheme(theme, root);
			try {
				storage?.setItem(THEME_STORAGE_KEY, theme);
			} catch {
				// Quota or private-mode failures lose persistence, not the session.
			}
			notify();
		},

		/** Adopt a value written by another tab. */
		sync(raw: string | null) {
			cache = isTheme(raw) ? raw : 'system';
			cacheLoaded = true;
			if (root) applyTheme(cache, root);
			notify();
		}
	};
}

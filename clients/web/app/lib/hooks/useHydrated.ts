import { useSyncExternalStore } from 'react';

function subscribe() {
	return () => {};
}
function onClient() {
	return true;
}
function duringHydration() {
	return false;
}

/**
 * False while React hydrates prerendered HTML, true on every render after
 * that and on client-only renders. A prerendered route waits for it before
 * showing anything the HTML could not know, such as the query string, so its
 * first render matches the HTML and React keeps the page.
 */
export function useHydrated(): boolean {
	return useSyncExternalStore(subscribe, onClient, duringHydration);
}

import type { OfflineViewerAssets } from './portable-audit';

/** Hosted-only: the standalone viewer must never import its own build output. */
export async function loadOfflineViewerAssets(): Promise<OfflineViewerAssets> {
	let timer: ReturnType<typeof setTimeout> | undefined;
	try {
		const { javascript, css } = await Promise.race([
			import('./offline-viewer-assets'),
			new Promise<never>((_, reject) => {
				timer = setTimeout(() => reject(new Error('Offline viewer loading timed out.')), 15_000);
			})
		]);
		if (new Blob([javascript]).size > 8 * 1024 * 1024 || new Blob([css]).size > 1024 * 1024)
			throw new Error('The offline viewer exceeds the size limit.');
		if (!javascript.includes('stageflow-offline-root') || !css.trim())
			throw new Error('The offline viewer is invalid.');
		return { javascript, css };
	} catch {
		throw new Error('The offline viewer is unavailable. Reload StageFlow and try again.');
	} finally {
		clearTimeout(timer);
	}
}

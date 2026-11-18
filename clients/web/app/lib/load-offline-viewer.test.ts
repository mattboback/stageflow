import { afterEach, describe, expect, it, vi } from 'vitest';
import { loadOfflineViewerAssets } from './load-offline-viewer';

const assets = vi.hoisted(() => ({ javascript: '', css: '' }));
vi.mock('./offline-viewer-assets', () => assets);

afterEach(() => {
	assets.javascript = '';
	assets.css = '';
	vi.unstubAllGlobals();
});

describe('offline viewer loading', () => {
	it('returns a bundled pair without requesting fixed-name assets', async () => {
		assets.javascript = 'document.getElementById("stageflow-offline-root")';
		assets.css = 'body { color: black; }';
		const fetch = vi.fn();
		vi.stubGlobal('fetch', fetch);
		expect(await loadOfflineViewerAssets()).toEqual(assets);
		expect(fetch).not.toHaveBeenCalled();
	});

	it('provides reload guidance if the bundled viewer is invalid', async () => {
		assets.javascript = '<html>Not a viewer</html>';
		assets.css = 'body {}';
		await expect(loadOfflineViewerAssets()).rejects.toThrow('Reload StageFlow and try again');
	});

	it('rejects an oversized viewer before embedding it in an export', async () => {
		assets.javascript = 'stageflow-offline-root';
		assets.css = 'x'.repeat(1024 * 1024 + 1);
		await expect(loadOfflineViewerAssets()).rejects.toThrow('Reload StageFlow and try again');
	});
});

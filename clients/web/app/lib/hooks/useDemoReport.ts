import { useEffect, useState } from 'react';

import type { ScanStatus, ScreenshotArtifact } from '../types/scan';
import type { UnifiedReport } from '../types/unified-report';

export const DEMO_JOB_ID = 'demo';
export const DEMO_REPORT_URL = '/demo/report.json';
export const DEMO_SCREENSHOTS_URL = '/demo/screenshots.json';

interface DemoReport {
	report: UnifiedReport | null;
	screenshots: ScreenshotArtifact[];
	status: ScanStatus;
	error: string | null;
}

/*
 * Loads the committed demo report. No API, no polling, no job.
 *
 * fetch rather than import(): the report is ~90 KB of JSON that would
 * otherwise be parsed into the module graph and shipped inside a bundle
 * chunk, and it is genuinely data, not code.
 *
 * Every failure lands in `error`. An unhandled rejection here would be a
 * pageerror, which the Playwright fixture fails every spec on -- and
 * ReportView already renders `error` through the same branch a failed scan
 * uses, so there is no new UI for a state that should never happen.
 */
export function useDemoReport(): DemoReport {
	const [state, setState] = useState<DemoReport>({
		report: null,
		screenshots: [],
		status: 'loading',
		error: null
	});

	useEffect(() => {
		const controller = new AbortController();

		async function load() {
			try {
				const [reportResponse, screenshotsResponse] = await Promise.all([
					fetch(DEMO_REPORT_URL, { signal: controller.signal }),
					fetch(DEMO_SCREENSHOTS_URL, { signal: controller.signal })
				]);
				if (!reportResponse.ok || !screenshotsResponse.ok) {
					throw new Error(
						`demo assets returned ${reportResponse.status} / ${screenshotsResponse.status}`
					);
				}
				const report = (await reportResponse.json()) as UnifiedReport;
				const screenshots = (await screenshotsResponse.json()) as ScreenshotArtifact[];
				/*
				 * Unconditionally, unlike useScanReport, which returns screenshots
				 * only when it has a job. There is no job here, and gating on one
				 * would render the empty branch of the visual review panel -- the
				 * single easiest thing to get wrong on this route.
				 */
				setState({ report, screenshots, status: 'complete', error: null });
			} catch (loadError) {
				if (controller.signal.aborted) return;
				setState({
					report: null,
					screenshots: [],
					status: 'failed',
					error:
						loadError instanceof Error
							? `Could not load the demo report: ${loadError.message}`
							: 'Could not load the demo report.'
				});
			}
		}

		void load();
		return () => {
			controller.abort();
		};
	}, []);

	return state;
}

import { fingerprintProjectConfiguration } from '../lib/projects';
import { createRoot } from 'react-dom/client';
import {
	MAX_AUDIT_BYTES,
	parsePortableAudit,
	portableScreenshotDigests
} from '../lib/portable-audit';
import { createReviewContext } from '../lib/report/review-context';
import { createReviewVerdictStore } from '../lib/report/review-verdict-store';
import { OfflineAudit } from './OfflineAudit';
/* The in-app report gets these two from route links(), which this entry never
   renders. Without them the exported audit opened as unstyled text. */
import '../styles/instrument.css';
import '../styles/report.css';
import '../routes/scan-report.css';
import './offline.css';

async function startOfflineAudit() {
	const root = document.getElementById('stageflow-offline-root');
	if (!root) return;
	try {
		const json = document.getElementById('stageflow-audit')?.textContent ?? 'null';
		if (new Blob([json]).size > MAX_AUDIT_BYTES)
			throw new Error('This audit exceeds the 96 MiB opening limit.');
		const raw: unknown = JSON.parse(json);
		const audit = parsePortableAudit(raw);
		if (!audit) throw new Error('This file does not contain a valid StageFlow portable audit.');
		const context = await createReviewContext(
			audit.report,
			'portable-audit',
			audit.configuration ? await fingerprintProjectConfiguration(audit.configuration) : undefined,
			await portableScreenshotDigests(audit.screenshots)
		);
		const store = createReviewVerdictStore(null, () => new Date().toISOString(), audit.reviews);
		const assets = {
			javascript: document.getElementById('stageflow-offline-js')?.textContent ?? '',
			css: document.getElementById('stageflow-offline-css')?.textContent ?? ''
		};
		createRoot(root).render(
			<OfflineAudit audit={audit} assets={assets} store={store} context={context} />
		);
	} catch (error) {
		root.textContent = error instanceof Error ? error.message : 'This audit could not be opened.';
	}
}

void startOfflineAudit();

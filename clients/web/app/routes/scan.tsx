import { useEffect, useMemo, useRef } from 'react';
import {
	Link,
	isRouteErrorResponse,
	useLocation,
	useNavigate,
	useParams,
	useRouteError,
	useSearchParams,
	type MetaFunction
} from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import { RouteFault } from '../components/RouteFault';
import { useScanStatus } from '../lib/hooks/useScanMonitor';
import { queueLabel } from '../lib/stores/scan-status/log-messages';
import { IS_HOSTED_DEMO, pageTitle } from '../lib/site-metadata';
import { scannerLabel } from '../lib/report';
import scanStyles from './scan.css?url';

export const links = () => [{ rel: 'stylesheet', href: scanStyles }];

export const meta: MetaFunction = () => [
	{ title: pageTitle('Scan running') },
	{ name: 'robots', content: 'noindex' }
];

const RUNNING_STATES = new Set([
	'pending',
	'processing',
	'extracting',
	'scanning',
	'completing',
	'loading'
]);

type ChannelState = 'done' | 'run' | 'queue' | 'err';

function fmtElapsed(totalSeconds: number): string {
	const s = Math.max(0, Math.floor(totalSeconds));
	const m = Math.floor(s / 60);
	const r = s % 60;
	return `${String(m).padStart(2, '0')}:${String(r).padStart(2, '0')}`;
}

const TAG_LABEL: Record<ChannelState, string> = {
	done: 'Done',
	run: 'Running',
	queue: 'Waiting',
	err: 'Failed'
};

const SUB_LABEL: Record<ChannelState, string> = {
	done: 'Complete',
	run: 'Scanning…',
	queue: 'Waiting',
	err: 'Did not finish'
};

export default function Scan() {
	const { id = '' } = useParams();
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();
	const projectId = searchParams.get('project');
	const projectQuery = projectId ? `?project=${encodeURIComponent(projectId)}` : '';
	const playgroundPath = projectId
		? `/playground?project=${encodeURIComponent(projectId)}`
		: '/playground';
	const { status, result, elapsed, logs, transport, error, retry } = useScanStatus(id);

	const logBodyRef = useRef<HTMLDivElement>(null);

	const isRunning = RUNNING_STATES.has(status);
	const isComplete = status === 'complete';
	const isFailed = status === 'failed' || status === 'error';

	// On completion, hand off to the report surface.
	useEffect(() => {
		if (isComplete) {
			void navigate(`/scan/${id}/report${projectQuery}`, { replace: true });
		}
	}, [isComplete, id, navigate, projectQuery]);

	// Keep the log stream pinned to the latest line.
	useEffect(() => {
		const el = logBodyRef.current;
		if (el) el.scrollTop = el.scrollHeight;
	}, [logs.length]);

	const expected = result?.expected_scanners ?? [];
	const completed = useMemo(
		() => new Set(result?.completed_scanners ?? []),
		[result?.completed_scanners]
	);
	const remaining = useMemo(
		() => new Set(result?.remaining_scanners ?? []),
		[result?.remaining_scanners]
	);

	const scannersLaunched = status === 'scanning' || status === 'completing';
	const channels = expected.map((scannerId) => {
		const scanner = result?.scanner_statuses?.[scannerId];
		let state: ChannelState;
		if (scanner?.state === 'failed') state = 'err';
		else if (scanner?.state === 'completed' || completed.has(scannerId)) state = 'done';
		else if (isFailed) state = 'err';
		else if (
			scanner?.state === 'running' ||
			(!scanner && scannersLaunched && remaining.has(scannerId))
		)
			state = 'run';
		else state = 'queue';
		const detail =
			scanner?.state === 'failed'
				? (scanner.error ?? 'Scanner failed')
				: state === 'done' && scanner
					? [
							scanner.total_ms !== undefined
								? `Complete in ${(scanner.total_ms / 1000).toFixed(1)}s`
								: 'Complete',
							scanner.total_pages > 0
								? `${scanner.pages_scanned} of ${scanner.total_pages} pages`
								: null,
							`${scanner.violations} ${scanner.violations === 1 ? 'issue' : 'issues'}`
						]
							.filter(Boolean)
							.join(' · ')
					: scanner && scanner.total_pages > 0
						? `Page progress: ${scanner.pages_scanned} of ${scanner.total_pages}`
						: SUB_LABEL[state];
		// How far this scanner is: settled ones count in full, a running one
		// by the pages it has finished, a waiting one not at all.
		const fraction =
			state === 'done' || state === 'err'
				? 1
				: state === 'run' && scanner && scanner.total_pages > 0
					? Math.min(1, scanner.pages_scanned / scanner.total_pages)
					: 0;
		return {
			id: scannerId,
			state,
			detail,
			fraction,
			violations: scanner?.violations,
			totalMs: scanner?.total_ms
		};
	});
	const doneCount = completed.size;
	const failedCount = channels.filter((ch) => ch.state === 'err').length;
	const settledCount = doneCount + failedCount;
	const totalCount = expected.length;
	// Page progress, not just settled scanners, so the bar moves while
	// scanners are still working through pages.
	const rawPct =
		totalCount > 0
			? Math.round((channels.reduce((sum, ch) => sum + ch.fraction, 0) / totalCount) * 100)
			: Math.round(result?.progress?.percentage ?? 0);
	const pct = isComplete ? 100 : Math.min(99, rawPct);

	/* Render every expected scanner from the first paint in Queued state so
	   the Scanners card height never changes as runners start (no CLS). */
	/* Set only while the job waits for a free slot; claiming the job clears it. */
	const queuePosition = result?.state === 'PENDING' ? result.queue_position : undefined;

	const artifacts = result?.artifacts;
	const shots = artifacts?.screenshots?.length ?? 0;
	const lastLogLine = logs.length > 0 ? logs[logs.length - 1] : null;

	const statusPill = isComplete ? (
		<Pill variant="done">Complete</Pill>
	) : isFailed ? (
		<Pill variant="error">Failed</Pill>
	) : status === 'loading' ? (
		<Pill variant="queued">Connecting</Pill>
	) : queuePosition !== undefined ? (
		<Pill variant="queued">Queued</Pill>
	) : (
		<Pill variant="live">Running</Pill>
	);

	const transportLabel =
		transport === 'streaming'
			? 'Live updates on'
			: transport === 'reconnecting'
				? 'Reconnecting…'
				: transport === 'polling'
					? 'Checking periodically'
					: 'Connecting…';

	const heading = isFailed
		? 'Scan failed'
		: isComplete
			? 'Scan complete'
			: status === 'completing' || (totalCount > 0 && settledCount === totalCount)
				? 'Finalizing report'
				: queuePosition !== undefined
					? 'Waiting in the queue'
					: 'Scan in progress';

	return (
		<>
			<SiteHeader
				app={{
					backTo: playgroundPath,
					backLabel: projectId ? 'Project' : 'New scan',
					section: 'Scan run'
				}}
			/>

			<main id="main" className="run">
				<div className="wrap wrap--app">
					{/* status header: the one progress model */}
					<div className={`shead${isFailed ? ' shead--failed' : ''}`}>
						<div className="shead__main">
							<div className="shead__id">
								<h1>{heading}</h1>
								{statusPill}
							</div>
							<div className="shead__meta-row">
								<span className="shead__url mono" title={id}>
									job {id}
								</span>
								{isRunning && (
									<span className="shead__leave">
										Safe to leave — this page reconnects to the run.
									</span>
								)}
							</div>
							{IS_HOSTED_DEMO && (
								<p className="shead__leave">
									On the hosted demo, this report stays available for 24 hours. Save it to a local
									project if you want to come back later.
								</p>
							)}
							<div
								className="meter"
								role="progressbar"
								aria-valuenow={pct}
								aria-valuemin={0}
								aria-valuemax={100}
								aria-label="Overall scan progress"
							>
								<i
									className={isRunning ? 'live' : undefined}
									style={{ transform: `scaleX(${pct / 100})` }}
								/>
							</div>
							<p className="shead__progress num">
								{totalCount > 0
									? `${settledCount} of ${totalCount} scanners settled · ${doneCount} successful${failedCount > 0 ? ` · ${failedCount} failed` : ''}`
									: `${pct}%`}
							</p>
						</div>
						<div className="shead__meta">
							<div className="timer">
								<b>{fmtElapsed(elapsed)}</b>
								<small>elapsed</small>
							</div>
							{isComplete && (
								<Link className="btn btn--primary" to={`/scan/${id}/report${projectQuery}`}>
									View report{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</Link>
							)}
						</div>
					</div>

					{isFailed && (
						<div className="note note--err" role="alert">
							<span className="note__ic" aria-hidden="true">
								!
							</span>
							<div>
								<span>{error ?? 'The scan did not complete. Check the stream for details.'}</span>
								<div className="note__actions">
									<Link className="btn btn--ghost btn--sm" to={playgroundPath}>
										Back to playground
									</Link>
									{id && (
										<button type="button" className="btn btn--ghost btn--sm" onClick={retry}>
											Retry connection
										</button>
									)}
								</div>
							</div>
						</div>
					)}

					<div className="grid">
						<div className="grid__main">
							<section className="panel" aria-label="Scanners">
								<div className="panel__head">
									<h2>
										Scanners
										{totalCount > 0 && (
											<span className="ch-summary num">
												{doneCount} of {totalCount} done
											</span>
										)}
									</h2>
									<span
										className={`transport${transport === 'streaming' ? ' transport--live' : ''}`}
									>
										{transportLabel}
									</span>
								</div>
								<div className="panel__body panel__body--flush">
									{queuePosition !== undefined ? (
										<div className="ch ch--prep">
											<span className="ch__led" aria-hidden="true" />
											<div>
												<div className="ch__name">{queueLabel(queuePosition)}</div>
												<div className="ch__sub">Scans run one at a time.</div>
											</div>
										</div>
									) : channels.length === 0 ? (
										<div className="ch">
											<span className="ch__led" aria-hidden="true" />
											<div>
												<div className="ch__name">Waiting for the orchestrator…</div>
												<div className="ch__sub">Scheduling containers</div>
											</div>
										</div>
									) : (
										channels.map((ch) => (
											<div className={`ch ch--${ch.state}`} key={ch.id}>
												<span className="ch__led" aria-hidden="true" />
												<div>
													<div className="ch__name">{scannerLabel(ch.id)}</div>
													<div className="ch__sub">{ch.detail}</div>
													{/* Track mounted in every state so row height is constant;
													    done fills fully, queued stays empty. */}
													<span className="ch__bar" aria-hidden="true">
														<i style={{ transform: `scaleX(${ch.fraction})` }} />
													</span>
												</div>
												<div className="ch__right">
													{ch.state === 'done' && ch.violations !== undefined && (
														<span className="ch__issues num">
															{ch.violations} {ch.violations === 1 ? 'issue' : 'issues'}
														</span>
													)}
													<span className="ch__tag">{TAG_LABEL[ch.state]}</span>
												</div>
											</div>
										))
									)}
								</div>
							</section>
						</div>

						<aside className="side" aria-label="Run summary">
							<div className="panel">
								<div className="panel__head">
									<h2>Artifacts</h2>
								</div>
								<div className="panel__body panel__body--tight">
									<div className={`artifact${shots > 0 ? '' : ' pending'}`}>
										<span className="ic">PNG</span>
										{shots > 0 ? `${shots} page screenshots` : 'Page screenshots · pending'}
									</div>
									<div className={`artifact${artifacts?.report_json ? '' : ' pending'}`}>
										<span className="ic">JSON</span>
										Unified report{artifacts?.report_json ? '' : ' · pending'}
									</div>
								</div>
							</div>
						</aside>

						{/* Diagnostic detail: collapsed by default, never the main event */}
						<details className="log">
							<summary className="log__summary">
								<span className="log__summary-title">Technical log</span>
								{lastLogLine && <span className="log__summary-last mono">{lastLogLine}</span>}
								<span className="log__summary-toggle" aria-hidden="true" />
							</summary>
							<div className="log__body" ref={logBodyRef} aria-live="polite">
								{logs.length === 0 ? (
									<div className="log__line">
										<span className="log__m log__empty">waiting for output…</span>
									</div>
								) : (
									logs.map((line, i) => (
										<div className="log__line" key={i}>
											<span className="log__m">{line}</span>
										</div>
									))
								)}
							</div>
						</details>
					</div>
				</div>
			</main>

			<SiteFooter slim />
		</>
	);
}

export function ErrorBoundary() {
	const error = useRouteError();
	const { pathname } = useLocation();
	const status = isRouteErrorResponse(error) ? error.status : 500;

	return (
		<>
			<SiteHeader app={{ backTo: '/playground', backLabel: 'New scan', section: 'Scan run' }} />
			<RouteFault
				status={status}
				title="This scan can't be shown."
				detail="Scan jobs are kept for a limited window after they complete. The job id may have expired, or the live view failed to load."
				traceLine={`scan view ${pathname} failed to load`}
				traceHint="check the job id · try again or run a new scan"
				actions={
					<>
						<Link className="btn btn--primary" to="/playground">
							Configure a scan{' '}
							<span className="ar" aria-hidden="true">
								→
							</span>
						</Link>
						<a className="btn btn--ghost" href={pathname}>
							Try again
						</a>
					</>
				}
			/>
			<SiteFooter slim />
		</>
	);
}

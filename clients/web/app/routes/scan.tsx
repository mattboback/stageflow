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
import { pageTitle } from '../lib/site-metadata';
import { SCANNER_META } from '../lib/report';
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

function scannerLabel(id: string): string {
	return SCANNER_META[id]?.label ?? id;
}

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
			navigate(`/scan/${id}/report${projectQuery}`, { replace: true });
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

	const channels = expected.map((scannerId) => {
		let state: ChannelState;
		if (completed.has(scannerId)) {
			state = 'done';
		} else if (isFailed) {
			state = 'err';
		} else if (remaining.has(scannerId)) {
			state = 'queue';
		} else {
			state = 'run';
		}
		return { id: scannerId, state };
	});

	const doneCount = completed.size;
	const totalCount = expected.length;
	/* One canonical progress number: scanner completion when we know the
	   roster, backend task progress only before the roster exists. */
	const pct =
		totalCount > 0
			? Math.round((doneCount / totalCount) * 100)
			: Math.round(result?.progress?.percentage ?? 0);

	/* Nothing has started yet: one "preparing" message beats seven identical
	   queued rows that all ask "waiting for what?". */
	const isPreparing =
		isRunning && channels.length > 0 && channels.every((ch) => ch.state === 'queue');

	const artifacts = result?.artifacts;
	const shots = artifacts?.screenshots?.length ?? 0;
	const lastLogLine = logs.length > 0 ? logs[logs.length - 1] : null;

	const statusPill = isComplete ? (
		<Pill variant="done">Complete</Pill>
	) : isFailed ? (
		<Pill variant="error">Failed</Pill>
	) : status === 'loading' ? (
		<Pill variant="queued">Connecting</Pill>
	) : (
		<Pill variant="live">Running</Pill>
	);

	const transportLabel =
		transport === 'streaming'
			? 'Live updates on'
			: transport === 'polling'
				? 'Checking periodically'
				: 'Connecting…';

	const heading = isFailed ? 'Scan failed' : isComplete ? 'Scan complete' : 'Scan in progress';

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
				<div className="wrap run__wrap">
					{/* status header: the one progress model */}
					<div className={`shead${isFailed ? ' shead--failed' : ''}`}>
						<div>
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
									? `${doneCount} of ${totalCount} scanners complete · ${pct}%`
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
								<div className="panel__body" style={{ padding: 0 }}>
									{channels.length === 0 ? (
										<div className="ch">
											<span className="ch__led" aria-hidden="true" />
											<div>
												<div className="ch__name">Waiting for the orchestrator…</div>
												<div className="ch__sub">Scheduling containers</div>
											</div>
										</div>
									) : isPreparing ? (
										<div className="ch ch--prep">
											<span className="ch__led" aria-hidden="true" />
											<div>
												<div className="ch__name">Preparing scan runner…</div>
												<div className="ch__sub">{totalCount} scanners will run in parallel</div>
											</div>
										</div>
									) : (
										channels.map((ch) => (
											<div className={`ch ch--${ch.state}`} key={ch.id}>
												<span className="ch__led" aria-hidden="true" />
												<div>
													<div className="ch__name">{scannerLabel(ch.id)}</div>
													{/* queued rows already say "Waiting" in the tag */}
													{ch.state !== 'done' && ch.state !== 'queue' && (
														<div className="ch__sub">{SUB_LABEL[ch.state]}</div>
													)}
												</div>
												<div className="ch__right">
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
									<div className={`artifact${artifacts?.report_html ? '' : ' pending'}`}>
										<span className="ic">HTML</span>
										Report bundle{artifacts?.report_html ? '' : ' · pending'}
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
							Run a new scan{' '}
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

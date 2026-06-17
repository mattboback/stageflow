import { useEffect, useMemo, useRef } from 'react';
import { Link, useNavigate, useParams, type MetaFunction } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import { Gauge } from '../components/Gauge';
import { useScanStatus } from '../lib/hooks/useScanMonitor';
import { SCANNER_META } from '../lib/report';
import scanStyles from './scan.css?url';

export const links = () => [{ rel: 'stylesheet', href: scanStyles }];

export const meta: MetaFunction = () => [
	{ title: 'Scan running — StageFlow' },
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
	done: 'done',
	run: 'running',
	queue: 'queued',
	err: 'failed'
};

export default function Scan() {
	const { id = '' } = useParams();
	const navigate = useNavigate();
	const { status, result, elapsed, logs, transport, error } = useScanStatus(id);

	const logBodyRef = useRef<HTMLDivElement>(null);

	const isRunning = RUNNING_STATES.has(status);
	const isComplete = status === 'complete';
	const isFailed = status === 'failed' || status === 'error';

	// On completion, hand off to the report surface.
	useEffect(() => {
		if (isComplete) {
			navigate(`/scan/${id}/report`, { replace: true });
		}
	}, [isComplete, id, navigate]);

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
	const pct =
		totalCount > 0
			? Math.round((doneCount / totalCount) * 100)
			: Math.round(result?.progress?.percentage ?? 0);

	const artifacts = result?.artifacts;
	const shots = artifacts?.screenshots?.length ?? 0;

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
			? 'SSE connected'
			: transport === 'polling'
				? 'polling'
				: 'connecting…';

	return (
		<>
			<SiteHeader />

			<main id="main" className="run">
				<div className="wrap">
					{/* status header */}
					<div className="shead">
						<div>
							<div className="shead__id">
								<h1>{isFailed ? 'Scan failed' : isComplete ? 'Scan complete' : 'Scan in progress'}</h1>
								{statusPill}
							</div>
							<div
								style={{
									marginTop: '.55rem',
									display: 'flex',
									gap: '1.2rem',
									flexWrap: 'wrap',
									alignItems: 'center'
								}}
							>
								<span className="shead__url">{id}</span>
							</div>
							<div
								className="meter"
								role="progressbar"
								aria-valuenow={pct}
								aria-valuemin={0}
								aria-valuemax={100}
								aria-label="Overall scan progress"
							>
								<i className={isRunning ? 'live' : undefined} style={{ transform: `scaleX(${pct / 100})` }} />
							</div>
						</div>
						<div className="shead__meta">
							<div className="readout">
								<span className="readout__val">
									{doneCount}
									<span style={{ color: 'var(--ink-faint)' }}>/{totalCount || '—'}</span>
								</span>
								<span className="readout__lab">channels done</span>
							</div>
							<div className="timer">
								<b>{fmtElapsed(elapsed)}</b>
								<small>elapsed</small>
							</div>
						</div>
					</div>

					{isFailed && error && (
						<div
							className="note note--err"
							role="status"
							style={{
								marginTop: '1.2rem',
								display: 'flex',
								gap: '.6rem',
								alignItems: 'flex-start',
								padding: '.7rem .85rem',
								borderRadius: 'var(--r-md)',
								background: 'var(--sev-critical-wash)',
								border: '1px solid oklch(0.86 0.05 27)',
								color: 'oklch(0.4 0.16 27)',
								fontSize: '.86rem'
							}}
						>
							<span aria-hidden="true" style={{ fontFamily: 'var(--mono)', fontWeight: 700 }}>
								!
							</span>
							<span>{error}</span>
						</div>
					)}

					<div className="grid">
						{/* channels + log */}
						<div>
							<section className="panel" aria-label="Scanner channels">
								<div className="panel__head">
									<span className="label">
										Channels · live{' '}
										{totalCount > 0 && (
											<span className="ch-summary">
												— {doneCount}/{totalCount} done
											</span>
										)}
									</span>
									<span className="mono" style={{ fontSize: '.72rem', color: 'var(--ink-muted)' }}>
										{transportLabel}
									</span>
								</div>
								<div className="panel__body" style={{ padding: 0 }}>
									{channels.length === 0 ? (
										<div className="ch">
											<span className="ch__led" aria-hidden="true" />
											<div>
												<div className="ch__name">Waiting for orchestrator…</div>
												<div className="ch__sub">scheduling pod</div>
											</div>
										</div>
									) : (
										channels.map((ch) => (
											<div className={`ch ch--${ch.state}`} key={ch.id}>
												<span className="ch__led" aria-hidden="true" />
												<div>
													<div className="ch__name">{scannerLabel(ch.id)}</div>
													<div className="ch__sub">
														{ch.state === 'done'
															? 'complete'
															: ch.state === 'run'
																? 'scanning…'
																: ch.state === 'err'
																	? 'did not finish'
																	: 'waiting for slot'}
													</div>
												</div>
												<div className="ch__right">
													<span className="ch__tag">{TAG_LABEL[ch.state]}</span>
												</div>
											</div>
										))
									)}
								</div>
							</section>

							<section className="panel log" aria-label="Scan log">
								<div className="panel__head">
									<span className="label">Stream</span>
									<span className="mono" style={{ fontSize: '.72rem', color: 'var(--ink-muted)' }}>
										stdout · {id}
									</span>
								</div>
								<div className="log__body" ref={logBodyRef} aria-live="polite">
									{logs.length === 0 ? (
										<div className="log__line">
											<span className="log__t" />
											<span className="log__m log__empty">waiting for output…</span>
										</div>
									) : (
										logs.map((line, i) => (
											<div className="log__line" key={i}>
												<span className="log__t" />
												<span className="log__m">{line}</span>
											</div>
										))
									)}
								</div>
							</section>
						</div>

						{/* sidebar */}
						<aside className="side" aria-label="Run summary">
							<div className="panel">
								<div className="panel__head">
									<span className="label">Progress</span>
									{isComplete ? (
										<Pill variant="done" style={{ fontSize: '.66rem' }}>
											done
										</Pill>
									) : isFailed ? (
										<Pill variant="error" style={{ fontSize: '.66rem' }}>
											failed
										</Pill>
									) : (
										<Pill variant="live" style={{ fontSize: '.66rem' }}>
											live
										</Pill>
									)}
								</div>
								<div className="panel__body">
									<div className="gaugewrap">
										<Gauge value={pct} caption="complete" size={120} valFontSize="1.9rem" />
									</div>
									{isComplete && (
										<Link
											className="btn btn--primary"
											to={`/scan/${id}/report`}
											style={{ width: '100%', justifyContent: 'center' }}
										>
											View report{' '}
											<span className="ar" aria-hidden="true">
												→
											</span>
										</Link>
									)}
								</div>
							</div>
							<div className="panel">
								<div className="panel__head">
									<span className="label">Artifacts</span>
								</div>
								<div className="panel__body" style={{ paddingBlock: '.4rem' }}>
									<div className={`artifact${shots > 0 ? '' : ' pending'}`}>
										<span className="ic">PNG</span>{' '}
										{shots > 0 ? `${shots} page screenshots` : 'page screenshots · pending'}
									</div>
									<div className={`artifact${artifacts?.report_json ? '' : ' pending'}`}>
										<span className="ic">JSON</span> unified report
										{artifacts?.report_json ? '' : ' · pending'}
									</div>
									<div className={`artifact${artifacts?.report_html ? '' : ' pending'}`}>
										<span className="ic">HTML</span> report bundle
										{artifacts?.report_html ? '' : ' · pending'}
									</div>
								</div>
							</div>
						</aside>
					</div>
				</div>
			</main>

			<SiteFooter />
		</>
	);
}

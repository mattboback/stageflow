import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams, type MetaFunction } from 'react-router';
import {
	CircleCheck,
	Clock,
	FileArchive,
	Info,
	KeyRound,
	Layers,
	Target
} from 'lucide-react';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Pill } from '../components/Pill';
import {
	fetchScanners,
	getDefaultScannerSelections,
	submitScanJob
} from '../lib/api/client';
import {
	buildAiNavigatorConfig,
	buildFormAuthConfig,
	DEFAULT_AI_CONFIG,
	isAuthConfigComplete,
	normalizeUrlInput,
	validateHttpUrls,
	validateZipUploadFile,
	type AiConfigState,
	type AuthFormConfig
} from '../lib/components/playground/playground-utils';
import type { ScannerDefinition, ScannerSelection } from '../lib/types/scan';
import { SCANNER_META, scannerLabel } from '../lib/report/scanner-identity';
import { PlaygroundAuthConfig } from '../components/playground/PlaygroundAuthConfig';
import { PlaygroundAiConfig } from '../components/playground/PlaygroundAiConfig';
import {
	buildSiteMeta,
	PLAYGROUND_DESCRIPTION,
	PLAYGROUND_TITLE
} from '../lib/site-metadata';
import playgroundStyles from './playground.css?url';

export const links = () => [{ rel: 'stylesheet', href: playgroundStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: PLAYGROUND_TITLE,
		description: PLAYGROUND_DESCRIPTION,
		path: '/playground'
	});

type Mode = 'url' | 'zip';

const CATEGORY_LABELS: Record<string, string> = {
	seo: 'SEO',
	ai: 'AI',
	custom: 'AI, opt-in'
};

function categoryLabel(s: string): string {
	if (CATEGORY_LABELS[s]) return CATEGORY_LABELS[s];
	return s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
}

export default function Playground() {
	const navigate = useNavigate();
	const [searchParams] = useSearchParams();

	const [catalog, setCatalog] = useState<ScannerDefinition[]>([]);
	const [selections, setSelections] = useState<ScannerSelection[]>([]);
	const [catalogError, setCatalogError] = useState<string | null>(null);
	const [catalogLoading, setCatalogLoading] = useState(true);

	const [mode, setMode] = useState<Mode>('url');
	const [urls, setUrls] = useState<string[]>(() => {
		const seed = searchParams.get('url');
		return seed ? [seed, ''] : ['https://example.com', ''];
	});
	const [file, setFile] = useState<File | null>(null);
	const fileInputRef = useRef<HTMLInputElement>(null);

	const [error, setError] = useState<string | null>(null);
	const [urlRowErrors, setUrlRowErrors] = useState<Record<number, string>>({});
	const [submitting, setSubmitting] = useState(false);

	const [authConfig, setAuthConfig] = useState<AuthFormConfig>({
		enabled: false,
		loginUrl: '',
		username: '',
		password: '',
		usernameSelector: '',
		passwordSelector: '',
		submitSelector: '',
		successStrategy: 'auto',
		successSelector: ''
	});
	const isAuthValid = isAuthConfigComplete(authConfig);

	const [aiConfig, setAiConfig] = useState<AiConfigState>(DEFAULT_AI_CONFIG);
	const isAiNavigatorEnabled = selections.some(
		(s) => s.id === 'ai-navigator' && s.enabled
	);

	useEffect(() => {
		const controller = new AbortController();
		fetchScanners(controller.signal)
			.then((res) => {
				setCatalog(res.scanners);
				setSelections(getDefaultScannerSelections(res.scanners));
				setCatalogError(null);
			})
			.catch((err: unknown) => {
				if (controller.signal.aborted) return;
				setCatalogError(err instanceof Error ? err.message : 'Failed to load scanners.');
			})
			.finally(() => {
				if (!controller.signal.aborted) setCatalogLoading(false);
			});
		return () => controller.abort();
	}, []);

	const enabledById = useMemo(() => {
		const map = new Map<string, boolean>();
		for (const s of selections) map.set(s.id, s.enabled);
		return map;
	}, [selections]);

	const armed = selections.filter((s) => s.enabled).length;
	const total = selections.length;

	function toggleScanner(id: string) {
		setSelections((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: !s.enabled } : s)));
	}

	function setAllScanners(enabled: boolean) {
		setSelections((prev) => prev.map((s) => ({ ...s, enabled })));
	}

	function updateUrl(index: number, value: string) {
		setUrls((prev) => prev.map((u, i) => (i === index ? value : u)));
		setUrlRowErrors((prev) => {
			if (!(index in prev)) return prev;
			const next = { ...prev };
			delete next[index];
			return next;
		});
	}

	function addUrlRow() {
		setUrls((prev) => [...prev, '']);
	}

	function removeUrlRow(index: number) {
		setUrls((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
		setUrlRowErrors({});
	}

	function onFilePicked(f: File | null) {
		if (!f) {
			setFile(null);
			return;
		}
		const problem = validateZipUploadFile(f);
		if (problem) {
			setError(problem);
			setFile(null);
			return;
		}
		setError(null);
		setFile(f);
	}

	async function runScan() {
		setError(null);
		setUrlRowErrors({});

		if (armed === 0) {
			setError('Enable at least one scanner.');
			return;
		}

		let validUrls: string[] = [];
		if (mode === 'url') {
			const rowErrors: Record<number, string> = {};
			const collected: string[] = [];
			urls.forEach((raw, i) => {
				const normalized = normalizeUrlInput(raw);
				if (!normalized) return;
				const { valid, invalid } = validateHttpUrls([normalized]);
				if (invalid.length > 0) {
					rowErrors[i] = invalid[0].reason;
				} else {
					collected.push(valid[0]);
				}
			});
			if (Object.keys(rowErrors).length > 0) {
				setUrlRowErrors(rowErrors);
				return;
			}
			if (collected.length === 0) {
				setError('Enter at least one URL to scan.');
				return;
			}
			validUrls = collected;
		} else if (!file) {
			setError('Select a ZIP archive to scan.');
			return;
		}

		if (authConfig.enabled && !isAuthValid) {
			setError('Authentication is enabled but Login URL, username, or password is missing.');
			return;
		}

		const auth = mode === 'url' ? buildFormAuthConfig(authConfig) : null;
		const scannersForSubmit =
			isAiNavigatorEnabled && aiConfig.objective.trim()
				? selections.map((s) =>
						s.id === 'ai-navigator'
							? { ...s, config: buildAiNavigatorConfig(aiConfig) }
							: s
					)
				: selections;

		setSubmitting(true);
		try {
			const { job_id } = await submitScanJob({
				mode,
				file,
				urls: validUrls,
				scanners: scannersForSubmit,
				highlightStyle: 'solid',
				auth
			});
			navigate(`/scan/${job_id}`);
		} catch (err: unknown) {
			setError(err instanceof Error ? err.message : 'Scan failed. Please try again.');
			setSubmitting(false);
		}
	}

	const estSeconds = Math.max(20, armed * 6);
	const targetCount = mode === 'url' ? urls.filter((u) => u.trim()).length : file ? 1 : 0;
	const authBlocking = mode === 'url' && authConfig.enabled && !isAuthValid;
	const ready =
		!catalogLoading && !catalogError && armed > 0 && targetCount > 0 && !authBlocking;
	const readyDetail = catalogLoading
		? 'Loading the scanner catalog.'
		: armed === 0
			? 'Enable at least one scanner.'
			: targetCount === 0
				? mode === 'url'
					? 'Add at least one URL to scan.'
					: 'Choose a ZIP archive to scan.'
				: authBlocking
					? 'Finish the authentication fields, or turn authentication off.'
					: "All set! You're ready to scan.";

	return (
		<>
			<SiteHeader current="playground" />

			<main id="main" className="console">
				<div className="wrap wrap--app">
					<div className="console__head">
						<div>
							<h1>Configure a scan</h1>
							<p>
								Point StageFlow at any public URL or a static-site archive, choose the scanners
								you need, and run. No account, no install.
							</p>
						</div>
						{submitting && <Pill variant="live">Submitting</Pill>}
					</div>

					<div className="console__grid">
						{/* left: configuration stack */}
						<div className="console__main">
							<section className="panel">
								<div className="panel__head">
									<div className="panel__head-copy">
										<h2>
											<span className="stepno num" aria-hidden="true">
												1
											</span>
											Target
										</h2>
										<p className="panel__sub">
											Add one or more URLs to scan, or upload a static-site archive.
										</p>
									</div>
									<div className="seg" role="group" aria-label="Target type">
										<button
											type="button"
											aria-pressed={mode === 'url'}
											onClick={() => setMode('url')}
										>
											URL
										</button>
										<button
											type="button"
											aria-pressed={mode === 'zip'}
											onClick={() => setMode('zip')}
										>
											ZIP upload
										</button>
									</div>
								</div>
								<div className="panel__body intake">
									{mode === 'url' ? (
										<>
											<div className="addurls">
												{urls.map((u, i) => (
													<div className="urlrow" key={i}>
														<div className="urlrow__field">
															<input
																className="input"
																type="url"
																value={u}
																placeholder="https://…"
																aria-label={`URL ${i + 1}`}
																aria-invalid={urlRowErrors[i] ? true : undefined}
																aria-describedby={
																	urlRowErrors[i] ? `urlrow-err-${i}` : undefined
																}
																onChange={(e) => updateUrl(i, e.target.value)}
															/>
															{urlRowErrors[i] && (
																<span
																	className="urlrow__err"
																	id={`urlrow-err-${i}`}
																	role="alert"
																>
																	{urlRowErrors[i]}
																</span>
															)}
														</div>
														<button
															type="button"
															className="rm"
															aria-label={`Remove URL ${i + 1}`}
															onClick={() => removeUrlRow(i)}
														>
															✕
														</button>
													</div>
												))}
											</div>
											<button type="button" className="addmore" onClick={addUrlRow}>
												+ Add another URL
											</button>
											<p className="intake__note">
												<Info size={15} aria-hidden="true" />
												Each URL is scanned as its own page — add every page you want
												covered.
											</p>
										</>
									) : (
										<>
											<button
												type="button"
												className="drop"
												onClick={() => fileInputRef.current?.click()}
											>
												<FileArchive size={22} aria-hidden="true" />
												<span className="drop__file">
													{file ? file.name : 'Choose a ZIP archive'}
												</span>
												<span className="drop__hint">.zip · static-site build · up to 100MB</span>
											</button>
											<input
												ref={fileInputRef}
												type="file"
												accept=".zip"
												hidden
												onChange={(e) => onFilePicked(e.target.files?.[0] ?? null)}
											/>
										</>
									)}
								</div>
							</section>

							<section className="panel">
								<div className="panel__head">
									<div className="panel__head-copy">
										<h2>
											<span className="stepno num" aria-hidden="true">
												2
											</span>
											Scanners
										</h2>
										<p className="panel__sub">Choose the scanners you want to run.</p>
									</div>
									<div className="chanhead__meta">
										<span className="chanhead__count num">
											{catalogLoading
												? 'Loading…'
												: armed === total
													? `${total} scanners enabled`
													: `${armed} of ${total} enabled`}
										</span>
										{!catalogLoading && !catalogError && (
											<button
												type="button"
												className="chanhead__bulk-btn"
												onClick={() => setAllScanners(armed < total)}
											>
												{armed < total ? 'Enable all' : 'Disable all'}
											</button>
										)}
									</div>
								</div>
								<div className="panel__body" style={{ padding: 0 }}>
									{catalogError ? (
										<div className="note note--err note--inset" role="status">
											<span className="note__ic" aria-hidden="true">
												!
											</span>
											<span>{catalogError}</span>
										</div>
									) : (
										<div className="rack" role="group" aria-label="Scanner channels">
											{catalog.map((scanner) => {
												const on = enabledById.get(scanner.id) ?? false;
												const meta = SCANNER_META[scanner.id];
												const Icon = meta?.icon ?? Layers;
												return (
													<button
														type="button"
														className="chan"
														role="checkbox"
														aria-checked={on}
														key={scanner.id}
														onClick={() => toggleScanner(scanner.id)}
													>
														<span className="chan__icon" aria-hidden="true">
															<Icon size={17} />
														</span>
														<span className="chan__main">
															<span className="chan__name">
																{scannerLabel(scanner.id, scanner.name)}
															</span>
															<span
																className="chan__desc"
																title={scanner.description || meta?.description || undefined}
															>
																{scanner.description || meta?.description || ''}
															</span>
														</span>
														<span className="chan__cat">
															{categoryLabel(scanner.categories[0] ?? '')}
														</span>
														<span className="tog" aria-hidden="true" />
													</button>
												);
											})}
										</div>
									)}
								</div>
							</section>

							{mode === 'url' && (
								<PlaygroundAuthConfig
									config={authConfig}
									isValid={isAuthValid}
									onConfigChange={setAuthConfig}
								/>
							)}

							{isAiNavigatorEnabled && (
								<PlaygroundAiConfig
									config={aiConfig}
									onConfigChange={setAiConfig}
								/>
							)}

						</div>

						{/* right: scan summary dock */}
						<aside className="dock" aria-label="Scan summary">
							<div className="panel">
								<div className="panel__head">
									<h2>Review & run</h2>
								</div>
								<div className="panel__body">
									<div
										className={ready ? 'readybox readybox--go' : 'readybox'}
										role="status"
										aria-live="polite"
									>
										<CircleCheck size={19} aria-hidden="true" />
										<span>
											<b>{ready ? 'Ready to run' : 'Almost there'}</b>
											{readyDetail}
										</span>
									</div>
									<ul className="sumlist">
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Target size={16} />
											</span>
											<span className="sumlist__lab">
												Targets
												<small>
													{mode === 'url'
														? 'Starting points for the scan'
														: 'Static-site archive'}
												</small>
											</span>
											<b className="num">
												{mode === 'url'
													? `${targetCount} ${targetCount === 1 ? 'URL' : 'URLs'}`
													: file
														? '1 archive'
														: 'None'}
											</b>
										</li>
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Layers size={16} />
											</span>
											<span className="sumlist__lab">
												Scanners
												<small>{armed === total ? 'All enabled' : 'Partial selection'}</small>
											</span>
											<b className="num">
												{armed} of {total || 8}
											</b>
										</li>
										{mode === 'url' && (
											<li>
												<span className="sumlist__icon" aria-hidden="true">
													<KeyRound size={16} />
												</span>
												<span className="sumlist__lab">
													Authentication
													<small>
														{authConfig.enabled
															? isAuthValid
																? 'Form login before scanning'
																: 'Required fields missing'
															: 'Scanning public pages'}
													</small>
												</span>
												<b>{authConfig.enabled ? (isAuthValid ? 'On' : 'Incomplete') : 'Off'}</b>
											</li>
										)}
										<li>
											<span className="sumlist__icon" aria-hidden="true">
												<Clock size={16} />
											</span>
											<span className="sumlist__lab">
												Estimated runtime
												<small>Per page, scanners run in parallel</small>
											</span>
											<b className="num">~ {estSeconds}s</b>
										</li>
									</ul>
									<button
										type="button"
										className="btn btn--primary btn--lg run"
										onClick={runScan}
										disabled={submitting || catalogLoading || armed === 0}
									>
										{submitting ? 'Submitting…' : 'Run scan'}{' '}
										{!submitting && (
											<span className="ar" aria-hidden="true">
												→
											</span>
										)}
									</button>
									<p className="runhint">No account required · Free to use</p>
								</div>
							</div>
							{error && (
								<div className="note note--err" role="status">
									<span className="note__ic" aria-hidden="true">
										!
									</span>
									<span>{error}</span>
								</div>
							)}
						</aside>
					</div>
				</div>

				{/* Mobile: the review summary collapses into one sticky action bar —
				    the dock's Run button is hidden there so only one Run exists. */}
				<div className="stickyrun">
					<span className="stickyrun__sum num">
						{mode === 'url'
							? `${targetCount} ${targetCount === 1 ? 'URL' : 'URLs'}`
							: file
								? '1 archive'
								: 'No archive'}{' '}
						· {armed} of {total || 8} scanners · auth{' '}
						{mode === 'url' && authConfig.enabled
							? isAuthValid
								? 'on'
								: 'incomplete'
							: 'off'}{' '}
						· ~{estSeconds}s
					</span>
					<button
						type="button"
						className="btn btn--primary"
						onClick={runScan}
						disabled={submitting || catalogLoading || armed === 0}
					>
						{submitting ? 'Submitting…' : 'Run scan'}{' '}
						{!submitting && (
							<span className="ar" aria-hidden="true">
								→
							</span>
						)}
					</button>
				</div>
			</main>

			<SiteFooter />
		</>
	);
}

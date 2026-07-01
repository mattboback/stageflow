import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams, type MetaFunction } from 'react-router';
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

function titleCase(s: string): string {
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

	function updateUrl(index: number, value: string) {
		setUrls((prev) => prev.map((u, i) => (i === index ? value : u)));
	}

	function addUrlRow() {
		setUrls((prev) => [...prev, '']);
	}

	function removeUrlRow(index: number) {
		setUrls((prev) => (prev.length <= 1 ? prev : prev.filter((_, i) => i !== index)));
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

		if (armed === 0) {
			setError('Arm at least one scanner channel.');
			return;
		}

		let validUrls: string[] = [];
		if (mode === 'url') {
			const normalized = urls
				.map((u) => normalizeUrlInput(u))
				.filter((u): u is string => Boolean(u));
			if (normalized.length === 0) {
				setError('Enter at least one URL to scan.');
				return;
			}
			const { valid, invalid } = validateHttpUrls(normalized);
			if (invalid.length > 0) {
				setError(`${invalid[0].url}: ${invalid[0].reason}`);
				return;
			}
			validUrls = valid;
		} else if (!file) {
			setError('Select a ZIP archive to scan.');
			return;
		}

		if (authConfig.enabled && !isAuthValid) {
			setError('Auth is enabled but Login URL / username / password are incomplete.');
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

	return (
		<>
			<SiteHeader current="playground" />

			<main id="main" className="console">
				<div className="wrap">
					<div className="console__head">
						<div>
							<span className="eyetag">Playground</span>
							<h1>Configure a scan.</h1>
							<p>
								Point StageFlow at any public URL or a static-site archive, arm the channels you
								need, and run. No account, no install.
							</p>
						</div>
						{submitting ? (
							<Pill variant="live">Submitting</Pill>
						) : (
							<Pill led ledColor="var(--ink-faint)">
								Idle
							</Pill>
						)}
					</div>

					<div className="console__grid">
						{/* left: configuration */}
						<div style={{ display: 'grid', gap: '1.5rem' }}>
							<section className="panel">
								<div className="panel__head">
									<span className="label">Target</span>
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
														<span className="ix">{i + 1}</span>
														<input
															className="input mono"
															type="url"
															value={u}
															placeholder="https://…"
															aria-label={`URL ${i + 1}`}
															onChange={(e) => updateUrl(i, e.target.value)}
														/>
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
												+ add another URL
											</button>
										</>
									) : (
										<>
											<button
												type="button"
												className="drop"
												onClick={() => fileInputRef.current?.click()}
											>
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
									<span className="label">Channels</span>
									<span className="mono" style={{ fontSize: '.72rem', color: 'var(--ink-muted)' }}>
										{catalogLoading ? 'loading…' : `${armed} / ${total} armed`}
									</span>
								</div>
								<div className="panel__body" style={{ padding: 0 }}>
									{catalogError ? (
										<div className="note note--err" role="status" style={{ margin: '1rem' }}>
											<span className="note__ic" aria-hidden="true">
												!
											</span>
											<span>{catalogError}</span>
										</div>
									) : (
										<div className="rack" role="group" aria-label="Scanner channels">
											{catalog.map((scanner) => {
												const on = enabledById.get(scanner.id) ?? false;
												return (
													<button
														type="button"
														className="chan"
														role="checkbox"
														aria-checked={on}
														key={scanner.id}
														onClick={() => toggleScanner(scanner.id)}
													>
														<span className="chan__main">
															<span className="chan__name">{scanner.name}</span>
															<span className="chan__cat">
																{titleCase(scanner.categories[0] ?? '')}
															</span>
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

						{/* right: run dock */}
						<aside className="dock" aria-label="Run controls">
							<div className="panel">
								<div className="panel__head">
									<span className="label">Ready to run</span>
									<Pill
										variant={armed > 0 ? 'live' : undefined}
										led
										ledColor={armed > 0 ? undefined : 'var(--ink-faint)'}
									>
										{armed > 0 ? 'Armed' : 'Idle'}
									</Pill>
								</div>
								<div className="panel__body">
									<div className="armline">
										<span className="label" style={{ alignSelf: 'center' }}>
											Channels armed
										</span>
										<b>
											{armed}
											<small>/{total || 8}</small>
										</b>
									</div>
									<hr className="divider" />
									<button
										type="button"
										className="btn btn--primary run"
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
									<p className="runhint">
										~ {estSeconds}s estimated · {targetCount}{' '}
										{mode === 'url' ? (targetCount === 1 ? 'URL' : 'URLs') : 'archive'} · {armed}{' '}
										{armed === 1 ? 'scanner' : 'scanners'}
									</p>
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
			</main>

			<SiteFooter />
		</>
	);
}

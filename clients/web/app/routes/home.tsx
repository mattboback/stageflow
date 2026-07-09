import { useState } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Gauge } from '../components/Gauge';
import { SeverityScale } from '../components/SeverityScale';
import { Pill } from '../components/Pill';
import { normalizeUrlInput, validateHttpUrls } from '../lib/components/playground/playground-utils';
import {
	buildSiteMeta,
	HOME_DESCRIPTION,
	HOME_TITLE,
	SITE_URL
} from '../lib/site-metadata';
import homeStyles from './home.css?url';

export const links = () => [{ rel: 'stylesheet', href: homeStyles }];

export const meta: MetaFunction = () => [
	...buildSiteMeta({
		title: HOME_TITLE,
		description: HOME_DESCRIPTION
	}),
	{
		'script:ld+json': {
			'@context': 'https://schema.org',
			'@type': 'SoftwareApplication',
			name: 'StageFlow',
			url: SITE_URL,
			applicationCategory: 'DeveloperApplication',
			description: HOME_DESCRIPTION
		}
	}
];

const PERF: { id: string; label: string; val: number; moderate?: boolean }[] = [
	{ id: 'a11y', label: 'Accessibility', val: 94 },
	{ id: 'perf', label: 'Performance', val: 88 },
	{ id: 'seo', label: 'SEO', val: 97 },
	{ id: 'sec', label: 'Security', val: 75, moderate: true },
	{ id: 'qual', label: 'Quality', val: 91 }
];

const SCANNERS: { name: string; sub: string; cat: string; off?: boolean }[] = [
	{ name: 'axe-core', sub: 'Accessibility rules, WCAG 2.1 mapping', cat: 'Accessibility' },
	{ name: 'Lighthouse', sub: 'Performance, best-practices, PWA signals', cat: 'Performance' },
	{ name: 'SEO', sub: 'Title, metadata, headings, structured data', cat: 'SEO' },
	{ name: 'security-headers', sub: 'HTTP response-header policy', cat: 'Security' },
	{ name: 'link-checker', sub: 'Internal & external link health', cat: 'Quality' },
	{ name: 'open-graph', sub: 'Social preview metadata', cat: 'SEO' },
	{ name: 'spelling-grammar', sub: 'Content quality checks', cat: 'Quality' },
	{ name: 'ai-navigator', sub: 'Natural-language navigation objectives', cat: 'AI · opt-in', off: true }
];

const BASELINE_STEPS = [
	{
		n: '1',
		title: 'Establish a baseline',
		body: 'Pick a known-good run as your baseline.'
	},
	{
		n: '2',
		title: 'Run on every deploy',
		body: 'Each scan compares against that baseline.'
	},
	{
		n: '3',
		title: 'See what changed',
		body: 'Only new or worse issues are highlighted.'
	}
] as const;

const PATH_STEPS = [
	{
		n: '01',
		title: 'Configure',
		body: 'Set the scope. Point at a URL or upload a ZIP archive.'
	},
	{
		n: '02',
		title: 'Run',
		body: 'Each scanner runs in isolation in a rootless container while progress streams live over SSE.'
	},
	{
		n: '03',
		title: 'Ship',
		body: 'Read the report. Merged findings with severity, evidence, and remediation in a single, triage-ready view.'
	}
] as const;

export default function Home() {
	const navigate = useNavigate();
	const [url, setUrl] = useState('https://example.com');
	const [urlError, setUrlError] = useState<string | null>(null);

	function onScan(e: React.FormEvent) {
		e.preventDefault();
		const normalized = normalizeUrlInput(url);
		if (!normalized) {
			setUrlError('Enter a URL to scan.');
			return;
		}
		const { valid, invalid } = validateHttpUrls([normalized]);
		if (invalid.length > 0) {
			setUrlError(invalid[0].reason);
			return;
		}
		setUrlError(null);
		navigate(`/playground?url=${encodeURIComponent(valid[0])}`);
	}

	return (
		<>
			<SiteHeader current="home" />

			<main id="main">
				{/* HERO — light, scan-first */}
				<section className="hero" aria-labelledby="hero-heading">
					<div className="rule-ticks" aria-hidden="true" />
					<div className="wrap hero__grid">
						<div className="hero__copy">
							<p className="hero__pills" aria-label="Product attributes">
								<span className="hero__pill">Self-hosted</span>
								<span className="hero__pill-sep" aria-hidden="true">
									·
								</span>
								<span className="hero__pill">Open source</span>
								<span className="hero__pill-sep" aria-hidden="true">
									·
								</span>
								<span className="hero__pill">CLI-first</span>
							</p>
							<h1 id="hero-heading">
								Every deploy answers one question:{' '}
								<span className="verdict">better</span>, or{' '}
								<span className="verdict">worse</span>.
							</h1>
							<p className="hero__lede">
								StageFlow runs eight scanners, compares against a baseline, and shows you exactly
								what regressed. Paste a URL and get a real report in seconds.
							</p>
							<form className="scanbar" onSubmit={onScan} noValidate>
								<label className="sr-only" htmlFor="url">
									URL to scan
								</label>
								<div className="scanbar__field">
									<input
										className="input mono"
										id="url"
										type="url"
										inputMode="url"
										autoComplete="url"
										placeholder="https://example.com"
										value={url}
										aria-invalid={urlError ? true : undefined}
										aria-describedby={urlError ? 'url-error' : undefined}
										onChange={(e) => {
											setUrl(e.target.value);
											if (urlError) setUrlError(null);
										}}
									/>
									{urlError && (
										<span className="scanbar__err" id="url-error" role="alert">
											{urlError}
										</span>
									)}
								</div>
								<button className="btn btn--primary" type="submit">
									Run a scan{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</button>
							</form>
							<div className="facts">
								<div className="readout">
									<span className="readout__val">8</span>
									<span className="readout__lab">scanners</span>
								</div>
								<div className="readout">
									<span className="readout__val">1</span>
									<span className="readout__lab">report</span>
								</div>
								<div className="readout">
									<span className="readout__val">CLI</span>
									<span className="readout__lab">first</span>
								</div>
								<div className="readout">
									<span className="readout__val">OPEN</span>
									<span className="readout__lab">source</span>
								</div>
							</div>
						</div>

						{/* Sample report preview — product chrome, not marketing card */}
						<aside className="panel hero__preview" aria-label="Sample report preview">
							<div className="panel__head">
								<span className="label">Sample report · example.com</span>
								<Pill variant="done">Completed</Pill>
							</div>
							<div className="panel__body">
								<div className="preview__top">
									<Gauge value={92} caption="quality score" size={132} valFontSize="2.1rem" />
									<div className="preview__cats">
										<p className="preview__cats-note muted">
											Weighted score across eight frontend quality areas.
										</p>
										<div className="perf-list">
											{PERF.map((p) => (
												<div className="perf" key={p.id}>
													<span className="perf__id">{p.label}</span>
													<span className="perf__bar">
														<i
															style={{
																width: `${p.val}%`,
																...(p.moderate ? { background: 'var(--sev-moderate)' } : {})
															}}
														/>
													</span>
													<span className="perf__val">{p.val}</span>
												</div>
											))}
										</div>
									</div>
								</div>
								<div className="preview__bottom">
									<SeverityScale counts={{ critical: 0, serious: 2, moderate: 5, minor: 11 }} />
									<p className="preview__baseline mono">
										<span className="preview__baseline-lab">Baseline</span>
										<span>score 88 → 92</span>
										<span className="preview__delta" aria-label="up four points">
											▲ 4
										</span>
									</p>
								</div>
							</div>
						</aside>
					</div>
				</section>

				{/* BASELINE IDEA — one deliberate 3-step sequence */}
				<section className="section section--band" aria-labelledby="baseline-heading">
					<div className="wrap baseline">
						<div className="baseline__intro">
							<h2 id="baseline-heading">The baseline idea</h2>
							<p>
								StageFlow gives every scan a point of reference, so changes are clear and
								actionable.
							</p>
						</div>
						<ol className="baseline__steps">
							{BASELINE_STEPS.map((s) => (
								<li className="baseline__step" key={s.n}>
									<span className="baseline__n mono" aria-hidden="true">
										{s.n}
									</span>
									<div>
										<h3>{s.title}</h3>
										<p>{s.body}</p>
									</div>
								</li>
							))}
						</ol>
					</div>
				</section>

				{/* SCANNERS + PATH */}
				<section className="section" id="scanners" aria-labelledby="scanners-heading">
					<div className="wrap mid">
						<div className="mid__scanners">
							<header className="section__head">
								<h2 id="scanners-heading">Eight scanners. One pass. One report.</h2>
								<p>
									Every channel runs in isolation and merges into a single severity-ranked report —
									the same shape from the CLI, CI, or the browser.
								</p>
							</header>
							<div className="array" role="table" aria-label="Built-in scanners">
								{SCANNERS.map((s) => (
									<div className="array__row" role="row" key={s.name}>
										<span className="array__name" role="cell">
											{s.name}
											<span>{s.sub}</span>
										</span>
										<span className="array__cat" role="cell">
											{s.cat}
										</span>
										{s.off ? (
											<span className="array__state array__state--off" role="cell">
												<i aria-hidden="true" />
												off
											</span>
										) : (
											<span className="array__state" role="cell">
												<i aria-hidden="true" />
												active
											</span>
										)}
									</div>
								))}
							</div>
						</div>

						<aside className="mid__path panel" aria-labelledby="path-heading">
							<div className="panel__body path">
								<header className="path__head">
									<h2 id="path-heading">From URL to report</h2>
									<p className="muted">Three steps. No setup required.</p>
								</header>
								<ol className="path__steps">
									{PATH_STEPS.map((s) => (
										<li className="path__step" key={s.n}>
											<span className="path__n mono" aria-hidden="true">
												{s.n}
											</span>
											<div>
												<h3>{s.title}</h3>
												<p>{s.body}</p>
											</div>
										</li>
									))}
								</ol>
								<div className="path__actions">
									<Link className="btn btn--primary" to="/playground">
										Open the playground{' '}
										<span className="ar" aria-hidden="true">
											→
										</span>
									</Link>
									<p className="path__cli mono" aria-label="Sample CLI command">
										$ stageflow scan https://example.com --fail-on serious
									</p>
								</div>
							</div>
						</aside>
					</div>
				</section>

				{/* CLI island + self-host */}
				<section className="section section--foot" aria-labelledby="cli-heading">
					<div className="wrap foot">
						<div className="terminal" aria-labelledby="cli-heading">
							<h2 id="cli-heading" className="sr-only">
								CLI
							</h2>
							<pre className="terminal__pre">
								<code>
									<span className="terminal__prompt">$</span> stageflow scan https://example.com
									{'\n'}
									{'  '}--fail-on serious
									{'\n'}
									{'\n'}
									<span className="terminal__ok">✓</span> 8 scanners
									{'\n'}
									<span className="terminal__ok">✓</span> Baseline: May 5, 2025 at 9:12 AM
									{'\n'}
									<span className="terminal__ok">✓</span> Completed in 14m 32s
									{'\n'}
									{'\n'}
									Report: reports/example-2025-05-07.html
								</code>
							</pre>
						</div>

						<div className="panel selfhost">
							<div className="panel__body">
								<h2>Self-hosted. CLI first. Built for teams.</h2>
								<p className="muted">
									StageFlow runs in your infrastructure and your data stays with you. Use the UI for
									exploration or the CLI for automation.
								</p>
								<ul className="selfhost__list">
									<li>
										<span className="selfhost__mark" aria-hidden="true" />
										Docker-based · rootless containers
									</li>
									<li>
										<span className="selfhost__mark" aria-hidden="true" />
										No SaaS lock-in · run anywhere
									</li>
									<li>
										<span className="selfhost__mark" aria-hidden="true" />
										Artifacts you own · HTML · JSON · SARIF
									</li>
								</ul>
								<div className="selfhost__actions">
									<a
										className="btn btn--ghost"
										href="https://github.com/mattboback/stageflow/tree/main/docs"
									>
										Documentation{' '}
										<span className="ar" aria-hidden="true">
											→
										</span>
									</a>
								</div>
							</div>
						</div>
					</div>
				</section>
			</main>

			<SiteFooter />
		</>
	);
}

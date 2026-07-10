import { useState } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';
import {
	Accessibility,
	Bot,
	Check,
	Gauge as GaugeIcon,
	Link2,
	Search,
	Share2,
	ShieldCheck,
	SpellCheck
} from 'lucide-react';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Delta } from '../components/Delta';
import { Gauge } from '../components/Gauge';
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
	{ id: 'a11y', label: 'Accessibility', val: 88, moderate: true },
	{ id: 'perf', label: 'Performance', val: 91 },
	{ id: 'seo', label: 'SEO', val: 97 },
	{ id: 'sec', label: 'Security', val: 75, moderate: true },
	{ id: 'qual', label: 'Quality', val: 93 }
];

/* One naming table: user-facing name first, implementation as the sub. */
const SCANNERS: {
	name: string;
	sub: string;
	cat: string;
	icon: React.ComponentType<{ size?: number | string; 'aria-hidden'?: boolean }>;
	optIn?: boolean;
}[] = [
	{ name: 'Axe Accessibility', sub: 'WCAG 2.1 rules, powered by axe-core', cat: 'Accessibility', icon: Accessibility },
	{ name: 'Lighthouse', sub: 'Google Lighthouse performance and best-practice audits', cat: 'Performance', icon: GaugeIcon },
	{ name: 'SEO', sub: 'Title, metadata, headings, structured data', cat: 'SEO', icon: Search },
	{ name: 'Security Headers', sub: 'HTTP response-header policy', cat: 'Security', icon: ShieldCheck },
	{ name: 'Link Checker', sub: 'Internal and external link health', cat: 'Quality', icon: Link2 },
	{ name: 'Open Graph', sub: 'Social preview metadata', cat: 'SEO', icon: Share2 },
	{ name: 'Spelling & Grammar', sub: 'Content quality checks', cat: 'Quality', icon: SpellCheck },
	{ name: 'AI Navigator', sub: 'Natural-language navigation objectives', cat: 'AI', icon: Bot, optIn: true }
];

const BASELINE_TRACK = [
	{
		score: '92',
		title: 'Establish a baseline',
		body: 'Pick a known-good run as your point of reference.'
	},
	{
		score: '88',
		title: 'Scan every deploy',
		body: 'Each new scan is compared against that baseline.'
	},
	{
		delta: -4,
		title: 'Review only what changed',
		body: 'New or worsened findings are highlighted; everything else stays quiet.'
	}
] as const;

const PATH_STEPS = [
	{
		n: '1',
		title: 'Configure',
		body: 'Point at a URL or upload a ZIP archive.'
	},
	{
		n: '2',
		title: 'Run',
		body: 'Each scanner runs in an isolated rootless container.'
	},
	{
		n: '3',
		title: 'Ship',
		body: 'One merged report with severity, evidence, and remediation.'
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
				{/* HERO — URL in, regression report out */}
				<section className="hero" aria-labelledby="hero-heading">
					<div className="wrap wrap--app hero__grid">
						<div className="hero__copy">
							<h1 id="hero-heading">
								Every deploy answers one question:{' '}
								<span className="verdict">better</span>, or{' '}
								<span className="verdict">worse</span>.
							</h1>
							<p className="hero__lede">
								StageFlow runs eight scanners, compares against a baseline, and shows you exactly
								what regressed. Self-hosted and open source. Paste a URL and get a real report in
								seconds.
							</p>
							<form className="scanbar" onSubmit={onScan} noValidate>
								<label className="sr-only" htmlFor="url">
									URL to scan
								</label>
								<div className="scanbar__field">
									<input
										className="input scanbar__input"
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
								<button className="btn btn--primary btn--lg" type="submit">
									Configure & run{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</button>
							</form>
							<p className="hero__note muted">
								No account, no install. Or run the same scan from the CLI in CI.
							</p>
						</div>

						{/* The output side of the scanbar: a report with a real regression */}
						<aside className="panel hero__preview" aria-label="Sample report preview">
							<div className="panel__head">
								<span className="label">
									<span className="hero__preview-arrow" aria-hidden="true">
										→
									</span>{' '}
									What comes back · example.com
								</span>
								<Pill variant="done">Completed</Pill>
							</div>
							<div className="panel__body">
								<div className="preview__top">
									<Gauge value={88} caption="Quality score" size={132} valFontSize="2.3rem" />
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
								<div className="preview__reg">
									<p className="preview__reg-title">
										<span className="sev-dot sev-serious" aria-hidden="true" /> Dashboard
										regressed since baseline
									</p>
									<ul className="preview__reg-rows">
										<li>
											<span>Accessibility</span>
											<span className="num">
												94 → 88 <Delta value={-6} />
											</span>
										</li>
										<li>
											<span>New serious issues</span>
											<span className="num">
												3 <Delta value={3} improvedWhenPositive={false} />
											</span>
										</li>
									</ul>
									<span className="preview__reg-cta">
										View evidence <span aria-hidden="true">→</span>
									</span>
								</div>
							</div>
						</aside>
					</div>
				</section>

				{/* BASELINE IDEA — before/after comparison track */}
				<section className="section section--band" aria-labelledby="baseline-heading">
					<div className="wrap wrap--app baseline">
						<div className="baseline__intro">
							<h2 id="baseline-heading">The baseline idea</h2>
							<p>
								StageFlow gives every scan a point of reference, so changes are clear and
								actionable.
							</p>
						</div>
						<ol className="basetrack">
							{BASELINE_TRACK.map((step, i) => (
								<li className="basetrack__node" key={step.title}>
									{i > 0 && (
										<span className="basetrack__link" aria-hidden="true">
											→
										</span>
									)}
									<div className="basetrack__card">
										<span className="basetrack__score num">
											{'score' in step ? step.score : <Delta value={step.delta} />}
										</span>
										<h3>{step.title}</h3>
										<p>{step.body}</p>
									</div>
								</li>
							))}
						</ol>
					</div>
				</section>

				{/* SCANNERS — list + how they merge into one report */}
				<section className="section" id="scanners" aria-labelledby="scanners-heading">
					<div className="wrap wrap--app mid">
						<div className="mid__scanners">
							<header className="section__head">
								<h2 id="scanners-heading">Eight scanners, one merged report</h2>
								<p>
									Every channel runs in isolation and merges into a single severity-ranked report,
									the same shape from the CLI, CI, or the browser.
								</p>
							</header>
							<div className="array" role="table" aria-label="Built-in scanners">
								{SCANNERS.map((s) => (
									<div className="array__row" role="row" key={s.name}>
										<span className="array__icon" aria-hidden="true">
											<s.icon size={18} />
										</span>
										<span className="array__name" role="cell">
											{s.name}
											<span>{s.sub}</span>
										</span>
										<span
											className={`array__cat${s.optIn ? ' array__cat--optin' : ''}`}
											role="cell"
										>
											{s.optIn ? 'Opt-in' : s.cat}
										</span>
									</div>
								))}
							</div>
						</div>

						<aside className="mid__path" aria-labelledby="path-heading">
							<div className="merge" aria-hidden="true">
								<ul className="merge__chips">
									{SCANNERS.map((s) => (
										<li key={s.name}>{s.name}</li>
									))}
								</ul>
								<span className="merge__brace" />
								<div className="merge__report panel">
									<span className="label">One report</span>
									<div className="merge__rows">
										<span className="merge__row">
											<span className="sev-dot sev-critical" /> 0 critical
										</span>
										<span className="merge__row">
											<span className="sev-dot sev-serious" /> 2 serious
										</span>
										<span className="merge__row">
											<span className="sev-dot sev-moderate" /> 5 moderate
										</span>
										<span className="merge__row">
											<span className="sev-dot sev-minor" /> 11 minor
										</span>
									</div>
								</div>
							</div>
							<div className="path">
								<header className="path__head">
									<h2 id="path-heading">From URL to report</h2>
									<p className="muted">Three steps. No setup required.</p>
								</header>
								<ol className="path__steps">
									{PATH_STEPS.map((s) => (
										<li className="path__step" key={s.n}>
											<span className="path__n" aria-hidden="true">
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
										Configure a scan{' '}
										<span className="ar" aria-hidden="true">
											→
										</span>
									</Link>
								</div>
							</div>
						</aside>
					</div>
				</section>

				{/* CLI island + self-host */}
				<section className="section section--foot" aria-labelledby="cli-heading">
					<div className="wrap wrap--app foot">
						<div className="terminal" aria-labelledby="cli-heading">
							<h2 id="cli-heading" className="sr-only">
								CLI
							</h2>
							<pre className="terminal__pre">
								<code>
									<span className="terminal__prompt">$</span> stageflow scan https://example.com
									--fail-on serious
									{'\n'}
									{'\n'}
									<span className="terminal__ok">✓</span> axe{'              '}21 issues · 36.8s
									{'\n'}
									<span className="terminal__ok">✓</span> lighthouse{'       '}82 issues · 82.6s
									{'\n'}
									<span className="terminal__ok">✓</span> seo{'              '}20 issues · 10.8s
									{'\n'}
									<span className="terminal__ok">✓</span> link-checker{'     '}20 issues · 23.2s
									{'\n'}
									<span className="terminal__ok">✓</span> 4 more scanners{'  '}completed
									{'\n'}
									{'\n'}
									Score: 88 (B) · serious 2 · moderate 5 · minor 11
									{'\n'}
									Baseline: May 5 → <span className="terminal__warn">3 new serious regressions</span>
									{'\n'}
									Report: reports/example-2026-07-10.html
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
										<span className="selfhost__mark" aria-hidden="true">
											<Check size={13} />
										</span>
										Docker-based, rootless containers
									</li>
									<li>
										<span className="selfhost__mark" aria-hidden="true">
											<Check size={13} />
										</span>
										No SaaS lock-in, run anywhere
									</li>
									<li>
										<span className="selfhost__mark" aria-hidden="true">
											<Check size={13} />
										</span>
										Artifacts you own: HTML, JSON, SARIF
									</li>
								</ul>
								<div className="selfhost__actions">
									<a
										className="btn btn--primary"
										href="https://github.com/mattboback/stageflow/tree/main/docs"
									>
										Documentation{' '}
										<span className="ar" aria-hidden="true">
											→
										</span>
									</a>
									<a className="btn btn--ghost" href="https://github.com/mattboback/stageflow">
										GitHub ↗
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

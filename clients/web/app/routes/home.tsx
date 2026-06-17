import { useState } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Gauge } from '../components/Gauge';
import { SeverityScale } from '../components/SeverityScale';
import { Pill } from '../components/Pill';
import homeStyles from './home.css?url';

const SITE_URL = 'https://stageflow.org';
const TAGLINE =
	'Self-hostable frontend quality platform. Eight scanners, one report, a baseline per project.';

export const links = () => [{ rel: 'stylesheet', href: homeStyles }];

export const meta: MetaFunction = () => [
	{ title: 'StageFlow — Frontend quality, measured' },
	{ name: 'description', content: TAGLINE },
	{
		'script:ld+json': {
			'@context': 'https://schema.org',
			'@type': 'SoftwareApplication',
			name: 'StageFlow',
			url: SITE_URL,
			applicationCategory: 'DeveloperApplication',
			description: TAGLINE
		}
	}
];

const PERF: { id: string; val: number; moderate?: boolean }[] = [
	{ id: 'axe', val: 94 },
	{ id: 'lighthouse', val: 88 },
	{ id: 'seo', val: 97 },
	{ id: 'sec-headers', val: 75, moderate: true }
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

export default function Home() {
	const navigate = useNavigate();
	const [url, setUrl] = useState('https://example.com');

	function onScan(e: React.FormEvent) {
		e.preventDefault();
		navigate(`/playground?url=${encodeURIComponent(url.trim())}`);
	}

	return (
		<>
			<SiteHeader current="home" />

			<main id="main">
				{/* HERO */}
				<section className="hero">
					<div className="wrap hero__grid">
						<div>
							<span className="eyetag">Frontend quality, measured</span>
							<h1>
								Did this change make the&nbsp;frontend <em>worse?</em>
							</h1>
							<p className="hero__lede">
								Eight scanners, one pipeline, one report. StageFlow remembers a baseline per
								project and tells you, in CI or the browser, exactly what regressed.
							</p>
							<form className="scanbar" onSubmit={onScan}>
								<label className="sr-only" htmlFor="url">
									URL to scan
								</label>
								<input
									className="input mono"
									id="url"
									type="url"
									inputMode="url"
									placeholder="https://example.com"
									value={url}
									onChange={(e) => setUrl(e.target.value)}
								/>
								<button className="btn btn--primary" type="submit">
									Scan{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</button>
							</form>
							<div className="facts">
								<div className="readout">
									<span className="readout__val">8</span>
									<span className="readout__lab">scanners / run</span>
								</div>
								<div className="readout">
									<span className="readout__val">WCAG&nbsp;AA</span>
									<span className="readout__lab">a11y baseline</span>
								</div>
								<div className="readout">
									<span className="readout__val">0</span>
									<span className="readout__lab">accounts required</span>
								</div>
							</div>
						</div>

						{/* instrument preview */}
						<div className="panel" style={{ boxShadow: 'var(--shadow-md)' }}>
							<div className="panel__head">
								<span className="label">Report · example.com</span>
								<Pill variant="done">Pass</Pill>
							</div>
							<div className="panel__body">
								<div className="preview__top">
									<Gauge value={92} caption="score" size={132} valFontSize="2.1rem" />
									<SeverityScale counts={{ critical: 0, serious: 2, moderate: 5, minor: 11 }} />
								</div>
								<div className="perf-list">
									{PERF.map((p) => (
										<div className="perf" key={p.id}>
											<span className="perf__id">{p.id}</span>
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
					</div>
				</section>

				{/* SCANNER ARRAY */}
				<section className="section" id="scanners">
					<div className="wrap">
						<div className="section__head">
							<span className="eyetag">The instrument</span>
							<h2>Eight scanners. One pass. One report contract.</h2>
							<p>
								Every channel runs in an isolated, rootless container and merges into a single
								severity-ranked report: the same shape from the CLI, CI, or the browser.
							</p>
						</div>
						<div className="array" role="table" aria-label="Built-in scanners">
							{SCANNERS.map((s) => (
								<div className="array__row" role="row" key={s.name}>
									<span className="array__name">
										{s.name}
										<span>{s.sub}</span>
									</span>
									<span className="array__cat">{s.cat}</span>
									{s.off ? (
										<span className="array__state" style={{ color: 'var(--ink-faint)' }}>
											<i style={{ background: 'var(--ink-faint)' }} />
											off
										</span>
									) : (
										<span className="array__state">
											<i />
											armed
										</span>
									)}
								</div>
							))}
						</div>
					</div>
				</section>

				{/* REGRESSION MEMORY */}
				<section className="section" style={{ paddingTop: 0 }}>
					<div className="wrap">
						<div className="regress">
							<div>
								<span className="eyetag" style={{ color: 'var(--signal)' }}>
									Regression memory
								</span>
								<h2 style={{ marginTop: '.7rem' }}>
									Promote a baseline once. Gate every scan after it.
								</h2>
								<p>
									Register a project, accept a known-good run, and every later scan diffs against
									it, exiting non-zero the moment a new issue appears or the severity gate trips.
									Straight into CI.
								</p>
							</div>
							<div className="diffcard" role="group" aria-label="Baseline diff readout">
								<div className="diffrow">
									<div className="diffstat">
										<b>88</b>
										<small>baseline</small>
									</div>
									<span className="delta up" aria-label="up four points">
										▲ 4
									</span>
									<div className="diffstat">
										<b>92</b>
										<small>this run</small>
									</div>
								</div>
								<div className="diffnew">
									<span className="badge">+2</span>
									<span style={{ color: 'oklch(0.78 0.01 240)' }}>
										new serious issues · gate:{' '}
										<strong style={{ color: '#fff' }}>fail</strong>
									</span>
								</div>
							</div>
						</div>
					</div>
				</section>

				{/* WORKFLOW */}
				<section className="section">
					<div className="wrap">
						<div className="section__head" style={{ marginBottom: '2.6rem' }}>
							<span className="eyetag">Operation</span>
							<h2>From URL to one report in three moves.</h2>
						</div>
						<div className="flow">
							<div className="step">
								<span className="step__no">01 · Configure</span>
								<h3>Set the scope</h3>
								<p>
									Point at URLs or upload a ZIP archive, then arm the scanners that matter for
									this release.
								</p>
							</div>
							<div className="step">
								<span className="step__no">02 · Run</span>
								<h3>Scan in isolation</h3>
								<p>Each scanner runs in a rootless container while progress streams live over SSE.</p>
							</div>
							<div className="step">
								<span className="step__no">03 · Ship</span>
								<h3>Read one report</h3>
								<p>
									Merged findings with severity, evidence, and remediation in a single,
									triage-ready view.
								</p>
							</div>
						</div>
					</div>
				</section>

				{/* CTA */}
				<section className="section" style={{ paddingTop: 0 }}>
					<div className="wrap">
						<div className="cta">
							<span className="eyetag">No account required</span>
							<h2 style={{ marginTop: '.8rem' }}>Measure your frontend in the next minute.</h2>
							<p>
								Run a hosted scan in the browser, or self-host the identical Podman stack. Same
								API, same report, your infrastructure.
							</p>
							<div className="cta__actions">
								<Link className="btn btn--primary" to="/playground">
									Open the playground{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</Link>
								<a className="btn btn--ghost" href="https://stageflow.org/docs">
									Read the docs
								</a>
							</div>
							<p className="cta__meta">$ stageflow scan https://example.com --fail-on serious</p>
						</div>
					</div>
				</section>
			</main>

			<SiteFooter />
		</>
	);
}

import { ScanProfileSelector } from '../components/ScanProfileSelector';
import { useState } from 'react';
import { Link, useNavigate, type MetaFunction } from 'react-router';

import { DiscoveryPreview } from '../components/DiscoveryPreview';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { Delta } from '../components/Delta';
import { DEMO_CROP } from '../lib/demo/demo-crop';
import { clamp, type Rect, type ViewBox } from '../lib/report/screenshot-crop';
import { DEMO_SUMMARY } from '../lib/demo/demo-summary';
import { discoverPages, submitScanJob, type DiscoverPagesResponse } from '../lib/api/client';
import { SCANNER_META } from '../lib/report/scanner-identity';
import { normalizeUrlInput, validateHttpUrl } from '../lib/url';
import {
	buildSiteMeta,
	HOME_DESCRIPTION,
	HOME_TITLE,
	IS_HOSTED_DEMO,
	SITE_NAME,
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
			name: SITE_NAME,
			url: SITE_URL,
			applicationCategory: 'DeveloperApplication',
			description: HOME_DESCRIPTION
		}
	}
];

const BASELINE_TRACK = [
	{
		score: String(DEMO_SUMMARY.baselineScore ?? '—'),
		title: 'Save a baseline',
		body: 'Choose a reviewed scan as your point of reference.'
	},
	{
		score: String(DEMO_SUMMARY.currentScore ?? '—'),
		title: 'Scan your next build',
		body: 'Inspect confirmed problems and actionable fixes.'
	},
	{
		delta: DEMO_SUMMARY.scoreDelta ?? 0,
		title: 'Review what changed',
		body: `${DEMO_SUMMARY.newIssues} new · ${DEMO_SUMMARY.fixedIssues} fixed in the real sample deployment.`
	}
] as const;

// The report's element crop centres a small window on the finding. At hero
// width that upscales a 48px icon into a blur, so show a card-sized window of
// the page instead, ending just past the element: in a left-to-right page the
// text that explains an element sits before it and above it.
function heroCrop(pageWidth: number, pageHeight: number, element: Rect): ViewBox {
	const width = Math.min(pageWidth, 1000);
	const height = Math.min(pageHeight, 270);
	return {
		x: clamp(element.x + element.width + 160 - width, 0, pageWidth - width),
		y: clamp(element.y + element.height / 2 - height * 0.66, 0, pageHeight - height),
		width,
		height
	};
}

const cropView = DEMO_CROP
	? heroCrop(DEMO_CROP.pageWidth, DEMO_CROP.pageHeight, DEMO_CROP.element)
	: null;

const [terminalCommand = '', ...terminalOutput] = DEMO_SUMMARY.terminal.split('\n');

const PATH_STEPS = [
	{
		n: '1',
		title: 'Configure',
		body: 'Point at a URL or upload a ZIP archive.'
	},
	{
		n: '2',
		title: 'Run',
		body: 'Review problems with page evidence and specific fixes.'
	},
	{
		n: '3',
		title: 'Ship',
		body: 'One merged report with severity, evidence, and remediation.'
	}
] as const;

export default function Home() {
	const navigate = useNavigate();
	const [url, setUrl] = useState('');
	const [profile, setProfile] = useState<'production' | 'staging'>('production');
	const [submitting, setSubmitting] = useState(false);
	const [urlError, setUrlError] = useState<string | null>(null);
	const [discovering, setDiscovering] = useState(false);
	const [discoveryPreview, setDiscoveryPreview] = useState<DiscoverPagesResponse | null>(null);

	function checkedSeed(): string | null {
		const normalized = normalizeUrlInput(url);
		if (!normalized) {
			setUrlError('Enter a URL to scan.');
			return null;
		}
		const checked = validateHttpUrl(normalized);
		if (!checked.ok) {
			setUrlError(checked.reason);
			return null;
		}
		setUrlError(null);
		return checked.url;
	}
	async function submitURLs(targets: string[]) {
		setSubmitting(true);
		setUrlError(null);
		try {
			const job = await submitScanJob({
				mode: 'url',
				file: null,
				urls: targets,
				profile,
				scanners: Object.keys(SCANNER_META).map((id) => ({ id, enabled: true })),
				screenshot: true
			});
			await navigate(`/scan/${encodeURIComponent(job.job_id)}`);
		} catch (error) {
			setUrlError(error instanceof Error ? error.message : 'Could not submit the scan.');
		} finally {
			setSubmitting(false);
		}
	}
	async function onScan(event: React.SyntheticEvent) {
		event.preventDefault();
		const seed = checkedSeed();
		if (seed) await submitURLs([seed]);
	}
	async function onScanWebsite() {
		const seed = checkedSeed();
		if (!seed) return;
		setDiscovering(true);
		setDiscoveryPreview(null);
		try {
			setDiscoveryPreview(await discoverPages(seed));
		} catch (error) {
			setUrlError(error instanceof Error ? error.message : 'Could not discover website pages.');
		} finally {
			setDiscovering(false);
		}
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
								Find what to fix. See what your <span className="verdict">deployment changed</span>.
							</h1>
							<p className="hero__lede">
								{SITE_NAME} checks accessibility, SEO, links, and security. Start with a prioritized
								fix list, then compare the next run with your saved baseline.
							</p>
							<form className="scanbar" onSubmit={(event) => void onScan(event)} noValidate>
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
										disabled={submitting || discovering}
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
								<button
									className="btn btn--primary btn--lg"
									type="submit"
									disabled={submitting || discovering}
								>
									{submitting ? 'Starting…' : 'Scan'}{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</button>
							</form>
							<div className="scanbar__opts">
								<button
									type="button"
									className="btn btn--ghost btn--sm"
									disabled={discovering || submitting}
									onClick={() => void onScanWebsite()}
								>
									{discovering ? 'Discovering website…' : 'Scan website'}
								</button>
								<ScanProfileSelector profile={profile} onChange={setProfile} />
							</div>
							{discoveryPreview && (
								<DiscoveryPreview
									key={JSON.stringify(discoveryPreview)}
									result={discoveryPreview}
									confirmLabel="Scan selected pages"
									busy={submitting}
									onConfirm={(selected) => void submitURLs(selected)}
									onCancel={() => setDiscoveryPreview(null)}
								/>
							)}
							<p className="hero__note muted">
								{IS_HOSTED_DEMO
									? 'No account. Artifacts expire after 24 hours. Local projects stay in this browser.'
									: 'No account, no install. Save a local project after the first report if you want a baseline.'}
							</p>
							<p className="hero__note muted">
								<Link to="/playground">Configure a scan</Link> ·{' '}
								<Link to="/projects">Compare releases in a local project</Link>
							</p>
						</div>

						{/* The output side of the scanbar. Every number below comes from the
						    committed demo report and is asserted against it in
						    app/lib/demo/demo-fixture.test.ts. */}
						<aside className="panel hero__preview" aria-label="Demo report summary">
							<div className="panel__head">
								<span className="label">
									<span className="hero__preview-arrow" aria-hidden="true">
										→
									</span>{' '}
									What comes back
								</span>
								<span className="hero__preview-site">sample.stageflow.org</span>
							</div>
							<div className="panel__body">
								<div className="preview__reg">
									<ul className="preview__stats" aria-label="Changes from baseline">
										<li className={DEMO_SUMMARY.newIssues > 0 ? 'preview__stat--bad' : undefined}>
											<span>New in this deploy</span>
											<strong className="num">{DEMO_SUMMARY.newIssues}</strong>
										</li>
										<li
											className={DEMO_SUMMARY.fixedIssues > 0 ? 'preview__stat--good' : undefined}
										>
											<span>Fixed</span>
											<strong className="num">{DEMO_SUMMARY.fixedIssues}</strong>
										</li>
										<li>
											<span>Score</span>
											<strong className="num">
												{DEMO_SUMMARY.currentScore ?? '—'}{' '}
												{DEMO_SUMMARY.scoreDelta !== null && (
													<Delta value={DEMO_SUMMARY.scoreDelta} />
												)}
											</strong>
											<small>baseline {DEMO_SUMMARY.baselineScore ?? '—'}</small>
										</li>
									</ul>
									{DEMO_CROP && cropView && (
										<Link
											className="preview__evidence"
											to={`/demo?section=issues&issue=${encodeURIComponent(DEMO_CROP.issueId)}`}
										>
											<span className="preview__evidence-head">
												<span className="preview__new">NEW</span>
												<span className="preview__evidence-title">{DEMO_CROP.title}</span>
												<span className="preview__evidence-open" aria-hidden="true">
													Open →
												</span>
											</span>
											<svg
												viewBox={`${cropView.x} ${cropView.y} ${cropView.width} ${cropView.height}`}
												style={{ aspectRatio: `${cropView.width} / ${cropView.height}` }}
												role="img"
												aria-label={`Evidence: ${DEMO_CROP.title}`}
											>
												<image
													href={DEMO_CROP.screenshot}
													width={DEMO_CROP.pageWidth}
													height={DEMO_CROP.pageHeight}
												/>
												<rect
													className="preview__mark"
													x={DEMO_CROP.element.x - 8}
													y={DEMO_CROP.element.y - 8}
													width={DEMO_CROP.element.width + 16}
													height={DEMO_CROP.element.height + 16}
													rx={6}
												/>
											</svg>
										</Link>
									)}
									<p className="preview__reg-note">
										Two real scans of the same sample site: inspect new problems, their evidence,
										and fixes.
									</p>

									<Link className="btn btn--primary" to="/demo">
										See what changed{' '}
										<span className="ar" aria-hidden="true">
											→
										</span>
									</Link>
								</div>
							</div>
						</aside>
					</div>
				</section>

				{/* BASELINE IDEA — the review step carries the weight */}
				<section className="section section--band" aria-labelledby="baseline-heading">
					<div className="wrap wrap--app baseline">
						<div className="baseline__intro">
							<h2 id="baseline-heading">The baseline idea</h2>
							<p>
								{SITE_NAME} gives every scan a point of reference, so changes are clear and
								actionable.
							</p>
						</div>
						<ol className="basetrack">
							{BASELINE_TRACK.map((step, i) => (
								<li
									className={`basetrack__node${i === BASELINE_TRACK.length - 1 ? ' basetrack__node--key' : ''}`}
									key={step.title}
								>
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

				{/* FROM URL TO REPORT — three steps, then the CTA */}
				<section className="section" aria-labelledby="path-heading">
					<div className="wrap wrap--app">
						<header className="section__head">
							<h2 id="path-heading">From URL to report</h2>
							<p>Three steps. No setup required.</p>
						</header>
						<ol className="path__steps path__steps--row">
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
							<Link className="btn btn--ghost" to="/projects">
								Or create a local project
							</Link>
						</div>
					</div>
				</section>

				{/* SCANNERS — the list the /#scanners anchor has always promised */}
				<section className="section section--band" id="scanners" aria-labelledby="scanners-heading">
					<div className="wrap wrap--app">
						<div className="scansum">
							<div className="scansum__copy">
								<h2 id="scanners-heading">Five scanners, one merged report</h2>
								<p>
									Each runs in its own rootless container and merges into a single severity-ranked
									report — the same shape from the CLI, from CI, or from the browser.
								</p>
							</div>
							<Link className="btn btn--ghost scansum__cta" to="/playground">
								Configure a scan{' '}
								<span className="ar" aria-hidden="true">
									→
								</span>
							</Link>
						</div>

						{/* Straight from SCANNER_META, the same map the configure page and the
						    report read. The anchor promised a list and delivered a paragraph;
						    duplicating the names in prose would have been a third place for
						    them to drift. */}
						<ul className="scanlist">
							{Object.entries(SCANNER_META).map(([id, meta]) => {
								const Icon = meta.icon;
								return (
									<li key={id} className="scanlist__item">
										<Icon className="scanlist__icon" size={18} aria-hidden="true" />
										<div>
											<h3 className="scanlist__name">{meta.label}</h3>
											<p className="scanlist__desc">{meta.description}</p>
										</div>
									</li>
								);
							})}
						</ul>
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
									<span className="terminal__prompt">$</span> {terminalCommand}
									{'\n'}
									{terminalOutput.join('\n')}
								</code>
							</pre>
						</div>

						<div className="panel foot__panel">
							<div className="panel__body">
								<h2>Put regressions in your CI summary.</h2>
								<p className="muted">
									Scan a build, compare it with the reviewed baseline, and open the full report for
									evidence and fixes.
								</p>
								<Link className="btn btn--primary" to="/demo">
									Explore the comparison →
								</Link>
							</div>
						</div>
					</div>
				</section>
			</main>

			<SiteFooter />
		</>
	);
}

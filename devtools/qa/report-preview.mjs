import fs from 'node:fs';
import path from 'node:path';
import { chromium } from '/home/matt/Deployment/stageflow/clients/web/node_modules/playwright/index.mjs';

const FIXTURE =
	'/home/matt/Deployment/stageflow/libs/contracts/report/fixtures/unified-report.v2.all-scans.json';
const OUT = '/home/matt/Deployment/stageflow/output/ui-refresh-preview';
const APP = 'http://localhost:5199';
const API = 'http://localhost:8080';

fs.mkdirSync(OUT, { recursive: true });

const report = JSON.parse(fs.readFileSync(FIXTURE, 'utf8'));
const byId = Object.fromEntries(report.issues.map((i) => [i.id, i]));

// ── Augment fixture: userImpact ─────────────────────────────────────────────
byId['axe-color-contrast-page-1'].userImpact = {
	statement:
		'Low-vision users cannot read the primary call-to-action against this background.',
	affectedGroups: ['low-vision', 'blind'],
	severity: 'blocking',
	userStory:
		'As a low-vision user, I cannot find the signup button because the text blends into the background.'
};
byId['axe-aria-input-field-name-page-1'].userImpact = {
	statement: 'Screen reader users hear an unlabeled input and cannot tell what to type.',
	affectedGroups: ['blind'],
	severity: 'degraded',
	userStory: 'As a screen reader user, I land on an input that only announces "edit text".'
};
byId['axe-image-alt-page-2'].userImpact = {
	statement: 'The team photo conveys no information to assistive technology.',
	affectedGroups: ['blind'],
	severity: 'inconvenient'
};

// ── Augment fixture: manual-review noise so the bucket renders ─────────────
const manualIssues = [
	{
		id: 'lh-manual-focus-visible-page-1',
		title: 'Interactive controls are keyboard focusable',
		ruleId: 'focusable-controls'
	},
	{
		id: 'lh-manual-focus-order-page-1',
		title: 'The page has a logical tab order',
		ruleId: 'logical-tab-order'
	},
	{
		id: 'lh-manual-offscreen-page-1',
		title: 'Offscreen content is hidden from assistive technology',
		ruleId: 'offscreen-content-hidden'
	}
].map((m) => ({
	id: m.id,
	scanner: 'lighthouse',
	ruleId: m.ruleId,
	severity: 'info',
	title: m.title,
	description: `Manually verify: ${m.title.toLowerCase()}.`,
	category: 'accessibility',
	pageId: 'page-1',
	pageUrl: 'http://localhost:8080/index.html',
	elementCount: 0
}));
report.issues.push(...manualIssues);
report.summary.totalIssues += manualIssues.length;
report.summary.bySeverity.info = (report.summary.bySeverity.info ?? 0) + manualIssues.length;
if (report.summary.byScanner?.lighthouse !== undefined)
	report.summary.byScanner.lighthouse += manualIssues.length;

// element boxes filled in after rendering the synthetic pages (real DOM coords)
function el(issueId, ruleId, severity, selector, box, W, H) {
	return {
		issueId,
		ruleId,
		severity,
		selector,
		nodeIndex: 0,
		x: box.x,
		y: box.y,
		width: box.width,
		height: box.height,
		xPercent: (box.x / W) * 100,
		yPercent: (box.y / H) * 100,
		widthPercent: (box.width / W) * 100,
		heightPercent: (box.height / H) * 100
	};
}

const jobStatus = {
	id: 'job-123',
	state: 'done',
	violations: report.summary.totalIssues,
	artifacts: {
		screenshots: [
			{
				kind: 'page_overview',
				artifact_id: 'shot-1',
				scanner_id: 'axe',
				page_id: 'page-1',
				url: `${API}/__mock/page-1.png`
			},
			{
				kind: 'page_overview',
				artifact_id: 'shot-2',
				scanner_id: 'axe',
				page_id: 'page-2',
				url: `${API}/__mock/page-2.png`
			}
		]
	},
	created_at: '2025-12-22T00:00:00Z',
	updated_at: '2025-12-22T00:00:12Z'
};

// ── Synthetic "scanned site" pages for the overlay screenshots ─────────────
const page1Html = `<!doctype html><html><head><meta charset="utf-8"><style>
body{margin:0;font-family:Georgia,serif;color:#222;width:1280px}
header{background:#16324f;color:#fff;padding:28px 80px;display:flex;justify-content:space-between;align-items:center}
nav a{color:#cfd8e3;margin-left:24px;text-decoration:none;font-size:15px}
#hero{background:linear-gradient(135deg,#eef3f8,#dce7f0);padding:90px 80px 110px}
#hero h1{font-size:44px;margin:0 0 14px}
#hero p{font-size:18px;color:#555;max-width:560px}
#hero a.cta{display:inline-block;margin-top:26px;background:#9fb4c7;color:#b8c9d9;padding:16px 46px;border-radius:8px;text-decoration:none;font-weight:bold}
section{padding:60px 80px}
.cards{display:flex;gap:24px}.card{flex:1;border:1px solid #e3e8ee;border-radius:12px;padding:24px;background:#fafbfc}
form{background:#f4f7fa;border-radius:12px;padding:32px;max-width:520px}
input{width:420px;padding:13px;border:1px solid #c8d2dc;border-radius:6px;font-size:15px}
a.more{color:#777;font-size:14px}
footer{background:#16324f;color:#9fb4c7;padding:48px 80px;margin-top:60px}
</style></head><body>
<header><b style="font-size:20px">Acme Studio</b><nav><a href="#">Work</a><a href="#">Services</a><a href="#">About</a><a href="#">Contact</a></nav></header>
<div id="hero"><h1>Design that ships.</h1><p>We build accessible, fast marketing sites for product teams that care about quality.</p>
<a class="cta" href="#">Start a project</a></div>
<section><h2>What we do</h2><div class="cards">
<div class="card"><h3>Brand systems</h3><p>Identity, voice, and component libraries that scale.</p></div>
<div class="card"><h3>Web platforms</h3><p>Marketing sites, docs, and storefronts on modern stacks.</p></div>
<div class="card"><h3>Audits</h3><p>Accessibility and performance reviews with fix plans.</p></div>
</div></section>
<section><h2>Stay in the loop</h2><form><label style="display:block;margin-bottom:8px;color:#555">Monthly notes on design &amp; a11y</label>
<input id="email" type="email" placeholder=""><button style="margin-left:10px;padding:13px 22px;border:0;background:#16324f;color:#fff;border-radius:6px">Subscribe</button></form>
<p style="margin-top:40px"><a class="more" href="">Read more case studies</a></p></section>
<footer>© 2026 Acme Studio — Crafted in Portland</footer>
</body></html>`;

const page2Html = `<!doctype html><html><head><meta charset="utf-8"><style>
body{margin:0;font-family:Georgia,serif;color:#222;width:1280px}
header{background:#16324f;color:#fff;padding:28px 80px}
section{padding:60px 80px}
img{border-radius:12px}
.bio{display:flex;gap:40px;align-items:flex-start}
</style></head><body>
<header><b style="font-size:20px">Acme Studio — About</b></header>
<section><h1>The team</h1><p style="max-width:560px;color:#555">Twelve people across four time zones, united by a love of well-built interfaces.</p>
<div class="bio"><svg id="team-photo" width="520" height="340" style="margin-left:300px;border-radius:12px"><rect width="520" height="340" fill="#dce7f0"/><circle cx="160" cy="140" r="60" fill="#9fb4c7"/><circle cx="320" cy="150" r="60" fill="#8aa3ba"/><rect x="90" y="220" width="340" height="80" rx="12" fill="#b8c9d9"/></svg></div>
<p style="margin-top:50px;color:#555;max-width:600px">We believe accessibility is a baseline, not a feature. Every engagement starts with an audit of the current experience.</p></section>
</body></html>`;

const browser = await chromium.launch();
const ctx0 = await browser.newContext({ viewport: { width: 1280, height: 900 } });
const shotPage = await ctx0.newPage();
const mockShots = {};

// page 1: screenshot + real element coords
await shotPage.setContent(page1Html, { waitUntil: 'load' });
const p1Size = await shotPage.evaluate(() => ({
	w: document.documentElement.scrollWidth,
	h: document.documentElement.scrollHeight
}));
const ctaBox = await shotPage.locator('#hero a.cta').boundingBox();
const emailBox = await shotPage.locator('#email').boundingBox();
const moreBox = await shotPage.locator('a.more').boundingBox();
mockShots['page-1.png'] = await shotPage.screenshot({ fullPage: true });

report.pages[0].pageOverview = {
	screenshotFilename: 'page-1.png',
	pageWidth: p1Size.w,
	pageHeight: p1Size.h,
	elements: [
		el('axe-color-contrast-page-1', 'color-contrast', 'critical', '#hero a.cta', ctaBox, p1Size.w, p1Size.h),
		el('axe-aria-input-field-name-page-1', 'aria-input-field-name', 'serious', '#email', emailBox, p1Size.w, p1Size.h),
		el('link-checker-empty-href-page-1', 'empty-href', 'serious', 'a.more', moreBox, p1Size.w, p1Size.h)
	]
};
byId['axe-color-contrast-page-1'].occurrences[0].boundingBox = ctaBox;
byId['axe-aria-input-field-name-page-1'].occurrences[0].boundingBox = emailBox;

// page 2
await shotPage.setContent(page2Html, { waitUntil: 'load' });
const p2Size = await shotPage.evaluate(() => ({
	w: document.documentElement.scrollWidth,
	h: document.documentElement.scrollHeight
}));
const photoBox = await shotPage.locator('#team-photo').boundingBox();
mockShots['page-2.png'] = await shotPage.screenshot({ fullPage: true });
report.pages[1].pageOverview = {
	screenshotFilename: 'page-2.png',
	pageWidth: p2Size.w,
	pageHeight: p2Size.h,
	elements: [el('axe-image-alt-page-2', 'image-alt', 'moderate', 'img#team-photo', photoBox, p2Size.w, p2Size.h)]
};
await ctx0.close();

// ── Drive the report UI with mocked API ────────────────────────────────────
const ctx = await browser.newContext({
	viewport: { width: 1500, height: 950 },
	deviceScaleFactor: 1.5
});
const page = await ctx.newPage();

await page.route('**/__mock/*.png', (route) => {
	const name = path.basename(new URL(route.request().url()).pathname);
	route.fulfill({ status: 200, contentType: 'image/png', body: mockShots[name] });
});
await page.route('**/api/v1/jobs/job-123/stream**', (route) =>
	route.fulfill({ status: 404, body: 'no stream' })
);
await page.route('**/api/v1/jobs/job-123/results**', (route) =>
	route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(report) })
);
await page.route('**/api/v1/jobs/job-123', (route) =>
	route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(jobStatus) })
);

async function snap(url, name, opts = {}) {
	await page.goto(url, { waitUntil: 'networkidle' });
	await page.waitForTimeout(opts.settle ?? 1600);
	if (opts.before) await opts.before();
	await page.screenshot({ path: `${OUT}/${name}.png`, fullPage: opts.fullPage ?? true });
	console.log('captured', name);
}

await snap(`${APP}/scan/job-123/report?section=overview`, '01-overview');
await snap(`${APP}/scan/job-123/report?section=issues`, '02-issues');

// expand the manual-review bucket
const bucket = page.getByTestId('manual-review-bucket');
if (await bucket.count()) {
	await bucket.locator('button').first().click();
	await page.waitForTimeout(500);
	await page.screenshot({ path: `${OUT}/02b-issues-manual-open.png`, fullPage: true });
	console.log('captured 02b-issues-manual-open');
} else {
	console.log('manual bucket not found');
}
await snap(`${APP}/scan/job-123/report`, '03-pages-visual');

// severity deep link from the header CTA
await page.goto(`${APP}/scan/job-123/report?section=overview`, { waitUntil: 'networkidle' });
await page.waitForTimeout(1200);
const cta = page.getByTestId('triage-cta');
if (await cta.count()) {
	await cta.click();
	await page.waitForTimeout(900);
	await page.screenshot({ path: `${OUT}/04-cta-deeplink.png`, fullPage: true });
	console.log('captured 04-cta-deeplink', page.url());
} else {
	console.log('no triage CTA found');
}

// issue modal with user impact
await page.goto(`${APP}/scan/job-123/report?section=issues&issueId=axe-color-contrast-page-1`, {
	waitUntil: 'networkidle'
});
await page.waitForTimeout(1400);
await page.screenshot({ path: `${OUT}/05-issue-modal.png`, fullPage: false });
console.log('captured 05-issue-modal');

// modal fix tab (user impact callout)
const fixTab = page.getByRole('tab', { name: 'Fix' }).first();
if (await fixTab.count()) {
	await fixTab.click();
	await page.waitForTimeout(500);
	await page.screenshot({ path: `${OUT}/06-issue-modal-fix.png`, fullPage: false });
	console.log('captured 06-issue-modal-fix');
}

await browser.close();
console.log('done ->', OUT);

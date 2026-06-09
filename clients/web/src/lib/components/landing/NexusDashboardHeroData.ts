// Static, high-fidelity mock data for the decorative landing-page dashboard hero.
// Pure presentation data — no runtime behavior. Consumed by NexusDashboardHero
// and its tab sub-components.

export type TabName = 'Overview' | 'Issues' | 'Pages' | 'Scanners';

export const navTabs: readonly TabName[] = ['Overview', 'Issues', 'Pages', 'Scanners'];

export const scanners = [
	{ name: 'Axe', score: 100, status: 'done', color: '#0d5c63', bg: '#e6f4f4' },
	{ name: 'Lighthouse', score: 89, status: 'done', color: '#f59e0b', bg: '#fffbeb' },
	{ name: 'Security Headers', score: 94, status: 'done', color: '#8b5cf6', bg: '#f5f3ff' },
	{ name: 'SEO', score: 82, status: 'done', color: '#f43f5e', bg: '#fff1f2' },
	{ name: 'Link Checker', score: 91, status: 'done', color: '#10b981', bg: '#f0fdf4' },
	{ name: 'AI Navigator', score: 78, status: 'done', color: '#d946ef', bg: '#fdf4ff' },
	{ name: 'Open Graph', score: 95, status: 'done', color: '#06b6d4', bg: '#ecfeff' },
	{ name: 'Spelling & Grammar', score: 88, status: 'done', color: '#65a30d', bg: '#f7fee7' }
] as const;

export const categoryScores = [
	{ label: 'Accessibility', score: 100, color: '#0d5c63' },
	{ label: 'Performance', score: 89, color: '#f59e0b' },
	{ label: 'SEO', score: 82, color: '#f43f5e' },
	{ label: 'Security', score: 94, color: '#8b5cf6' }
] as const;

export const findings = [
	{ label: 'Critical', count: 3, color: '#dc2626', bg: '#fef2f2' },
	{ label: 'Serious', count: 8, color: '#ea580c', bg: '#fff7ed' },
	{ label: 'Moderate', count: 14, color: '#d97706', bg: '#fffbeb' }
] as const;

const severityTotal = findings.reduce((sum, f) => sum + f.count, 0);

export const severitySegments = findings.map((f) => ({
	...f,
	width: (f.count / severityTotal) * 100
}));

export const trendBars = [38, 44, 52, 48, 61, 68, 72, 70, 79, 84, 88, 94] as const;

export const mockIssues = [
	{
		id: 1,
		title: 'Insufficient color contrast ratio (2.3:1) on secondary action buttons',
		impact: 'Critical',
		selector: 'button.btn-secondary',
		code: '<button class="bg-[#a5e] text-white">Subscribe</button>',
		remediation:
			'Increase contrast ratio to at least 4.5:1 by darkening the background color to HSL(267, 75%, 35%).'
	},
	{
		id: 2,
		title: 'Input field does not have an associated explicit label element',
		impact: 'Serious',
		selector: 'input#newsletter-email',
		code: '<input type="email" id="newsletter-email" placeholder="Enter email" />',
		remediation:
			'Wrap input with a descriptive label or add a matching label element with the attribute for="newsletter-email".'
	},
	{
		id: 3,
		title: 'Strict-Transport-Security (HSTS) security response header is missing',
		impact: 'Moderate',
		selector: 'HTTP Response Headers',
		code: 'Header missing: Strict-Transport-Security',
		remediation:
			'Configure your web server to append "Strict-Transport-Security: max-age=63072000; includeSubDomains; preload" response header.'
	}
] as const;

export const mockPages = [
	{ path: '/', label: 'Landing Page', issues: 0, score: 100, loadTime: '0.4s' },
	{
		path: '/playground',
		label: 'Playground Configuration',
		issues: 11,
		score: 82,
		loadTime: '0.9s'
	},
	{
		path: '/scan/status',
		label: 'Real-time Progress Tracker',
		issues: 6,
		score: 88,
		loadTime: '0.7s'
	},
	{
		path: '/report/staging',
		label: 'Audited Report Summary',
		issues: 8,
		score: 94,
		loadTime: '0.5s'
	}
] as const;

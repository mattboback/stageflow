<script lang="ts">
	import { fade } from 'svelte/transition';

	const navTabs = ['Overview', 'Issues', 'Pages', 'Scanners'] as const;

	let activeTab = $state<'Overview' | 'Issues' | 'Pages' | 'Scanners'>('Overview');
	let expandedIssueIdx = $state<number | null>(null);

	const scanners = [
		{ name: 'Axe', score: 100, status: 'done', color: '#0d5c63', bg: '#e6f4f4' },
		{ name: 'Lighthouse', score: 89, status: 'done', color: '#f59e0b', bg: '#fffbeb' },
		{ name: 'Security Headers', score: 94, status: 'done', color: '#8b5cf6', bg: '#f5f3ff' },
		{ name: 'SEO', score: 82, status: 'done', color: '#f43f5e', bg: '#fff1f2' },
		{ name: 'Link Checker', score: 91, status: 'done', color: '#10b981', bg: '#f0fdf4' },
		{ name: 'AI Navigator', score: 78, status: 'done', color: '#d946ef', bg: '#fdf4ff' },
		{ name: 'Open Graph', score: 95, status: 'done', color: '#06b6d4', bg: '#ecfeff' },
		{ name: 'Spelling & Grammar', score: 88, status: 'done', color: '#65a30d', bg: '#f7fee7' }
	] as const;

	const categoryScores = [
		{ label: 'Accessibility', score: 100, color: '#0d5c63' },
		{ label: 'Performance', score: 89, color: '#f59e0b' },
		{ label: 'SEO', score: 82, color: '#f43f5e' },
		{ label: 'Security', score: 94, color: '#8b5cf6' }
	] as const;

	const findings = [
		{ label: 'Critical', count: 3, color: '#dc2626', bg: '#fef2f2' },
		{ label: 'Serious', count: 8, color: '#ea580c', bg: '#fff7ed' },
		{ label: 'Moderate', count: 14, color: '#d97706', bg: '#fffbeb' }
	] as const;

	const severityTotal = findings.reduce((sum, f) => sum + f.count, 0);
	const severitySegments = findings.map((f) => ({
		...f,
		width: (f.count / severityTotal) * 100
	}));

	const trendBars = [38, 44, 52, 48, 61, 68, 72, 70, 79, 84, 88, 94] as const;

	// Simulated high-fidelity data for interactive subviews
	const mockIssues = [
		{
			id: 1,
			title: 'Insufficient color contrast ratio (2.3:1) on secondary action buttons',
			impact: 'Critical',
			selector: 'button.btn-secondary',
			code: '<button class="bg-[#a5e] text-white">Subscribe</button>',
			remediation: 'Increase contrast ratio to at least 4.5:1 by darkening the background color to HSL(267, 75%, 35%).'
		},
		{
			id: 2,
			title: 'Input field does not have an associated explicit label element',
			impact: 'Serious',
			selector: 'input#newsletter-email',
			code: '<input type="email" id="newsletter-email" placeholder="Enter email" />',
			remediation: 'Wrap input with a descriptive label or add a matching label element with the attribute for="newsletter-email".'
		},
		{
			id: 3,
			title: 'Strict-Transport-Security (HSTS) security response header is missing',
			impact: 'Moderate',
			selector: 'HTTP Response Headers',
			code: 'Header missing: Strict-Transport-Security',
			remediation: 'Configure your web server to append "Strict-Transport-Security: max-age=63072000; includeSubDomains; preload" response header.'
		}
	] as const;

	const mockPages = [
		{ path: '/', label: 'Landing Page', issues: 0, score: 100, loadTime: '0.4s' },
		{ path: '/playground', label: 'Playground Configuration', issues: 11, score: 82, loadTime: '0.9s' },
		{ path: '/scan/status', label: 'Real-time Progress Tracker', issues: 6, score: 88, loadTime: '0.7s' },
		{ path: '/report/staging', label: 'Audited Report Summary', issues: 8, score: 94, loadTime: '0.5s' }
	] as const;
</script>

<div class="nf-frame" aria-hidden="true">
	<!-- Browser chrome bar -->
	<div class="nf-chrome">
		<div class="nf-dots">
			<i class="nf-dot" style="background:#fc5f57"></i>
			<i class="nf-dot" style="background:#fdbc2d"></i>
			<i class="nf-dot" style="background:#34c749"></i>
		</div>
		<div class="nf-urlbar">stageflow.org/report/staging-deploy</div>
	</div>

	<!-- White dashboard card -->
	<div class="nf-card">
		<!-- Product nav strip -->
		<div class="nf-nav">
			<div class="nf-logo">
				<div class="nf-logo-icon">SF</div>
				<span class="nf-logo-text">StageFlow</span>
			</div>
			<div class="nf-tabs">
				{#each navTabs as tab (tab)}
					<button
						type="button"
						class="nf-tab"
						class:nf-tab-active={activeTab === tab}
						onclick={() => {
							activeTab = tab;
							expandedIssueIdx = null;
						}}
					>
						{tab}
					</button>
				{/each}
			</div>
		</div>

		<!-- Conditional Svelte Views -->
		{#if activeTab === 'Overview'}
			<div in:fade={{ duration: 120 }}>
				<!-- Main bento: scan overview (left 7) + scanners (right 5) -->
				<div class="nf-bento-main">
					<!-- Left: Scan Overview panel -->
					<div class="nf-panel-left">
						<div class="nf-panel-toprow">
							<span class="nf-panel-title">Scan Overview</span>
							<span class="nf-pill">Latest run</span>
						</div>
						<div class="nf-score-area">
							<div class="nf-score-left">
								<span class="nf-status-pill">
									<span class="nf-status-dot"></span>
									Strong
								</span>
								<div class="nf-score-num">94<span class="nf-score-denom">/100</span></div>
								<p class="nf-score-desc">staging.example.com · 4 pages · 2m 18s</p>
								<div class="nf-sev-bar" aria-label="Severity distribution">
									{#each severitySegments as seg (seg.label)}
										<span
											class="nf-sev-seg"
											style="width:{seg.width}%; background:{seg.color}"
											title={`${seg.count} ${seg.label}`}
										></span>
									{/each}
								</div>
							</div>
							<div class="nf-score-chart">
								{#each categoryScores as cat (cat.label)}
									<div class="nf-cat-row">
										<span class="nf-cat-label">{cat.label}</span>
										<div class="nf-cat-track">
											<div class="nf-cat-fill" style="width:{cat.score}%; background:{cat.color}"></div>
										</div>
										<span class="nf-cat-num">{cat.score}</span>
									</div>
								{/each}
							</div>
						</div>
					</div>

					<!-- Right: Active Scanners panel -->
					<div class="nf-panel-right">
						<div class="nf-panel-toprow">
							<span class="nf-panel-title">Active Scanners</span>
							<span class="nf-see-all">Run scan →</span>
						</div>
						<div class="nf-scanner-list">
							{#each scanners.slice(0, 4) as s (s.name)}
								<div class="nf-scanner-item">
									<div class="nf-scanner-icon" style="background:{s.bg}">
										<div class="nf-scanner-dot" style="background:{s.color}"></div>
									</div>
									<span class="nf-scanner-name">{s.name}</span>
									<span class="nf-badge nf-badge-done">DONE</span>
									<span class="nf-scanner-score">{s.score}</span>
								</div>
							{/each}
						</div>
					</div>
				</div>

				<!-- Bottom row: findings | cta | score trend -->
				<div class="nf-bento-bottom">
					<!-- Findings -->
					<div class="nf-bottom-panel">
						<div class="nf-panel-toprow">
							<span class="nf-panel-title">Findings</span>
							<button type="button" class="nf-see-all-btn" onclick={() => activeTab = 'Issues'}>View details →</button>
						</div>
						<div class="nf-findings">
							{#each findings as f (f.label)}
								<div class="nf-finding-row">
									<div class="nf-finding-avatar" style="background:{f.bg}">
										<span style="color:{f.color}; font-size:9px; font-weight:800">{f.label[0]}</span>
									</div>
									<span class="nf-finding-label">{f.label}</span>
									<span class="nf-finding-count" style="color:{f.color}">{f.count} found</span>
									<span class="nf-finding-caret">›</span>
								</div>
							{/each}
						</div>
					</div>

					<!-- CTA card (teal-tinted, like Nexus lavender "Upgrade to Pro") -->
					<div class="nf-bottom-panel nf-cta-card">
						<p class="nf-cta-title">One run. All scanners.</p>
						<p class="nf-cta-desc">
							No account required. Accessibility, performance, SEO, and security in one execution path.
						</p>
						<a href="/playground" class="nf-cta-btn">Start scanning →</a>
					</div>

					<!-- Score Trend (like Nexus "Funnel Metrics") -->
					<div class="nf-bottom-panel">
						<div class="nf-panel-toprow">
							<span class="nf-panel-title">Score Trend</span>
						</div>
						<div class="nf-trend-stats">
							<div class="nf-trend-stat">
								<span class="nf-trend-num">8</span>
								<span class="nf-trend-lbl">Scanners</span>
							</div>
							<div class="nf-stat-div"></div>
							<div class="nf-trend-stat">
								<span class="nf-trend-num">25</span>
								<span class="nf-trend-lbl">Issues</span>
							</div>
							<div class="nf-stat-div"></div>
							<div class="nf-trend-stat">
								<span class="nf-trend-num" style="color:#0d5c63">+12</span>
								<span class="nf-trend-lbl">Score ↑</span>
							</div>
						</div>
						<div class="nf-barchart">
							{#each trendBars as h, i (i)}
								<div
									class="nf-bar"
									class:nf-bar-current={i === trendBars.length - 1}
									style="height:{h}%"
								></div>
							{/each}
						</div>
					</div>
				</div>
			</div>
		{:else if activeTab === 'Issues'}
			<div class="nf-tab-container" in:fade={{ duration: 120 }}>
				<div class="nf-section-header">
					<div class="nf-panel-toprow">
						<span class="nf-panel-title">Interactive Audit Findings</span>
						<span class="nf-pill">Click items to view remediation code</span>
					</div>
				</div>
				<div class="nf-issues-list">
					{#each mockIssues as issue, idx (issue.id)}
						<button
							type="button"
							class="nf-interactive-issue-row"
							class:nf-issue-expanded={expandedIssueIdx === idx}
							onclick={() => expandedIssueIdx = expandedIssueIdx === idx ? null : idx}
						>
							<div class="nf-issue-meta">
								<span
									class="nf-issue-badge"
									class:nf-badge-crit={issue.impact === 'Critical'}
									class:nf-badge-ser={issue.impact === 'Serious'}
									class:nf-badge-mod={issue.impact === 'Moderate'}
								>
									{issue.impact}
								</span>
								<span class="nf-issue-selector-pill">{issue.selector}</span>
							</div>
							<p class="nf-interactive-issue-title">{issue.title}</p>
							
							{#if expandedIssueIdx === idx}
								<div class="nf-issue-drawer" in:fade={{ duration: 100 }}>
									<p class="nf-drawer-heading">HTML Evidence</p>
									<div class="nf-issue-code-box">
										<code>{issue.code}</code>
									</div>
									<p class="nf-drawer-heading mt-2">Remediation Guidance</p>
									<p class="nf-drawer-body">{issue.remediation}</p>
								</div>
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{:else if activeTab === 'Pages'}
			<div class="nf-tab-container" in:fade={{ duration: 120 }}>
				<div class="nf-section-header">
					<div class="nf-panel-toprow">
						<span class="nf-panel-title">Site Hierarchy & Crawled Index</span>
						<span class="nf-pill">4 entries audited</span>
					</div>
				</div>
				<div class="nf-pages-grid">
					{#each mockPages as p (p.path)}
						<div class="nf-page-card">
							<div class="nf-page-header">
								<span class="nf-page-path font-mono">{p.path}</span>
								<span class="nf-page-badge" class:nf-page-badge-success={p.issues === 0}>
									{p.issues === 0 ? 'CLEAN' : `${p.issues} FINDINGS`}
								</span>
							</div>
							<p class="nf-page-label">{p.label}</p>
							<div class="nf-page-metrics">
								<div class="nf-page-metric">
									<span class="nf-metric-label">Score</span>
									<span class="nf-metric-value" style="color: {p.score > 90 ? '#10b981' : '#f59e0b'}">{p.score}</span>
								</div>
								<div class="nf-page-metric">
									<span class="nf-metric-label">Audited</span>
									<span class="nf-metric-value text-ink-muted">{p.loadTime}</span>
								</div>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{:else if activeTab === 'Scanners'}
			<div class="nf-tab-container" in:fade={{ duration: 120 }}>
				<div class="nf-section-header">
					<div class="nf-panel-toprow">
						<span class="nf-panel-title">Auditing Engines Status</span>
						<span class="nf-pill">All tests completed</span>
					</div>
				</div>
				<div class="nf-scanners-grid">
					{#each scanners as s (s.name)}
						<div class="nf-scanner-card">
							<div class="nf-scanner-card-header">
								<div class="nf-scanner-circle-icon" style="background:{s.bg}">
									<div class="nf-scanner-circle-dot" style="background:{s.color}"></div>
								</div>
								<span class="nf-scanner-card-name">{s.name}</span>
								<span class="nf-scanner-card-score">{s.score}</span>
							</div>
							<div class="nf-scanner-card-status">
								<span class="nf-status-wave"></span>
								Active diagnostics verified
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	/* Outer slate frame — the "device" wrapper */
	.nf-frame {
		background: #d1d9e0;
		border-radius: 1.5rem;
		padding: 0.875rem 0.875rem 0;
		box-shadow:
			0 24px 56px rgba(0, 0, 0, 0.1),
			0 8px 16px rgba(0, 0, 0, 0.06);
		overflow: hidden;
	}

	/* Chrome bar */
	.nf-chrome {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding-bottom: 0.75rem;
	}

	.nf-dots {
		display: flex;
		gap: 5px;
		flex-shrink: 0;
	}

	.nf-dot {
		display: block;
		width: 11px;
		height: 11px;
		border-radius: 50%;
	}

	.nf-urlbar {
		flex: 1;
		background: rgba(255, 255, 255, 0.45);
		border-radius: 6px;
		padding: 4px 12px;
		font-size: 11px;
		color: #475569;
		font-family: ui-monospace, monospace;
		text-align: center;
		border: 1px solid rgba(255, 255, 255, 0.3);
	}

	/* White dashboard card */
	.nf-card {
		background: #ffffff;
		border-radius: 1rem 1rem 0 0;
		overflow: hidden;
		min-height: 420px;
		display: flex;
		flex-direction: column;
	}

	/* Product nav */
	.nf-nav {
		display: flex;
		align-items: center;
		gap: 1.5rem;
		padding: 0.75rem 1.25rem;
		border-bottom: 1px solid #e2e8f0;
		background: #ffffff;
		flex-shrink: 0;
	}

	.nf-logo {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-shrink: 0;
	}

	.nf-logo-icon {
		width: 28px;
		height: 28px;
		background: #0d5c63;
		border-radius: 7px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 10px;
		font-weight: 700;
		color: white;
		letter-spacing: 0.02em;
		font-family: inherit;
	}

	.nf-logo-text {
		font-weight: 700;
		font-size: 13px;
		color: #0f172a;
	}

	.nf-tabs {
		display: flex;
		gap: 0.25rem;
	}

	.nf-tab {
		padding: 0.375rem 0.875rem;
		border-radius: 9999px;
		font-size: 12px;
		font-weight: 600;
		color: #64748b;
		cursor: pointer;
		white-space: nowrap;
		background: transparent;
		border: none;
		outline: none;
		transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
	}

	.nf-tab:hover {
		color: #0d5c63;
		background: #e6f4f4;
	}

	.nf-tab-active {
		background: #0d5c63;
		color: white !important;
	}

	/* Tab view containers */
	.nf-tab-container {
		padding: 1.25rem 1.5rem;
		flex: 1;
		display: flex;
		flex-direction: column;
		background: #f8fafc;
	}

	.nf-section-header {
		margin-bottom: 1rem;
		flex-shrink: 0;
	}

	/* Simulated Issues layout */
	.nf-issues-list {
		display: flex;
		flex-direction: column;
		gap: 0.625rem;
		overflow-y: auto;
		max-height: 280px;
	}

	.nf-interactive-issue-row {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 0.875rem 1rem;
		text-align: left;
		cursor: pointer;
		display: flex;
		flex-direction: column;
		width: 100%;
		transition: all 0.2s ease;
		outline: none;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02);
	}

	.nf-interactive-issue-row:hover {
		border-color: #0d5c63;
		box-shadow: 0 4px 12px rgba(13, 92, 99, 0.05);
	}

	.nf-issue-expanded {
		border-color: #0d5c63;
		background: #ffffff;
		box-shadow: 0 6px 16px rgba(13, 92, 99, 0.08);
	}

	.nf-issue-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 0.375rem;
	}

	.nf-issue-badge {
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.06em;
		padding: 2px 7px;
		border-radius: 9999px;
		text-transform: uppercase;
	}

	.nf-badge-crit {
		background: #fef2f2;
		color: #dc2626;
		border: 1px solid #fee2e2;
	}

	.nf-badge-ser {
		background: #fff7ed;
		color: #ea580c;
		border: 1px solid #ffedd5;
	}

	.nf-badge-mod {
		background: #fffbeb;
		color: #d97706;
		border: 1px solid #fef3c7;
	}

	.nf-issue-selector-pill {
		font-family: ui-monospace, monospace;
		font-size: 10px;
		color: #64748b;
		background: #f1f5f9;
		padding: 1px 6px;
		border-radius: 4px;
	}

	.nf-interactive-issue-title {
		font-size: 12px;
		font-weight: 600;
		color: #1e293b;
		margin: 0;
	}

	/* Issue Code Drawer */
	.nf-issue-drawer {
		margin-top: 0.75rem;
		border-top: 1px solid #f1f5f9;
		padding-top: 0.75rem;
	}

	.nf-drawer-heading {
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: #64748b;
		margin: 0 0 0.25rem;
	}

	.nf-issue-code-box {
		background: #0f172a;
		border-radius: 6px;
		padding: 0.5rem 0.75rem;
		overflow-x: auto;
	}

	.nf-issue-code-box code {
		color: #e2e8f0;
		font-family: ui-monospace, monospace;
		font-size: 10px;
		white-space: pre-wrap;
		word-break: break-all;
	}

	.nf-drawer-body {
		font-size: 11px;
		color: #475569;
		line-height: 1.5;
		margin: 0;
	}

	/* Simulated Pages layout */
	.nf-pages-grid {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: 0.75rem;
		overflow-y: auto;
		max-height: 280px;
	}

	.nf-page-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 0.875rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02);
	}

	.nf-page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.5rem;
	}

	.nf-page-path {
		font-size: 11px;
		color: #0d5c63;
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.nf-page-badge {
		font-size: 8px;
		font-weight: 800;
		letter-spacing: 0.06em;
		padding: 1.5px 5px;
		border-radius: 4px;
		background: #fee2e2;
		color: #dc2626;
	}

	.nf-page-badge-success {
		background: #d1fae5;
		color: #065f46;
	}

	.nf-page-label {
		font-size: 12px;
		font-weight: 600;
		color: #1e293b;
		margin: 0;
	}

	.nf-page-metrics {
		display: flex;
		gap: 1.5rem;
		margin-top: 0.25rem;
		border-top: 1px solid #f1f5f9;
		padding-top: 0.375rem;
	}

	.nf-page-metric {
		display: flex;
		flex-direction: column;
	}

	.nf-metric-label {
		font-size: 9px;
		text-transform: uppercase;
		color: #94a3b8;
		font-weight: 600;
	}

	.nf-metric-value {
		font-size: 11px;
		font-weight: 700;
		font-family: ui-monospace, monospace;
	}

	/* Simulated Scanners layout */
	.nf-scanners-grid {
		display: grid;
		grid-template-columns: repeat(4, 1fr);
		gap: 0.625rem;
		overflow-y: auto;
		max-height: 280px;
	}

	.nf-scanner-card {
		background: #ffffff;
		border: 1px solid #e2e8f0;
		border-radius: 0.75rem;
		padding: 0.75rem;
		display: flex;
		flex-direction: column;
		justify-content: space-between;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.02);
		min-height: 84px;
	}

	.nf-scanner-card-header {
		display: flex;
		align-items: center;
		gap: 0.375rem;
	}

	.nf-scanner-circle-icon {
		width: 20px;
		height: 20px;
		border-radius: 5px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.nf-scanner-circle-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
	}

	.nf-scanner-card-name {
		font-size: 11px;
		font-weight: 600;
		color: #1e293b;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.nf-scanner-card-score {
		margin-left: auto;
		font-size: 11px;
		font-weight: 700;
		color: #0f172a;
		font-family: ui-monospace, monospace;
	}

	.nf-scanner-card-status {
		font-size: 9px;
		color: #64748b;
		margin-top: 0.5rem;
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.nf-status-wave {
		width: 5px;
		height: 5px;
		background: #10b981;
		border-radius: 50%;
		display: inline-block;
		box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
	}

	/* Main bento: 7fr left + 5fr right */
	.nf-bento-main {
		display: grid;
		grid-template-columns: 7fr 5fr;
		border-bottom: 1px solid #e2e8f0;
		min-height: 0;
	}

	.nf-panel-left {
		padding: 1.25rem 1.5rem;
		border-right: 1px solid #e2e8f0;
	}

	.nf-panel-right {
		padding: 1.25rem 1.25rem;
	}

	.nf-panel-toprow {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.nf-panel-title {
		font-size: 13px;
		font-weight: 600;
		color: #0f172a;
	}

	.nf-pill {
		background: #f1f5f9;
		color: #64748b;
		font-size: 11px;
		font-weight: 500;
		padding: 3px 10px;
		border-radius: 9999px;
	}

	.nf-see-all {
		font-size: 11px;
		font-weight: 500;
		color: #0d5c63;
		text-decoration: none;
	}

	.nf-see-all-btn {
		font-size: 11px;
		font-weight: 500;
		color: #0d5c63;
		background: transparent;
		border: none;
		cursor: pointer;
		outline: none;
		padding: 0;
	}

	.nf-see-all-btn:hover {
		text-decoration: underline;
	}

	/* Score area: grade+number on left, bars on right */
	.nf-score-area {
		display: grid;
		grid-template-columns: 1fr 1.1fr;
		gap: 1.25rem;
		align-items: start;
	}

	.nf-score-left {
		display: flex;
		flex-direction: column;
	}

	.nf-status-pill {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 3px 10px;
		font-size: 10px;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		background: #d1fae5;
		color: #065f46;
		border: 1px solid #a7f3d0;
		border-radius: 9999px;
		margin-bottom: 0.5rem;
		width: max-content;
	}

	.nf-status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: #10b981;
	}

	.nf-score-num {
		font-size: 52px;
		font-weight: 700;
		color: #0f172a;
		line-height: 1;
		letter-spacing: -0.04em;
		font-family: 'JetBrains Mono Variable', ui-monospace, SFMono-Regular, monospace;
		font-variant-numeric: tabular-nums;
	}

	.nf-score-denom {
		font-size: 22px;
		font-weight: 500;
		color: #94a3b8;
		letter-spacing: 0;
		font-family: 'JetBrains Mono Variable', ui-monospace, SFMono-Regular, monospace;
	}

	.nf-sev-bar {
		display: flex;
		height: 6px;
		width: 100%;
		max-width: 220px;
		margin-top: 0.6rem;
		background: #f1f5f9;
		border-radius: 9999px;
		overflow: hidden;
	}

	.nf-sev-seg {
		height: 100%;
	}

	.nf-score-desc {
		font-size: 11px;
		color: #64748b;
		margin-top: 0.5rem;
		line-height: 1.4;
	}

	/* Category score bars */
	.nf-score-chart {
		display: flex;
		flex-direction: column;
		gap: 0.6rem;
		padding-top: 0.25rem;
	}

	.nf-cat-row {
		display: grid;
		grid-template-columns: 88px 1fr 28px;
		align-items: center;
		gap: 0.5rem;
	}

	.nf-cat-label {
		font-size: 11px;
		color: #475569;
		font-weight: 500;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.nf-cat-track {
		height: 6px;
		background: #f1f5f9;
		border-radius: 9999px;
		overflow: hidden;
	}

	.nf-cat-fill {
		height: 100%;
		border-radius: 9999px;
	}

	.nf-cat-num {
		font-size: 11px;
		font-weight: 700;
		color: #0f172a;
		text-align: right;
		font-variant-numeric: tabular-nums;
	}

	/* Scanner rows */
	.nf-scanner-list {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.nf-scanner-item {
		display: grid;
		grid-template-columns: 28px 1fr auto 32px;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0.625rem;
		border-radius: 0.625rem;
		background: #f8fafc;
	}

	.nf-scanner-icon {
		width: 28px;
		height: 28px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.nf-scanner-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
	}

	.nf-scanner-name {
		font-size: 12px;
		font-weight: 500;
		color: #1e293b;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.nf-badge {
		font-size: 9px;
		font-weight: 700;
		letter-spacing: 0.06em;
		padding: 2px 7px;
		border-radius: 9999px;
		white-space: nowrap;
	}

	.nf-badge-done {
		background: #d1fae5;
		color: #065f46;
	}

	.nf-scanner-score {
		font-size: 12px;
		font-weight: 700;
		color: #0f172a;
		text-align: right;
		font-variant-numeric: tabular-nums;
		font-family: 'JetBrains Mono Variable', ui-monospace, SFMono-Regular, monospace;
	}

	/* Bottom bento row */
	.nf-bento-bottom {
		display: grid;
		grid-template-columns: 1fr 1fr 1fr;
		border-top: 1px solid #e2e8f0;
	}

	.nf-bottom-panel {
		padding: 1.125rem 1.25rem;
		border-right: 1px solid #e2e8f0;
	}

	.nf-bottom-panel:last-child {
		border-right: none;
	}

	/* Findings list */
	.nf-findings {
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.nf-finding-row {
		display: grid;
		grid-template-columns: 26px 1fr auto 14px;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 0;
		border-bottom: 1px solid #f1f5f9;
	}

	.nf-finding-row:last-child {
		border-bottom: none;
	}

	.nf-finding-avatar {
		width: 26px;
		height: 26px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.nf-finding-label {
		font-size: 12px;
		font-weight: 500;
		color: #1e293b;
	}

	.nf-finding-count {
		font-size: 11px;
		font-weight: 600;
		white-space: nowrap;
	}

	.nf-finding-caret {
		font-size: 12px;
		color: #cbd5e1;
	}

	/* CTA card — teal tinted, like Nexus lavender */
	.nf-cta-card {
		background: linear-gradient(135deg, #e6f4f4 0%, #d1f0ef 100%);
		border-right-color: #e2e8f0;
	}

	.nf-cta-title {
		font-size: 18px;
		font-weight: 800;
		color: #0d5c63;
		margin-bottom: 0.375rem;
		letter-spacing: -0.02em;
	}

	.nf-cta-desc {
		font-size: 11px;
		color: rgba(13, 92, 99, 0.75);
		line-height: 1.5;
		margin-bottom: 0.875rem;
	}

	.nf-cta-btn {
		display: inline-flex;
		align-items: center;
		background: white;
		color: #0d5c63;
		font-size: 11px;
		font-weight: 700;
		padding: 6px 14px;
		border-radius: 9999px;
		text-decoration: none;
		box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
	}

	/* Score trend */
	.nf-trend-stats {
		display: flex;
		align-items: center;
		gap: 0.875rem;
		margin-bottom: 0.75rem;
	}

	.nf-trend-stat {
		display: flex;
		flex-direction: column;
	}

	.nf-trend-num {
		font-size: 20px;
		font-weight: 700;
		color: #0f172a;
		line-height: 1;
		letter-spacing: -0.02em;
		font-variant-numeric: tabular-nums;
		font-family: 'JetBrains Mono Variable', ui-monospace, SFMono-Regular, monospace;
	}

	.nf-trend-lbl {
		font-size: 10px;
		color: #64748b;
		margin-top: 2px;
	}

	.nf-stat-div {
		width: 1px;
		height: 28px;
		background: #e2e8f0;
		flex-shrink: 0;
	}

	/* Mini bar chart */
	.nf-barchart {
		display: flex;
		align-items: flex-end;
		gap: 3px;
		height: 36px;
	}

	.nf-bar {
		flex: 1;
		background: #cbd5e1;
		border-radius: 2px 2px 0 0;
		min-height: 4px;
	}

	.nf-bar-current {
		background: #0d5c63;
	}

	/* Responsive — simplified mobile dashboard */
	@media (max-width: 768px) {
		.nf-frame {
			border-radius: 1rem;
			padding: 0.625rem 0.625rem 0;
		}

		.nf-dot {
			width: 8px;
			height: 8px;
		}

		.nf-urlbar {
			font-size: 9px;
			padding: 3px 8px;
		}

		.nf-bento-main {
			grid-template-columns: 1fr;
		}

		.nf-panel-left {
			padding: 1rem;
			border-right: none;
		}

		.nf-panel-right {
			display: none;
		}

		.nf-score-area {
			grid-template-columns: 1fr;
			gap: 1rem;
		}

		.nf-score-num {
			font-size: 40px;
		}

		.nf-score-denom {
			font-size: 18px;
		}

		.nf-nav {
			padding: 0.5rem 0.75rem;
			gap: 0.75rem;
		}

		.nf-logo-icon {
			width: 22px;
			height: 22px;
			font-size: 8px;
		}

		.nf-logo-text {
			font-size: 11px;
		}

		.nf-tab {
			font-size: 10px;
			padding: 0.25rem 0.5rem;
		}

		.nf-bento-bottom {
			grid-template-columns: 1fr;
		}

		.nf-bottom-panel {
			padding: 0.875rem 1rem;
			border-right: none;
			border-bottom: 1px solid #e2e8f0;
		}

		.nf-bottom-panel:last-child {
			border-bottom: none;
		}

		.nf-cta-title {
			font-size: 15px;
		}

		.nf-trend-num {
			font-size: 16px;
		}

		.nf-scanners-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.nf-pages-grid {
			grid-template-columns: 1fr;
		}
	}

	/* Very small screens — hide bottom row entirely */
	@media (max-width: 480px) {
		.nf-bento-bottom {
			display: none;
		}

		.nf-tabs {
			display: none;
		}

		.nf-scanners-grid {
			grid-template-columns: 1fr;
		}
	}
</style>

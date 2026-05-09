<script lang="ts">
	const navTabs = ['Overview', 'Scanners', 'Issues', 'Visual'];

	const scanners = [
		{ name: 'Axe', score: 100, status: 'done', color: '#3b82f6', bg: '#eff6ff' },
		{ name: 'Lighthouse', score: 89, status: 'done', color: '#f59e0b', bg: '#fffbeb' },
		{ name: 'Security Headers', score: 94, status: 'done', color: '#8b5cf6', bg: '#f5f3ff' },
		{ name: 'SEO', score: 82, status: 'live', color: '#f43f5e', bg: '#fff1f2' },
		{ name: 'Link Checker', score: 71, status: 'live', color: '#10b981', bg: '#f0fdf4' },
		{ name: 'Open Graph', score: 65, status: 'queued', color: '#06b6d4', bg: '#ecfeff' }
	] as const;

	const categoryScores = [
		{ label: 'Accessibility', score: 100, color: '#3b82f6' },
		{ label: 'Performance', score: 89, color: '#f59e0b' },
		{ label: 'SEO', score: 82, color: '#f43f5e' },
		{ label: 'Security', score: 94, color: '#8b5cf6' }
	] as const;

	const findings = [
		{ label: 'Critical', count: 3, color: '#dc2626', bg: '#fef2f2' },
		{ label: 'Serious', count: 8, color: '#ea580c', bg: '#fff7ed' },
		{ label: 'Moderate', count: 14, color: '#d97706', bg: '#fffbeb' }
	] as const;

	const trendBars = [38, 44, 52, 48, 61, 68, 72, 70, 79, 84, 88, 94] as const;
</script>

<!-- Nexus-style: slate frame → white bento card inside -->
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
				{#each navTabs as tab, i (tab)}
					<span class={i === 0 ? 'nf-tab nf-tab-active' : 'nf-tab'}>{tab}</span>
				{/each}
			</div>
		</div>

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
						<div class="nf-grade">A</div>
						<div class="nf-score-num">94<span class="nf-score-denom">/100</span></div>
						<p class="nf-score-desc">staging.example.com · 3 pages · 2m 18s</p>
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
					<a href="/playground" class="nf-see-all">Run scan →</a>
				</div>
				<div class="nf-scanner-list">
					{#each scanners as s (s.name)}
						<div class="nf-scanner-item">
							<div class="nf-scanner-icon" style="background:{s.bg}">
								<div class="nf-scanner-dot" style="background:{s.color}"></div>
							</div>
							<span class="nf-scanner-name">{s.name}</span>
							{#if s.status === 'done'}
								<span class="nf-badge nf-badge-done">DONE</span>
							{:else if s.status === 'live'}
								<span class="nf-badge nf-badge-live">LIVE</span>
							{:else}
								<span class="nf-badge nf-badge-queued">QUEUED</span>
							{/if}
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
					<span class="nf-see-all">View report →</span>
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
				<p class="nf-cta-title">8 Scanners</p>
				<p class="nf-cta-desc">
					No account required. Run accessibility, performance, SEO, and security in one flow.
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
						<span class="nf-trend-num">24</span>
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
	}

	/* Product nav */
	.nf-nav {
		display: flex;
		align-items: center;
		gap: 1.5rem;
		padding: 0.75rem 1.25rem;
		border-bottom: 1px solid #e2e8f0;
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
		font-weight: 500;
		color: #64748b;
		cursor: default;
		white-space: nowrap;
	}

	.nf-tab-active {
		background: #0d5c63;
		color: white;
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

	.nf-grade {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		background: #d1fae5;
		color: #065f46;
		font-size: 15px;
		font-weight: 800;
		border-radius: 50%;
		margin-bottom: 0.5rem;
	}

	.nf-score-num {
		font-size: 52px;
		font-weight: 800;
		color: #0f172a;
		line-height: 1;
		letter-spacing: -0.04em;
	}

	.nf-score-denom {
		font-size: 22px;
		font-weight: 500;
		color: #94a3b8;
		letter-spacing: 0;
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

	.nf-badge-live {
		background: #dbeafe;
		color: #1d4ed8;
	}

	.nf-badge-queued {
		background: #f1f5f9;
		color: #64748b;
	}

	.nf-scanner-score {
		font-size: 12px;
		font-weight: 700;
		color: #0f172a;
		text-align: right;
		font-variant-numeric: tabular-nums;
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
		font-weight: 800;
		color: #0f172a;
		line-height: 1;
		letter-spacing: -0.02em;
		font-variant-numeric: tabular-nums;
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

	/* Responsive — hide complex layout on small screens */
	@media (max-width: 768px) {
		.nf-bento-main {
			grid-template-columns: 1fr;
		}
		.nf-panel-right {
			display: none;
		}
		.nf-bento-bottom {
			grid-template-columns: 1fr;
		}
		.nf-bottom-panel:not(:first-child) {
			display: none;
		}
	}
</style>

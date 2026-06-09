<script lang="ts">
	import { fade } from 'svelte/transition';

	import {
		categoryScores,
		findings,
		scanners,
		severitySegments,
		trendBars
	} from './NexusDashboardHeroData';

	const { onViewIssues }: { onViewIssues: () => void } = $props();
</script>

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
				<button type="button" class="nf-see-all-btn" onclick={onViewIssues}>View details →</button>
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

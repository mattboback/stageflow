<script lang="ts">
	import { fade } from 'svelte/transition';

	import { mockIssues } from './NexusDashboardHeroData';

	// Local to this view: collapses automatically when the tab is switched away,
	// because the parent {#if} destroys and recreates this component.
	let expandedIssueIdx = $state<number | null>(null);
</script>

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
				onclick={() => (expandedIssueIdx = expandedIssueIdx === idx ? null : idx)}
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

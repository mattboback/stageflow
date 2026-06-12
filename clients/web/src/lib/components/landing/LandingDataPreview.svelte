<script lang="ts">
	import { SeverityBar } from '$lib/components/ui';

	// Static, realistic sample output. This is a picture of the product's
	// actual deliverable — a merged report — not a mocked dashboard.
	const rows = [
		{ scanner: 'Axe', category: 'Accessibility', result: '12 issues', state: 'done' },
		{ scanner: 'Lighthouse', category: 'Performance', result: '94 / 100', state: 'done' },
		{ scanner: 'Security Headers', category: 'Security', result: '2 missing', state: 'done' },
		{ scanner: 'SEO', category: 'SEO', result: '3 issues', state: 'done' },
		{ scanner: 'Open Graph', category: 'SEO', result: '1 issue', state: 'done' },
		{ scanner: 'Link Checker', category: 'Quality', result: '4 broken links', state: 'done' },
		{ scanner: 'Spelling & Grammar', category: 'Quality', result: '7 issues', state: 'done' },
		{ scanner: 'AI Navigator', category: 'AI', result: 'goal reached', state: 'done' }
	] as const;

	const severityCounts = { critical: 3, serious: 9, moderate: 11, minor: 4 } as const;
</script>

<figure class="border-line bg-surface rounded-md border">
	<figcaption
		class="border-line text-ink-faint flex flex-wrap items-center gap-x-3 gap-y-1 border-b px-4 py-2.5 font-mono text-[11px] sm:px-5"
	>
		<span class="text-ink font-semibold">unified report</span>
		<span aria-hidden="true">·</span>
		<span>example-storefront.com</span>
		<span aria-hidden="true">·</span>
		<span>6 pages</span>
		<span class="text-accent-ink bg-accent-soft ml-auto rounded-sm px-1.5 py-0.5 font-semibold"
			>COMPLETE</span
		>
	</figcaption>

	<table class="w-full text-sm">
		<caption class="sr-only">Sample scan results from all eight scanners</caption>
		<thead>
			<tr class="border-line text-ink-faint border-b text-left font-mono text-[11px] uppercase">
				<th scope="col" class="px-4 py-2 font-medium sm:px-5">Scanner</th>
				<th scope="col" class="hidden px-4 py-2 font-medium sm:table-cell">Category</th>
				<th scope="col" class="px-4 py-2 text-right font-medium sm:px-5">Findings</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.scanner)}
				<tr class="border-line/70 border-b last:border-b-0">
					<td class="text-ink-strong px-4 py-2.5 font-medium sm:px-5">{row.scanner}</td>
					<td class="text-ink-muted hidden px-4 py-2.5 text-xs sm:table-cell">{row.category}</td>
					<td class="text-ink px-4 py-2.5 text-right font-mono text-xs tabular-nums sm:px-5">
						{row.result}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>

	<div class="border-line border-t px-4 py-3 sm:px-5">
		<p class="text-ink-faint mb-2 font-mono text-[11px] uppercase">27 findings by severity</p>
		<SeverityBar counts={severityCounts} showLabels height="sm" />
	</div>
</figure>

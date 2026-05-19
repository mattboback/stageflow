<script lang="ts">
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { cn } from '$lib/utils';

	interface Section {
		id: string;
		label: string;
		shortLabel?: string;
		count?: number;
	}

	interface Props {
		section: string;
		report: UnifiedReport;
		onSectionChange: (sectionId: string) => void;
	}

	let { section, report, onSectionChange }: Props = $props();

	const sections = $derived<Section[]>([
		{ id: 'overview', label: 'Overview', shortLabel: 'Overview' },
		{
			id: 'issues',
			label: 'Issues',
			shortLabel: 'Issues',
			count: report.issues.length
		},
		{
			id: 'pages',
			label: 'Pages',
			shortLabel: 'Pages',
			count: report.pages.length
		},
		{
			id: 'scanners',
			label: 'Scanners',
			shortLabel: 'Scans',
			count: report.scanners.length
		},
		{ id: 'visual', label: 'Visual Inspector', shortLabel: 'Visual' },
		{
			id: 'artifacts',
			label: 'Diagnostics',
			shortLabel: 'Diagnostics',
			count: (report.artifacts?.length ?? 0) + (report.errors?.length ?? 0)
		}
	]);
</script>

<Panel
	variant="muted"
	padding="xs"
	rounded="2xl"
	class="border-line bg-surface sticky top-3 z-20 mb-6 border shadow-sm"
>
	<div class="relative -mx-1 overflow-x-auto px-1 py-1">
		<div
			class="flex min-w-max items-center gap-1 sm:gap-1.5"
			role="toolbar"
			aria-label="Report sections"
		>
			{#each sections as item (item.id)}
				<button
					onclick={() => onSectionChange(item.id)}
					class={cn(
						'flex items-center gap-1.5 rounded-xl px-3 py-2 text-[13px] font-semibold whitespace-nowrap transition sm:px-3.5 sm:text-sm',
						section === item.id
							? 'bg-accent text-surface shadow-sm'
							: 'text-ink-muted hover:bg-surface-muted hover:text-ink'
					)}
					aria-pressed={section === item.id}
					id={`report-tab-${item.id}`}
				>
					<span>{item.shortLabel ?? item.label}</span>
					{#if typeof item.count === 'number'}
						<span
							class={cn(
								'hidden rounded-full px-1.5 py-0.5 font-mono text-[11px] font-semibold tabular-nums sm:inline-flex',
								section === item.id ? 'text-surface bg-white/20' : 'bg-surface-muted text-ink-faint'
							)}
							aria-label={`${item.count} ${item.label.toLowerCase()}`}
						>
							{item.count}
						</span>
					{/if}
				</button>
			{/each}
		</div>
		<div
			class="from-surface/90 pointer-events-none absolute inset-y-0 left-0 w-4 bg-gradient-to-r to-transparent"
		></div>
		<div
			class="from-surface/90 pointer-events-none absolute inset-y-0 right-0 w-4 bg-gradient-to-l to-transparent"
		></div>
	</div>
</Panel>

<script lang="ts">
	import type { ReportAudience } from '$lib/types/report-audience';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { cn } from '$lib/utils';

	interface Section {
		id: string;
		label: string;
		count?: number;
	}

	interface Props {
		section: string;
		report: UnifiedReport;
		audience?: ReportAudience;
		onSectionChange: (sectionId: string) => void;
		onAudienceChange?: (audience: ReportAudience) => void;
	}

	const { section, report, audience = 'pm', onSectionChange, onAudienceChange }: Props = $props();

	const sections = $derived<Section[]>([
		{ id: 'overview', label: 'Overview' },
		{ id: 'issues', label: 'Issues', count: report.issues.length },
		{ id: 'pages', label: 'Pages', count: report.pages.length },
		{ id: 'scanners', label: 'Scanners', count: report.scanners.length },
		{ id: 'artifacts', label: 'Artifacts' },
		{ id: 'errors', label: 'Errors', count: report.errors?.length ?? 0 }
	]);

	const audienceOptions: { id: ReportAudience; label: string; hint: string }[] = [
		{ id: 'pm', label: 'PM', hint: 'Screenshots + impact' },
		{ id: 'engineer', label: 'Engineer', hint: 'Rules + code details' },
		{ id: 'designer', label: 'Designer', hint: 'Visual context' }
	];
</script>

<Panel
	variant="muted"
	padding="xs"
	rounded="2xl"
	class="mb-8 flex flex-wrap items-center justify-between gap-2"
>
	<div class="flex flex-wrap gap-2" role="tablist" aria-label="Report sections">
		{#each sections as item (item.id)}
			<button
				onclick={() => onSectionChange(item.id)}
				class={cn(
					'flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-semibold transition',
					section === item.id
						? 'bg-ink text-surface'
						: 'text-ink-muted hover:bg-surface hover:text-ink'
				)}
				role="tab"
				aria-selected={section === item.id}
				aria-controls={`report-panel-${item.id}`}
				id={`report-tab-${item.id}`}
			>
				<span>{item.label}</span>
				{#if typeof item.count === 'number'}
					<span
						class={cn(
							'rounded-full px-2 py-0.5 text-xs font-semibold',
							section === item.id ? 'bg-surface/20 text-surface' : 'bg-surface text-ink-muted'
						)}
						aria-label={`${item.count} ${item.label.toLowerCase()}`}
					>
						{item.count}
					</span>
				{/if}
			</button>
		{/each}
	</div>
	{#if onAudienceChange}
		<div class="flex items-center gap-2">
			<span class="text-ink-faint text-xs font-semibold tracking-wide uppercase">View</span>
			{#each audienceOptions as opt (opt.id)}
				<button
					type="button"
					onclick={() => onAudienceChange(opt.id)}
					class={cn(
						'border-line text-ink-muted hover:bg-surface rounded-lg border px-2.5 py-1 text-xs font-semibold transition',
						audience === opt.id && 'border-accent bg-accent/10 text-accent'
					)}
					aria-pressed={audience === opt.id}
					title={opt.hint}
				>
					{opt.label}
				</button>
			{/each}
		</div>
	{/if}
</Panel>

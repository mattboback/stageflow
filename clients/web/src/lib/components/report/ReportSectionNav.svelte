<script lang="ts">
	import type { ReportAudience } from '$lib/types/report-audience';
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
		audience?: ReportAudience;
		onSectionChange: (sectionId: string) => void;
		onAudienceChange?: (audience: ReportAudience) => void;
	}

	let { section, report, audience = 'pm', onSectionChange, onAudienceChange }: Props = $props();

	const sections = $derived<Section[]>(
		[
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
			{ id: 'visual', label: 'Visual Review', shortLabel: 'Visual' },
			{ id: 'artifacts', label: 'Artifacts', shortLabel: 'Files' },
			{
				id: 'errors',
				label: 'Errors',
				shortLabel: 'Errors',
				count: report.errors?.length ?? 0
			}
		].filter((item) => {
			if (item.id !== 'errors') return true;
			if (section === 'errors') return true;
			return (item.count ?? 0) > 0;
		})
	);

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
	class="border-line bg-surface sticky top-3 z-20 mb-6 space-y-2 border shadow-sm"
>
	<div class="relative -mx-1 overflow-x-auto px-1 pt-1">
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
								'hidden rounded-full px-1.5 py-0.5 text-[11px] font-semibold sm:inline-flex',
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
	{#if onAudienceChange}
		<div class="border-line flex flex-wrap items-center justify-between gap-2 border-t px-2 pt-2">
			<div>
				<span class="text-ink-faint text-xs font-semibold tracking-wide uppercase">Audience</span>
				<p class="text-ink-faint text-[11px]">
					Adjust issue detail depth without changing report data.
				</p>
			</div>
			<div class="flex items-center gap-1.5">
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
		</div>
	{/if}
</Panel>

<script lang="ts">
	import { cn } from '$lib/utils';

	export type StatusPillTone =
		| 'strong'
		| 'watch'
		| 'needs-work'
		| 'high-risk'
		| 'failing'
		| 'neutral';

	interface Props {
		tone: StatusPillTone;
		label?: string;
		size?: 'sm' | 'md';
		class?: string;
	}

	let { tone, label, size = 'md', class: className }: Props = $props();

	const defaultLabels: Record<StatusPillTone, string> = {
		strong: 'Strong',
		watch: 'Watch',
		'needs-work': 'Needs work',
		'high-risk': 'High risk',
		failing: 'Failing',
		neutral: 'Unknown'
	};

	const toneClasses: Record<StatusPillTone, string> = {
		strong: 'bg-emerald-500/5 text-emerald-800 border-emerald-500/20',
		watch: 'bg-blue-500/5 text-blue-800 border-blue-500/20',
		'needs-work': 'bg-amber-500/5 text-amber-800 border-amber-500/20',
		'high-risk': 'bg-orange-500/5 text-orange-800 border-orange-500/20',
		failing: 'bg-red-500/5 text-red-800 border-red-500/20',
		neutral: 'bg-ink/5 text-ink border-ink/20'
	};
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-sm border font-semibold tracking-[0.08em] uppercase',
		size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-1 text-[11px]',
		toneClasses[tone],
		className
	)}
	data-tone={tone}
	role="status"
>
	<span
		aria-hidden="true"
		class={cn(
			'h-1.5 w-1.5 rounded-full',
			tone === 'strong' && 'bg-emerald-500',
			tone === 'watch' && 'bg-blue-500',
			tone === 'needs-work' && 'bg-amber-500',
			tone === 'high-risk' && 'bg-orange-500',
			tone === 'failing' && 'bg-red-500',
			tone === 'neutral' && 'bg-ink-faint'
		)}
	></span>
	{label ?? defaultLabels[tone]}
</span>

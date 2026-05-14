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
		strong: 'bg-emerald-50 text-emerald-700 border-emerald-200',
		watch: 'bg-blue-50 text-blue-700 border-blue-200',
		'needs-work': 'bg-amber-50 text-amber-700 border-amber-200',
		'high-risk': 'bg-orange-50 text-orange-700 border-orange-200',
		failing: 'bg-red-50 text-red-700 border-red-200',
		neutral: 'bg-slate-50 text-slate-700 border-slate-200'
	};
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full border font-semibold tracking-[0.08em] uppercase',
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
			tone === 'neutral' && 'bg-slate-400'
		)}
	></span>
	{label ?? defaultLabels[tone]}
</span>

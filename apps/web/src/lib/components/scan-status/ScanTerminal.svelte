<script lang="ts">
	import { cn } from '$lib/utils';
	import { Loader2, Terminal } from 'lucide-svelte';
	import { onMount } from 'svelte';

	interface Props {
		logs: string[];
		startTime?: number;
	}

	interface LogStyle {
		textClass: string;
		prefix?: string;
		prefixClass?: string;
	}

	const LOG_PATTERNS: { test: (_log: string) => boolean; style: LogStyle }[] = [
		{
			test: (log) => /error|critical|fail/i.test(log),
			style: {
				textClass: 'text-red-400',
				prefix: 'ERROR',
				prefixClass: 'bg-red-500/20 text-red-400'
			}
		},
		{
			test: (log) => /complete|done|success/i.test(log),
			style: {
				textClass: 'text-green-400',
				prefix: 'OK',
				prefixClass: 'bg-green-500/20 text-green-400'
			}
		},
		{
			test: (log) => /^\[[^\]]+\]/i.test(log),
			style: {
				textClass: 'text-accent',
				prefix: 'SCAN',
				prefixClass: 'bg-accent/20 text-accent'
			}
		},
		{
			test: (log) => /\[axe-core\]|axe/i.test(log),
			style: {
				textClass: 'text-blue-400',
				prefix: 'AXE',
				prefixClass: 'bg-blue-500/20 text-blue-400'
			}
		},
		{
			test: (log) => /extract|verify/i.test(log),
			style: {
				textClass: 'text-amber-400',
				prefix: 'PREP',
				prefixClass: 'bg-amber-500/20 text-amber-400'
			}
		},
		{
			test: (log) => /upload|storage/i.test(log),
			style: {
				textClass: 'text-orange-400',
				prefix: 'I/O',
				prefixClass: 'bg-orange-500/20 text-orange-400'
			}
		},
		{
			test: (log) => /queue|waiting|allocat/i.test(log),
			style: {
				textClass: 'text-slate-400',
				prefix: 'SYS',
				prefixClass: 'bg-slate-500/20 text-slate-400'
			}
		}
	];

	const DEFAULT_STYLE: LogStyle = { textClass: 'text-slate-300' };

	function getLogStyle(log: string): LogStyle {
		return LOG_PATTERNS.find((p) => p.test(log))?.style ?? DEFAULT_STYLE;
	}

	function formatTimestamp(startTime: number, index: number, now: number): string {
		const elapsedSeconds = Math.floor((now - startTime) / 1000) - (10 - index);
		const seconds = Math.max(0, elapsedSeconds);
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
	}

	const { logs, startTime }: Props = $props();

	let containerRef = $state<HTMLDivElement | null>(null);
	// Capture the time when the component was first created as a fallback
	const fallbackStartTime = Date.now();
	const effectiveStartTime = $derived(startTime ?? fallbackStartTime);
	let now = $state(Date.now());

	onMount(() => {
		const interval = setInterval(() => {
			now = Date.now();
		}, 1000);
		return () => clearInterval(interval);
	});

	$effect(() => {
		// Access logs to track dependency
		if (logs.length === 0 || !containerRef) return;

		// Only auto-scroll if user is near bottom (within 100px)
		const isNearBottom =
			containerRef.scrollHeight - containerRef.scrollTop - containerRef.clientHeight < 100;
		if (isNearBottom) {
			containerRef.scrollTop = containerRef.scrollHeight;
		}
	});
</script>

<div
	class="mt-8 overflow-hidden rounded-[12px] border border-slate-800 bg-[#0D1117] shadow-2xl shadow-black/50"
>
	<div class="flex items-center justify-between border-b border-slate-800 bg-[#161b22] px-4 py-2">
		<div class="flex items-center gap-2 text-slate-400">
			<Terminal class="h-3.5 w-3.5" />
			<span class="font-mono text-xs">stageflow-orchestrator</span>
		</div>
		<div class="flex items-center gap-3">
			{#if logs.length > 0}
				<span class="font-mono text-xs text-slate-500">{logs.length} events</span>
			{/if}
			<div class="flex gap-1.5">
				<div class="h-2.5 w-2.5 rounded-full bg-[#FF5F56]"></div>
				<div class="h-2.5 w-2.5 rounded-full bg-[#FFBD2E]"></div>
				<div class="h-2.5 w-2.5 rounded-full bg-[#27C93F]"></div>
			</div>
		</div>
	</div>
	<div bind:this={containerRef} class="h-72 space-y-1 overflow-y-auto p-4 font-mono text-xs">
		{#if logs.length === 0}
			<div class="flex items-center gap-2 text-slate-500">
				<Loader2 class="h-3 w-3 animate-spin" />
				<span>Waiting for live scan events...</span>
			</div>
		{/if}
		{#each logs as log, idx (log + '-' + idx)}
			{@const style = getLogStyle(log)}
			<div class="flex items-start gap-2 py-0.5">
				<!-- Timestamp -->
				<span class="shrink-0 text-slate-600 tabular-nums select-none">
					[{formatTimestamp(effectiveStartTime, logs.length - idx, now)}]
				</span>

				<!-- Status prefix badge -->
				{#if style.prefix}
					<span
						class={cn(
							'shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium tracking-wider uppercase',
							style.prefixClass
						)}
					>
						{style.prefix}
					</span>
				{/if}

				<!-- Log message -->
				<span class={cn('break-all', style.textClass)}>{log}</span>
			</div>
		{/each}
	</div>
</div>

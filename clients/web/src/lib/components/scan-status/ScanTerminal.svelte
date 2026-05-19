<script lang="ts">
	import { cn } from '$lib/utils';
	import { Loader2, Terminal } from 'lucide-svelte';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';

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
				textClass: 'text-red-400 drop-shadow-[0_0_6px_rgba(248,113,113,0.35)]',
				prefix: 'ERROR',
				prefixClass: 'border-red-500/30 bg-red-500/10 text-red-400'
			}
		},
		{
			test: (log) => /complete|done|success/i.test(log),
			style: {
				textClass: 'text-emerald-400 drop-shadow-[0_0_6px_rgba(52,211,153,0.35)]',
				prefix: 'OK',
				prefixClass: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-400'
			}
		},
		{
			test: (log) => /^\[[^\]]+\]/i.test(log),
			style: {
				textClass: 'text-cyan-400 drop-shadow-[0_0_6px_rgba(34,211,238,0.35)]',
				prefix: 'SCAN',
				prefixClass: 'border-cyan-500/30 bg-cyan-500/10 text-cyan-400'
			}
		},
		{
			test: (log) => /\[axe-core\]|axe/i.test(log),
			style: {
				textClass: 'text-teal-300 drop-shadow-[0_0_6px_rgba(94,234,212,0.35)]',
				prefix: 'AXE',
				prefixClass: 'border-teal-500/30 bg-teal-500/10 text-teal-300'
			}
		},
		{
			test: (log) => /extract|verify/i.test(log),
			style: {
				textClass: 'text-amber-300 drop-shadow-[0_0_6px_rgba(252,211,77,0.35)]',
				prefix: 'PREP',
				prefixClass: 'border-amber-500/30 bg-amber-500/10 text-amber-300'
			}
		},
		{
			test: (log) => /upload|storage/i.test(log),
			style: {
				textClass: 'text-orange-300 drop-shadow-[0_0_6px_rgba(253,186,116,0.35)]',
				prefix: 'I/O',
				prefixClass: 'border-orange-500/30 bg-orange-500/10 text-orange-300'
			}
		},
		{
			test: (log) => /queue|waiting|allocat/i.test(log),
			style: {
				textClass: 'text-slate-400',
				prefix: 'SYS',
				prefixClass: 'border-slate-500/30 bg-slate-500/10 text-slate-400'
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

	let { logs, startTime }: Props = $props();

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
	class="terminal-glow-bg relative mt-8 overflow-hidden rounded-2xl border border-[#1e293b]/70 bg-[#080b11] shadow-[0_25px_50px_-12px_rgba(0,0,0,0.7)]"
>
	<!-- Glassy terminal header -->
	<div class="flex items-center justify-between border-b border-[#1e293b]/70 bg-[#0e131f]/90 px-5 py-3 backdrop-blur-md">
		<div class="flex items-center gap-2 text-[#64748b]">
			<Terminal class="text-accent h-4 w-4 drop-shadow-[0_0_6px_rgba(13,92,99,0.5)]" />
			<span class="font-mono text-xs font-semibold tracking-wide text-slate-300">stageflow-orchestrator</span>
		</div>
		<div class="flex items-center gap-3">
			{#if logs.length > 0}
				<span class="bg-[#1e293b]/40 rounded-full px-2.5 py-0.5 font-mono text-[10px] font-bold text-slate-400 border border-[#1e293b]/50">
					{logs.length} events
				</span>
			{/if}
			<div class="flex gap-1.5">
				<div class="h-2.5 w-2.5 rounded-full bg-[#FF5F56] opacity-90 shadow-[0_0_6px_rgba(255,95,86,0.5)]"></div>
				<div class="h-2.5 w-2.5 rounded-full bg-[#FFBD2E] opacity-90 shadow-[0_0_6px_rgba(255,189,46,0.5)]"></div>
				<div class="h-2.5 w-2.5 rounded-full bg-[#27C93F] opacity-90 shadow-[0_0_6px_rgba(39,201,63,0.5)]"></div>
			</div>
		</div>
	</div>

	<!-- Scrollable logs body with scanline overlay -->
	<div
		bind:this={containerRef}
		class="terminal-console-scroller h-80 space-y-1.5 overflow-y-auto p-5 font-mono text-xs text-slate-300"
	>
		{#if logs.length === 0}
			<div class="flex items-center gap-2.5 text-slate-500 py-2">
				<Loader2 class="text-accent h-3.5 w-3.5 animate-spin" />
				<span class="animate-pulse">Awaiting connection to remote scanner orchestration pipeline…</span>
			</div>
		{/if}

		{#each logs as log, idx (log + '-' + idx)}
			{@const style = getLogStyle(log)}
			<div class="terminal-line flex items-start gap-2.5 py-0.5" in:fade={{ duration: 150 }}>
				<!-- Timestamp -->
				<span class="shrink-0 text-slate-600 tabular-nums select-none tracking-tight">
					[{formatTimestamp(effectiveStartTime, logs.length - idx, now)}]
				</span>

				<!-- Status prefix badge -->
				{#if style.prefix}
					<span
						class={cn(
							'shrink-0 rounded border px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.08em] uppercase',
							style.prefixClass
						)}
					>
						{style.prefix}
					</span>
				{/if}

				<!-- Log message -->
				<span class={cn('break-all leading-relaxed', style.textClass)}>{log}</span>
			</div>
		{/each}

		<!-- Active Blinking Prompt at bottom when logs exist -->
		{#if logs.length > 0}
			<div class="flex items-center gap-2.5 py-1 text-slate-500" in:fade={{ duration: 120 }}>
				<span class="shrink-0 text-slate-600 select-none">[{formatTimestamp(effectiveStartTime, 0, now)}]</span>
				<span class="border-emerald-500/30 bg-emerald-500/10 text-emerald-400 shrink-0 rounded border px-1.5 py-0.5 text-[9px] font-semibold tracking-[0.08em] uppercase">
					AGNT
				</span>
				<span class="flex items-center gap-1.5 text-emerald-400 font-semibold tracking-wide">
					stageflow-agent: listening_active_diagnostics
					<span class="terminal-cursor inline-block h-3.5 w-2 bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]"></span>
				</span>
			</div>
		{/if}
	</div>
</div>

<style>
	/* Background with radial highlight */
	.terminal-glow-bg {
		background: radial-gradient(circle at 50% 30%, #0d1424 0%, #060910 100%) !important;
	}

	/* CRT scanline overlay */
	.terminal-glow-bg::before {
		content: " ";
		display: block;
		position: absolute;
		top: 0; left: 0; bottom: 0; right: 0;
		background: linear-gradient(rgba(255, 255, 255, 0) 50%, rgba(0, 0, 0, 0.18) 50%);
		background-size: 100% 4px;
		z-index: 10;
		pointer-events: none;
	}

	/* Styled custom scrollbar */
	.terminal-console-scroller::-webkit-scrollbar {
		width: 8px;
	}
	.terminal-console-scroller::-webkit-scrollbar-track {
		background: rgba(13, 20, 36, 0.5);
		border-radius: 4px;
	}
	.terminal-console-scroller::-webkit-scrollbar-thumb {
		background: rgba(100, 116, 139, 0.35);
		border-radius: 4px;
	}
	.terminal-console-scroller::-webkit-scrollbar-thumb:hover {
		background: rgba(100, 116, 139, 0.55);
	}

	/* Blinking block terminal cursor */
	@keyframes cursor-blink {
		0%, 100% { opacity: 1; }
		50% { opacity: 0; }
	}

	.terminal-cursor {
		animation: cursor-blink 1.1s step-end infinite;
	}

	/* Neon text glow animations for logs */
	.terminal-line {
		animation: line-fade-glow 0.25s ease-out forwards;
	}

	@keyframes line-fade-glow {
		0% {
			opacity: 0;
			transform: translateY(2px);
		}
		100% {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>

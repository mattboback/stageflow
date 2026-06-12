<script lang="ts">
	import { cn } from '$lib/utils';

	interface Props {
		/** Number of shimmer lines to render. */
		lines?: number;
		class?: string;
	}

	let { lines = 3, class: className }: Props = $props();
</script>

<div class={cn('space-y-2.5', className)} aria-hidden="true" data-testid="skeleton">
	{#each Array(lines) as _, i (i)}
		<div
			class="skeleton-line bg-surface-muted relative h-4 overflow-hidden rounded-sm"
			style={`width: ${Math.max(40, 95 - i * 14)}%`}
		></div>
	{/each}
</div>

<style>
	.skeleton-line::after {
		content: '';
		position: absolute;
		inset: 0;
		transform: translateX(-100%);
		background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.65), transparent);
		animation: skeleton-shimmer 1.5s infinite;
	}

	@keyframes skeleton-shimmer {
		100% {
			transform: translateX(100%);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.skeleton-line::after {
			animation: none;
		}
	}
</style>

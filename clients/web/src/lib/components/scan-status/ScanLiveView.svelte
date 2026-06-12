<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		headerContent: Snippet;
		mainContent: Snippet;
		sidebarContent: Snippet;
	}

	let { headerContent, mainContent, sidebarContent }: Props = $props();
</script>

<div aria-label="Scan status">
	<!-- Status strip: the single source of truth for job id · elapsed · state -->
	<header class="border-line bg-surface rounded-md border px-4 py-3 sm:px-5">
		{@render headerContent()}
	</header>

	<div class="mt-6 grid items-start gap-6 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
		<div class="min-w-0">
			{@render mainContent()}
		</div>
		<div class="min-w-0">
			{@render sidebarContent()}
		</div>
	</div>
</div>

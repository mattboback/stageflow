<script lang="ts">
	import { page } from '$app/state';
	import { Button, Container, Panel } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { AlertTriangle, Home, RotateCcw } from 'lucide-svelte';

	const status = $derived(page.status);
	const message = $derived(page.error?.message ?? 'An unexpected error occurred');

	const isNotFound = $derived(status === 404);
	const title = $derived(isNotFound ? 'Page Not Found' : 'Something Went Wrong');
	const description = $derived(
		isNotFound
			? "The page you're looking for doesn't exist or has been moved."
			: 'We encountered an error while processing your request.'
	);
</script>

<svelte:head>
	<title>{title} | {SITE.name}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="bg-paper flex min-h-screen items-center justify-center py-16">
	<Container class="max-w-lg text-center">
		<Panel padding="xl" rounded="xl" class="shadow-md">
			<div class="bg-accent/10 mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full">
				<AlertTriangle class="text-accent h-8 w-8" />
			</div>

			<p class="text-accent mb-2 font-mono text-sm font-semibold">{status}</p>
			<h1 class="text-ink mb-3 text-2xl font-bold">{title}</h1>
			<p class="text-ink-muted mb-6">{description}</p>

			{#if !isNotFound}
				<details class="bg-surface-muted mb-6 rounded-lg p-4 text-left">
					<summary class="text-ink-muted cursor-pointer text-sm font-medium">
						Technical Details
					</summary>
					<pre class="text-ink-faint mt-2 overflow-auto text-xs">{message}</pre>
				</details>
			{/if}

			<div class="flex flex-wrap justify-center gap-3">
				<Button onclick={() => window.location.reload()} variant="outline" class="gap-2">
					<RotateCcw class="h-4 w-4" />
					Try Again
				</Button>
				<a href="/">
					<Button class="gap-2">
						<Home class="h-4 w-4" />
						Go Home
					</Button>
				</a>
			</div>
		</Panel>
	</Container>
</div>

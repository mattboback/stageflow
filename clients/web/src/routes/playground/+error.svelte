<script lang="ts">
	import { page } from '$app/state';
	import { Button, Container, Panel } from '$lib/components/ui';
	import { SITE } from '$lib/config/site';
	import { AlertTriangle, ArrowLeft, Play, RotateCcw } from 'lucide-svelte';

	const status = $derived(page.status);
	const message = $derived(page.error?.message ?? 'An unexpected error occurred');
</script>

<svelte:head>
	<title>Playground Error | {SITE.name}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="bg-paper flex min-h-screen items-center justify-center py-16">
	<Container class="max-w-lg text-center">
		<Panel padding="xl" rounded="xl" class="shadow-md">
			<div
				class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-amber-100"
			>
				<AlertTriangle class="h-8 w-8 text-amber-600" />
			</div>

			<p class="text-accent mb-2 font-mono text-sm font-semibold">{status}</p>
			<h1 class="text-ink mb-3 text-2xl font-bold">Playground Error</h1>
			<p class="text-ink-muted mb-6">
				Something went wrong while loading the scan playground. This could be a temporary issue.
			</p>

			<details class="bg-surface-muted mb-6 rounded-lg p-4 text-left">
				<summary class="text-ink-muted cursor-pointer text-sm font-medium">
					Technical Details
				</summary>
				<pre class="text-ink-faint mt-2 overflow-auto text-xs">{message}</pre>
			</details>

			<div class="flex flex-wrap justify-center gap-3">
				<Button onclick={() => window.location.reload()} variant="outline" class="gap-2">
					<RotateCcw class="h-4 w-4" />
					Try Again
				</Button>
				<a href="/playground">
					<Button class="gap-2">
						<Play class="h-4 w-4" />
						Restart Playground
					</Button>
				</a>
			</div>

			<div class="border-line mt-6 border-t pt-6">
				<a
					href="/"
					class="text-ink-muted hover:text-accent inline-flex items-center gap-2 text-sm transition-colors"
				>
					<ArrowLeft class="h-4 w-4" />
					Back to Home
				</a>
			</div>
		</Panel>
	</Container>
</div>

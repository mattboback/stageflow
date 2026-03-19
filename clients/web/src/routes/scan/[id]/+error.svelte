<script lang="ts">
import { page } from "$app/state";
import { Button, Container, Panel } from "$lib/components/ui";
import { SITE } from "$lib/config/site";
import {
	AlertTriangle,
	ArrowLeft,
	FileSearch,
	Play,
	RotateCcw,
} from "lucide-svelte";

const status = $derived(page.status);
const message = $derived(page.error?.message ?? "An unexpected error occurred");
const scanId = $derived(page.params.id ?? "unknown");

const isNotFound = $derived(status === 404);
const title = $derived(isNotFound ? "Scan Not Found" : "Scan Error");
const description = $derived(
	isNotFound
		? `We couldn't find a scan with ID "${scanId}". It may have expired or never existed.`
		: "Something went wrong while loading your scan results.",
);
</script>

<svelte:head>
	<title>{title} | {SITE.name}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<div class="bg-paper flex min-h-screen items-center justify-center py-16">
	<Container class="max-w-lg text-center">
		<Panel padding="xl" rounded="xl" class="shadow-md">
			<div
				class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full"
				class:bg-amber-100={!isNotFound}
				class:bg-surface-muted={isNotFound}
			>
				{#if isNotFound}
					<FileSearch class="text-ink-faint h-8 w-8" />
				{:else}
					<AlertTriangle class="h-8 w-8 text-amber-600" />
				{/if}
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
				{#if !isNotFound}
					<Button onclick={() => window.location.reload()} variant="outline" class="gap-2">
						<RotateCcw class="h-4 w-4" />
						Try Again
					</Button>
				{/if}
				<a href="/playground">
					<Button class="gap-2">
						<Play class="h-4 w-4" />
						Start New Scan
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

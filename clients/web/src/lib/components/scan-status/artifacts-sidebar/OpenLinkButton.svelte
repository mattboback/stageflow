<script lang="ts">
import { Button } from "$lib/components/ui";
import { ExternalLink } from "lucide-svelte";

interface Props {
	label: string;
	url: string | null | undefined;
}

const { label, url }: Props = $props();

function open() {
	if (!url) return;
	const opened = window.open(url, "_blank", "noopener,noreferrer");
	if (opened) opened.opener = null;
}
</script>

{#if url}
	<Button variant="outline" class="h-10 w-full justify-between rounded-lg text-sm" onclick={open}>
		<span class="flex items-center gap-2">
			<ExternalLink class="h-4 w-4" />
			{label}
		</span>
		<span class="text-ink-faint text-xs">Open</span>
	</Button>
{/if}

<script lang="ts">
import { buttonVariants } from "$lib/components/ui";
import { cn } from "$lib/utils";
import { Check, Copy, Send } from "lucide-svelte";

interface Props {
	email: string;
	class?: string;
}

let { email, class: className = "" }: Props = $props();

let copied = $state(false);

async function handleCopy(e: MouseEvent) {
	e.preventDefault();
	try {
		await navigator.clipboard.writeText(email);
		copied = true;
		setTimeout(() => {
			copied = false;
		}, 2000);
	} catch (err) {
		console.error("Failed to copy text: ", err);
	}
}
</script>

<div class={cn('flex flex-col gap-3 sm:flex-row', className)}>
	<a
		href="mailto:{email}"
		class={cn(buttonVariants({ variant: 'glow', size: 'lg' }), 'group w-full sm:w-auto')}
	>
		<Send class="mr-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
		{email}
	</a>
	<button
		onclick={handleCopy}
		class={cn(buttonVariants({ variant: 'outline', size: 'lg' }), 'group w-full sm:w-auto')}
		aria-label="Copy email address"
	>
		{#if copied}
			<Check class="mr-2 h-4 w-4 text-emerald-600" />
			<span class="text-emerald-700">Copied!</span>
		{:else}
			<Copy class="text-ink-faint group-hover:text-ink mr-2 h-4 w-4" />
			Copy
		{/if}
	</button>
</div>

<script lang="ts">
import { Label } from "$lib/components/ui";
import { cn } from "$lib/utils";
import { CheckCircle2, FileUp, Globe } from "lucide-svelte";

type Mode = "url" | "zip";

interface Props {
	mode: Mode;
	onModeChange: (mode: Mode) => void;
}

let { mode, onModeChange }: Props = $props();

function selectableSurfaceClass(base: string, isSelected: boolean) {
	return cn(
		base,
		"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-paper",
		isSelected
			? "border-accent/60 bg-accent/5 shadow-[0_0_0_1px_rgba(220,38,38,0.18)]"
			: "border-line bg-surface hover:border-accent/30 hover:bg-surface-muted",
	);
}
</script>

<div>
	<Label class="mb-3 block text-sm font-bold">Input Type</Label>
	<div class="grid grid-cols-2 gap-3">
		<button
			type="button"
			onclick={() => onModeChange('url')}
			class={selectableSurfaceClass(
				'group relative flex flex-col items-center gap-2.5 rounded-2xl border p-4 transition-all duration-200',
				mode === 'url'
			)}
		>
			{#if mode === 'url'}
				<div class="absolute top-2 right-2">
					<CheckCircle2 class="text-accent h-4 w-4" />
				</div>
			{/if}
			<div
				class={cn(
					'flex h-11 w-11 items-center justify-center rounded-xl transition-colors',
					mode === 'url'
						? 'bg-accent text-white'
						: 'bg-surface-muted text-ink-muted group-hover:bg-accent/10 group-hover:text-accent'
				)}
			>
				<Globe class="h-6 w-6" />
			</div>
			<div class="text-center">
				<span class="text-ink block text-sm font-bold">Live URLs</span>
				<span class="text-ink-muted text-xs">Scan websites directly</span>
			</div>
		</button>
		<button
			type="button"
			onclick={() => onModeChange('zip')}
			class={selectableSurfaceClass(
				'group relative flex flex-col items-center gap-2.5 rounded-2xl border p-4 transition-all duration-200',
				mode === 'zip'
			)}
		>
			{#if mode === 'zip'}
				<div class="absolute top-2 right-2">
					<CheckCircle2 class="text-accent h-4 w-4" />
				</div>
			{/if}
			<div
				class={cn(
					'flex h-11 w-11 items-center justify-center rounded-xl transition-colors',
					mode === 'zip'
						? 'bg-accent text-white'
						: 'bg-surface-muted text-ink-muted group-hover:bg-accent/10 group-hover:text-accent'
				)}
			>
				<FileUp class="h-6 w-6" />
			</div>
			<div class="text-center">
				<span class="text-ink block text-sm font-bold">ZIP Archive</span>
				<span class="text-ink-muted text-xs">Upload static site</span>
			</div>
		</button>
	</div>
</div>

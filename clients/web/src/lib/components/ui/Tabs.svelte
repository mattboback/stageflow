<script lang="ts">
import type { Snippet } from "svelte";

import { cn } from "$lib/utils";

export interface Tab {
	id: string;
	label: string;
	content?: Snippet;
}

interface Props {
	tabs: Tab[];
	defaultTab?: string;
	value?: string;
	onValueChange?: (_tabId: string) => void;
	class?: string;
	panel?: Snippet<[string]>;
}

const {
	tabs,
	defaultTab,
	value,
	onValueChange,
	class: className,
	panel,
}: Props = $props();

let internalValue = $state<string | null>(null);
const activeTab = $derived(
	value ?? internalValue ?? defaultTab ?? tabs[0]?.id ?? "",
);
const activeTabContent = $derived(
	tabs.find((tab) => tab.id === activeTab)?.content,
);

function handleTabChange(tabId: string) {
	onValueChange?.(tabId);
	if (value === undefined) {
		internalValue = tabId;
	}
}
</script>

<div class={cn('w-full', className)}>
	<!-- Tab Headers -->
	<div class="no-scrollbar border-line mb-6 flex gap-2 overflow-x-auto border-b pb-px">
		{#each tabs as tab (tab.id)}
			<button
				onclick={() => handleTabChange(tab.id)}
				class={cn(
					'border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap transition-all',
					activeTab === tab.id
						? 'border-accent text-accent'
						: 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
				)}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Tab Content -->
	<div class="animate-fade-in">
		{#if panel}
			{@render panel(activeTab)}
		{:else if activeTabContent}
			{@render activeTabContent()}
		{/if}
	</div>
</div>

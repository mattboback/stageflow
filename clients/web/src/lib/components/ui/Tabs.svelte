<script lang="ts">
	import type { Snippet } from 'svelte';

	import { cn } from '$lib/utils';

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

	let { tabs, defaultTab, value, onValueChange, class: className, panel }: Props = $props();

	let internalValue = $state<string | null>(null);
	const activeTab = $derived(value ?? internalValue ?? defaultTab ?? tabs[0]?.id ?? '');
	const activeTabContent = $derived(tabs.find((tab) => tab.id === activeTab)?.content);
	const tabListId = $derived(`tabs-${tabs.map((tab) => tab.id).join('-')}`);

	function handleTabChange(tabId: string) {
		onValueChange?.(tabId);
		if (value === undefined) {
			internalValue = tabId;
		}
	}

	function focusTab(tabId: string) {
		const node = document.getElementById(`tab-${tabId}`);
		if (node instanceof HTMLButtonElement) {
			node.focus();
		}
	}

	function handleTabKeydown(event: KeyboardEvent, index: number) {
		if (tabs.length === 0) return;

		let nextIndex = index;
		switch (event.key) {
			case 'ArrowRight':
			case 'ArrowDown':
				nextIndex = (index + 1) % tabs.length;
				break;
			case 'ArrowLeft':
			case 'ArrowUp':
				nextIndex = (index - 1 + tabs.length) % tabs.length;
				break;
			case 'Home':
				nextIndex = 0;
				break;
			case 'End':
				nextIndex = tabs.length - 1;
				break;
			default:
				return;
		}

		event.preventDefault();
		const nextTab = tabs[nextIndex];
		if (!nextTab) return;
		handleTabChange(nextTab.id);
		focusTab(nextTab.id);
	}
</script>

<div class={cn('w-full', className)}>
	<!-- Tab Headers -->
	<div
		class="no-scrollbar border-line mb-6 flex gap-2 overflow-x-auto border-b pb-px"
		role="tablist"
		aria-orientation="horizontal"
		id={tabListId}
	>
		{#each tabs as tab, index (tab.id)}
			<button
				onclick={() => handleTabChange(tab.id)}
				onkeydown={(event) => handleTabKeydown(event, index)}
				class={cn(
					'border-b-2 px-4 py-2.5 text-sm font-medium whitespace-nowrap transition-[color,border-color]',
					activeTab === tab.id
						? 'border-accent text-accent'
						: 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'
				)}
				role="tab"
				id={`tab-${tab.id}`}
				aria-selected={activeTab === tab.id}
				aria-controls={`panel-${tab.id}`}
				tabindex={activeTab === tab.id ? 0 : -1}
			>
				{tab.label}
			</button>
		{/each}
	</div>

	<!-- Tab Content -->
	<div
		class="animate-fade-in"
		role="tabpanel"
		id={`panel-${activeTab}`}
		aria-labelledby={`tab-${activeTab}`}
		tabindex="0"
	>
		{#if panel}
			{@render panel(activeTab)}
		{:else if activeTabContent}
			{@render activeTabContent()}
		{/if}
	</div>
</div>

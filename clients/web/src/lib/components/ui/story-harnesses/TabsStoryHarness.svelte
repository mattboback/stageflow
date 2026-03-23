<script lang="ts">
	import Tabs, { type Tab } from '../Tabs.svelte';

	interface Props {
		defaultTab?: string;
		onValueChange?: (tabId: string) => void;
	}

	let { defaultTab = 'overview', onValueChange }: Props = $props();

	const tabs: Tab[] = [
		{ id: 'overview', label: 'Overview' },
		{ id: 'accessibility', label: 'Accessibility' },
		{ id: 'security', label: 'Security' }
	];

	let lastSelection = $derived(defaultTab);

	function handleValueChange(tabId: string) {
		lastSelection = tabId;
		onValueChange?.(tabId);
	}
</script>

<div class="w-[min(92vw,42rem)] space-y-3 text-black">
	<Tabs {tabs} {defaultTab} onValueChange={handleValueChange}>
		{#snippet panel(activeTab)}
			<div
				aria-hidden="true"
				class="border-line bg-surface rounded-md border p-3 text-sm text-black"
				data-testid="active-panel"
				style="background-color: #ffffff; color: #000000; font-size: 20px; line-height: 1.5;"
			>
				Active panel: {activeTab}
			</div>
		{/snippet}
	</Tabs>
	<p class="text-ink-muted text-sm" data-testid="last-selection">Last selection: {lastSelection}</p>
</div>

<script lang="ts">
import type {
	PanelInteractive,
	PanelPadding,
	PanelRounded,
	PanelVariant,
} from "../panel";

import Panel from "../Panel.svelte";

interface Props {
	variant?: PanelVariant;
	padding?: PanelPadding;
	rounded?: PanelRounded;
	interactive?: PanelInteractive;
	title?: string;
	body?: string;
}

const {
	variant = "default",
	padding = "md",
	rounded = "lg",
	interactive = false,
	title = "Panel title",
	body = "Panel body content",
}: Props = $props();
</script>

<Panel {variant} {padding} {rounded} {interactive} class="w-[min(92vw,34rem)]" data-testid="panel">
	<h3 class="text-base font-semibold" data-testid="panel-title">{title}</h3>
	<p class="mt-2 text-sm text-ink-muted">{body}</p>
</Panel>

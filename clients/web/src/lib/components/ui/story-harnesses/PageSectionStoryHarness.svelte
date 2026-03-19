<script lang="ts">
import PageSection from "../PageSection.svelte";

type SectionPadding = "default" | "page" | "none";

interface Props {
	padding?: SectionPadding;
	disableContainer?: boolean;
	containerClass?: string;
}

const {
	padding = "default",
	disableContainer = false,
	containerClass,
}: Props = $props();
</script>

<PageSection {padding} {disableContainer} {...(containerClass !== undefined ? { containerClass } : {})} data-testid="page-section">
	<div class="rounded-md border border-line bg-surface p-4" data-testid="page-section-child">
		Page section content
	</div>
</PageSection>

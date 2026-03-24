<script lang="ts">
	interface Props {
		class?: string;
		size?: number;
	}

	let { class: className, size = 20 }: Props = $props();
</script>

<svg
	viewBox="0 0 64 64"
	width={size}
	height={size}
	class={className}
	fill="none"
	aria-hidden="true"
>
	<rect x="8" y="14" width="28" height="8" rx="2" fill="currentColor" opacity="0.4" />
	<rect x="16" y="28" width="32" height="8" rx="2" fill="currentColor" opacity="0.7" />
	<rect x="24" y="42" width="32" height="8" rx="2" fill="currentColor" />
</svg>

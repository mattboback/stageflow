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
	<rect width="64" height="64" rx="14" fill="currentColor" />
	<text
		x="32"
		y="44"
		text-anchor="middle"
		font-size="30"
		font-family="system-ui, -apple-system, sans-serif"
		font-weight="800"
		letter-spacing="-2"
		fill="#ffffff"
	>SF</text>
</svg>

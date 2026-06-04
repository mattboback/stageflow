<script lang="ts">
	import type { ChipCaps, ChipInteractive, ChipSize, ChipTone } from '../chip';

	import Chip from '../Chip.svelte';

	interface Props {
		label: string;
		tone?: ChipTone;
		size?: ChipSize;
		caps?: ChipCaps;
		interactive?: ChipInteractive;
		as?: 'span' | 'button' | 'a';
	}

	let {
		label,
		tone = 'default',
		size = 'sm',
		caps = false,
		interactive = false,
		as = 'span'
	}: Props = $props();
</script>

<Chip {as} {tone} {size} {caps} {interactive} data-testid="chip">{label}</Chip>

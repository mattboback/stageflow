<script lang="ts">
	import Label from '../Label.svelte';

	interface Props {
		required?: boolean;
	}

	let { required = false }: Props = $props();
</script>

<Label for="project-name" {required} data-testid="label">Project name</Label>
<input id="project-name" class="sr-only" />

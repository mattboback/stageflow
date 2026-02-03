<script lang="ts">
	import { SelectField } from '$lib/components/ui';
</script>

<div>
	<div data-testid="select-default">
		<SelectField>
			<option value="a">A</option>
		</SelectField>
	</div>
	<div data-testid="select-prominent">
		<SelectField variant="prominent">
			<option value="b">B</option>
		</SelectField>
	</div>
</div>

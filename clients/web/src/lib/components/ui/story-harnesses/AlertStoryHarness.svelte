<script lang="ts">
import type { AlertVariant } from "../alert";

import Alert from "../Alert.svelte";

interface Props {
	variant?: AlertVariant;
	message?: string;
}

const { variant = "info", message = "Scan started successfully." }: Props =
	$props();
</script>

<Alert {variant} data-testid="alert">
	{message}
</Alert>

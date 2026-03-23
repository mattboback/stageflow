<script lang="ts">
	import Modal from '../Modal.svelte';

	interface Props {
		title?: string;
		openLabel?: string;
		closeLabel?: string;
		closeOnBackdrop?: boolean;
		closeOnEscape?: boolean;
		trapFocus?: boolean;
		onClose?: () => void;
	}

	let {
		title = 'Scanner details',
		openLabel = 'Open modal',
		closeLabel = 'Close modal',
		closeOnBackdrop = true,
		closeOnEscape = true,
		trapFocus = true,
		onClose
	}: Props = $props();

	let open = $state(false);

	function handleOpen() {
		open = true;
	}

	function handleClose() {
		open = false;
		onClose?.();
	}

	function preventNavigate(event: MouseEvent) {
		event.preventDefault();
	}
</script>

<div class="w-[min(92vw,38rem)] space-y-3">
	<button
		data-testid="open-modal"
		class="rounded-md border px-3 py-2"
		onclick={handleOpen}
		type="button"
	>
		{openLabel}
	</button>

	<Modal
		{open}
		onClose={handleClose}
		{closeOnBackdrop}
		{closeOnEscape}
		{trapFocus}
		ariaLabel={title}
		contentClass="bg-surface border border-line rounded-lg p-4"
	>
		<div data-testid="modal-body" class="space-y-3">
			<h2 class="text-lg font-semibold">{title}</h2>
			<div class="flex flex-wrap items-center gap-2">
				<button
					data-modal-initial-focus
					data-testid="close-modal"
					class="rounded-md border px-3 py-2"
					onclick={handleClose}
					type="button"
				>
					{closeLabel}
				</button>
				<a
					data-testid="help-link"
					class="text-accent underline"
					href="https://example.com/docs"
					onclick={preventNavigate}
				>
					Help docs
				</a>
				<button data-testid="last-button" class="rounded-md border px-3 py-2" type="button">
					Last action
				</button>
			</div>
		</div>
	</Modal>

	<p data-testid="modal-state" class="text-ink-muted text-sm">state: {open ? 'open' : 'closed'}</p>
</div>

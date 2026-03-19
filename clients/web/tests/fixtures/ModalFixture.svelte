<script lang="ts">
import { Modal } from "$lib/components/ui";

let open = $state(false);

function openModal() {
	open = true;
}

function closeModal() {
	open = false;
}
</script>

<button data-testid="open-button" onclick={openModal} type="button">Open modal</button>

<Modal {open} onClose={closeModal} ariaLabel="Test modal" contentClass="bg-surface p-4">
	<div data-testid="modal-body">
		<button data-modal-initial-focus data-testid="close-button" onclick={closeModal} type="button">
			Close
		</button>
		<a data-testid="help-link" href="https://example.com"> Help </a>
		<button data-testid="last-button" type="button">Last</button>
	</div>
</Modal>

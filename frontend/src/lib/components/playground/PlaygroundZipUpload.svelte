<script lang="ts">
	import { isZipFilename } from '$lib/components/playground/playground-utils';
	import { Button, Label } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { FileUp, Upload } from 'lucide-svelte';

	interface Props {
		file: File | null;
		onFileChange: (file: File | null) => void;
		onError: (error: string) => void;
	}

	const { file, onFileChange, onError }: Props = $props();

	let fileInputRef = $state<HTMLInputElement | null>(null);
	let isDragOver = $state(false);

	function handleFileChange(event: Event) {
		const target = event.target as HTMLInputElement;
		const selectedFile = target.files?.[0];
		if (selectedFile) {
			if (!isZipFilename(selectedFile.name)) {
				onError('Please select a ZIP file');
				onFileChange(null);
				return;
			}
			onFileChange(selectedFile);
		}
	}

	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		isDragOver = true;
	}

	function handleDragLeave() {
		isDragOver = false;
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragOver = false;
		const droppedFile = event.dataTransfer?.files?.[0];
		if (droppedFile) {
			if (!isZipFilename(droppedFile.name)) {
				onError('Please select a ZIP file');
				onFileChange(null);
				return;
			}
			onFileChange(droppedFile);
		}
	}

	function clearFile(e: MouseEvent) {
		e.stopPropagation();
		onFileChange(null);
		if (fileInputRef) fileInputRef.value = '';
	}
</script>

<div class="animate-fade-in">
	<Label class="mb-3 block text-sm font-semibold">Static Site Archive</Label>
	<div
		role="button"
		tabindex="0"
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
		onclick={() => fileInputRef?.click()}
		onkeydown={(e) => e.key === 'Enter' && fileInputRef?.click()}
		class={cn(
			'group relative flex cursor-pointer flex-col items-center justify-center rounded-2xl border border-dashed p-8 transition-all duration-200',
			isDragOver && 'border-accent bg-accent/5 scale-[1.01]',
			file
				? 'border-accent/60 bg-accent/5 shadow-[0_0_0_1px_rgba(220,38,38,0.18)]'
				: 'border-line hover:border-accent/30 hover:bg-surface-muted'
		)}
	>
		<input
			bind:this={fileInputRef}
			type="file"
			accept=".zip"
			onchange={handleFileChange}
			class="hidden"
		/>
		{#if file}
			<div class="bg-accent/10 mb-3 flex h-14 w-14 items-center justify-center rounded-2xl">
				<FileUp class="text-accent h-7 w-7" />
			</div>
			<p class="text-ink font-semibold">{file.name}</p>
			<p class="text-ink-muted stat-mono text-sm">
				{(file.size / (1024 * 1024)).toFixed(2)} MB
			</p>
			<Button variant="ghost" size="sm" class="mt-3" onclick={clearFile}>
				Remove file
			</Button>
		{:else}
			<div
				class="bg-surface-muted group-hover:bg-accent/10 mb-3 flex h-14 w-14 items-center justify-center rounded-2xl transition-colors"
			>
				<Upload class="text-ink-faint group-hover:text-accent h-7 w-7 transition-colors" />
			</div>
			<p class="text-ink mb-1 font-medium">Drop your ZIP file here</p>
			<p class="text-ink-muted text-sm">or click to browse</p>
		{/if}
	</div>
	<p class="text-ink-faint mt-2 flex items-center gap-1.5 text-xs">
		<span class="bg-ink-faint inline-block h-1 w-1 rounded-full"></span>
		Upload a ZIP containing your static site. Maximum 100MB.
	</p>
</div>

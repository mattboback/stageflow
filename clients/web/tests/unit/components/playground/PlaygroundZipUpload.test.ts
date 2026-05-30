import { MAX_ZIP_UPLOAD_BYTES } from '$lib/components/playground/playground-utils';
import PlaygroundZipUpload from '$lib/components/playground/PlaygroundZipUpload.svelte';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';

function fileWithSize(name: string, size: number): File {
	const file = new File(['x'], name);
	Object.defineProperty(file, 'size', { value: size });

	return file;
}

describe('PlaygroundZipUpload', () => {
	afterEach(() => {
		cleanup();
	});

	it('accepts a valid ZIP selected from the file input', async () => {
		const onFileChange = vi.fn();
		const onError = vi.fn();
		const file = fileWithSize('site.zip', 1024);

		const { container } = render(PlaygroundZipUpload, {
			props: { file: null, onFileChange, onError }
		});

		const input = container.querySelector<HTMLInputElement>('input[type="file"]');
		expect(input).not.toBeNull();

		await fireEvent.change(input as HTMLInputElement, { target: { files: [file] } });

		expect(onFileChange).toHaveBeenCalledWith(file);
		expect(onError).not.toHaveBeenCalled();
	});

	it('rejects non-ZIP and oversized selected files before upload', async () => {
		const onFileChange = vi.fn();
		const onError = vi.fn();
		const oversized = fileWithSize('site.zip', MAX_ZIP_UPLOAD_BYTES + 1);

		const { container } = render(PlaygroundZipUpload, {
			props: { file: null, onFileChange, onError }
		});
		const input = container.querySelector<HTMLInputElement>('input[type="file"]') as HTMLInputElement;

		await fireEvent.change(input, { target: { files: [new File(['x'], 'site.txt')] } });
		expect(onError).toHaveBeenLastCalledWith('Please select a ZIP file');
		expect(onFileChange).toHaveBeenLastCalledWith(null);

		await fireEvent.change(input, { target: { files: [oversized] } });
		expect(onError).toHaveBeenLastCalledWith('ZIP file must be 100MB or smaller');
		expect(onFileChange).toHaveBeenLastCalledWith(null);
	});

	it('accepts and rejects dropped files using the same validation path', async () => {
		const onFileChange = vi.fn();
		const onError = vi.fn();
		const valid = fileWithSize('drop.zip', 512);

		render(PlaygroundZipUpload, {
			props: { file: null, onFileChange, onError }
		});

		const button = screen.getByRole('button', { name: 'Choose a ZIP file to upload' });
		await fireEvent.drop(button, { dataTransfer: { files: [valid] } });
		expect(onFileChange).toHaveBeenLastCalledWith(valid);

		await fireEvent.drop(button, { dataTransfer: { files: [new File(['x'], 'drop.png')] } });
		expect(onError).toHaveBeenLastCalledWith('Please select a ZIP file');
		expect(onFileChange).toHaveBeenLastCalledWith(null);
	});

	it('renders selected file actions and clears the current file', async () => {
		const onFileChange = vi.fn();
		const onError = vi.fn();
		const selected = fileWithSize('selected.zip', 2048);

		render(PlaygroundZipUpload, {
			props: { file: selected, onFileChange, onError }
		});

		expect(screen.getByText('selected.zip')).toBeInTheDocument();
		expect(screen.getByText('0.00 MB')).toBeInTheDocument();

		await fireEvent.click(screen.getByRole('button', { name: 'Remove file' }));
		expect(onFileChange).toHaveBeenCalledWith(null);
		expect(onError).not.toHaveBeenCalled();
	});
});

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { DiscoveryPreview } from './DiscoveryPreview';

const result = {
	urls: ['https://example.com/', 'https://example.com/about'],
	source: 'sitemap' as const,
	truncated: true,
	complete: false,
	warnings: ['Discovery reached its link-depth limit.']
};

describe('website discovery selection', () => {
	it('shows incomplete coverage and submits only selected pages', () => {
		const onConfirm = vi.fn();
		render(<DiscoveryPreview result={result} onConfirm={onConfirm} onCancel={() => {}} />);
		expect(screen.getByRole('status').textContent).toContain('coverage is incomplete');
		expect(screen.getByText(result.warnings[0]!)).toBeTruthy();
		fireEvent.click(screen.getByRole('checkbox', { name: result.urls[1]! }));
		fireEvent.click(screen.getByRole('button', { name: 'Use selected pages' }));
		expect(onConfirm).toHaveBeenCalledWith([result.urls[0]]);
	});
	it('prevents empty selection and cancels without submission', () => {
		const onConfirm = vi.fn();
		const onCancel = vi.fn();
		render(<DiscoveryPreview result={result} onConfirm={onConfirm} onCancel={onCancel} />);
		for (const box of screen.getAllByRole('checkbox')) fireEvent.click(box);
		expect(
			screen.getByRole<HTMLButtonElement>('button', { name: 'Use selected pages' }).disabled
		).toBe(true);
		fireEvent.click(screen.getByRole('button', { name: 'Cancel discovery' }));
		expect(onCancel).toHaveBeenCalledOnce();
		expect(onConfirm).not.toHaveBeenCalled();
	});
	it('keeps older truncated responses readable', () => {
		render(
			<DiscoveryPreview
				result={{ urls: result.urls, source: 'crawl', truncated: true }}
				onConfirm={() => {}}
				onCancel={() => {}}
			/>
		);
		expect(screen.getByRole('status').textContent).toContain('coverage is incomplete');
	});
});

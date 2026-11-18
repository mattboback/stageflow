import { describe, expect, it } from 'vitest';
import { useState } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';

import { useRovingTabList } from './useRovingTabList';

const TABS = ['Review', 'Fix', 'Evidence'] as const;

function Tablist() {
	const [selected, setSelected] = useState(0);
	const moveFocus = useRovingTabList(TABS.length, setSelected);

	return (
		<div role="tablist">
			{TABS.map((label, index) => (
				<button
					key={label}
					type="button"
					role="tab"
					aria-selected={index === selected}
					tabIndex={index === selected ? 0 : -1}
					onKeyDown={(event) => moveFocus(event, index)}
				>
					{label}
				</button>
			))}
		</div>
	);
}

function tab(name: string): HTMLElement {
	return screen.getByRole('tab', { name });
}

function expectSelectedAndFocused(name: string) {
	expect(tab(name).getAttribute('aria-selected')).toBe('true');
	expect(document.activeElement).toBe(tab(name));
}

describe('useRovingTabList', () => {
	it('ArrowRight selects and focuses the next tab, wrapping at the end', () => {
		render(<Tablist />);

		fireEvent.keyDown(tab('Review'), { key: 'ArrowRight' });
		expectSelectedAndFocused('Fix');

		fireEvent.keyDown(tab('Fix'), { key: 'ArrowRight' });
		fireEvent.keyDown(tab('Evidence'), { key: 'ArrowRight' });
		expectSelectedAndFocused('Review');
	});

	it('ArrowLeft wraps backwards from the first tab', () => {
		render(<Tablist />);

		fireEvent.keyDown(tab('Review'), { key: 'ArrowLeft' });
		expectSelectedAndFocused('Evidence');
	});

	it('Home and End jump to the ends', () => {
		render(<Tablist />);

		fireEvent.keyDown(tab('Review'), { key: 'End' });
		expectSelectedAndFocused('Evidence');

		fireEvent.keyDown(tab('Evidence'), { key: 'Home' });
		expectSelectedAndFocused('Review');
	});

	it('leaves other keys alone so Tab can exit the list', () => {
		render(<Tablist />);

		// fireEvent returns false only when preventDefault was called.
		expect(fireEvent.keyDown(tab('Review'), { key: 'Tab' })).toBe(true);
		expect(tab('Review').getAttribute('aria-selected')).toBe('true');
	});
});

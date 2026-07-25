import { describe, expect, it, beforeEach } from 'vitest';

import { cycleTabFocus, focusableWithin } from './focus-trap';

/** Minimal Tab KeyboardEvent whose preventDefault we can observe. */
function tabEvent(shiftKey = false): KeyboardEvent {
	return new KeyboardEvent('keydown', { key: 'Tab', shiftKey, cancelable: true });
}

/**
 * Mounts `count` buttons and returns them positionally.
 *
 * Throws rather than returning possibly-undefined entries, so tests read as
 * assertions about behaviour instead of about array bounds.
 */
function mountButtons(count: number): HTMLButtonElement[] {
	const container = document.createElement('div');
	document.body.appendChild(container);
	return Array.from({ length: count }, (_, i) => {
		const button = document.createElement('button');
		button.textContent = `b${String(i)}`;
		container.appendChild(button);
		return button;
	});
}

function at<T>(items: readonly T[], index: number): T {
	const item = items[index];
	if (item === undefined) {
		throw new Error(`test fixture has no element at index ${String(index)}`);
	}
	return item;
}

describe('cycleTabFocus', () => {
	beforeEach(() => {
		document.body.innerHTML = '';
	});

	it('wraps from the last element forward to the first', () => {
		const nodes = mountButtons(3);
		const first = at(nodes, 0);
		const last = at(nodes, 2);
		last.focus();

		const event = tabEvent();
		expect(cycleTabFocus([first, last], event)).toBe(true);
		expect(document.activeElement).toBe(first);
		expect(event.defaultPrevented).toBe(true);
	});

	it('wraps from the first element backward to the last on Shift+Tab', () => {
		const nodes = mountButtons(3);
		const first = at(nodes, 0);
		const last = at(nodes, 2);
		first.focus();

		const event = tabEvent(true);
		expect(cycleTabFocus([first, last], event)).toBe(true);
		expect(document.activeElement).toBe(last);
	});

	it('leaves interior focus alone so the browser advances normally', () => {
		const nodes = mountButtons(3);
		const middle = at(nodes, 1);
		middle.focus();

		const event = tabEvent();
		expect(cycleTabFocus(nodes, event)).toBe(false);
		expect(document.activeElement).toBe(middle);
		expect(event.defaultPrevented).toBe(false);
	});

	it('pulls focus back inside when it has escaped the trap', () => {
		const nodes = mountButtons(2);
		const outside = document.createElement('button');
		document.body.appendChild(outside);
		outside.focus();

		expect(cycleTabFocus(nodes, tabEvent())).toBe(true);
		expect(document.activeElement).toBe(at(nodes, 0));
	});

	it('does nothing when there is nothing focusable', () => {
		const event = tabEvent();
		expect(cycleTabFocus([], event)).toBe(false);
		expect(event.defaultPrevented).toBe(false);
	});

	it('handles a single focusable element without moving focus off it', () => {
		const nodes = mountButtons(1);
		const only = at(nodes, 0);
		only.focus();

		expect(cycleTabFocus(nodes, tabEvent())).toBe(true);
		expect(document.activeElement).toBe(only);
	});
});

describe('focusableWithin', () => {
	beforeEach(() => {
		document.body.innerHTML = '';
	});

	it('skips disabled controls and tabindex="-1"', () => {
		const container = document.createElement('div');
		container.innerHTML = `
			<button id="ok">ok</button>
			<button id="off" disabled>off</button>
			<input id="text" />
			<input id="text-off" disabled />
			<div id="skip" tabindex="-1">skip</div>
			<div id="keep" tabindex="0">keep</div>
		`;
		document.body.appendChild(container);

		// jsdom reports zero layout for every element, so this asserts on the
		// selector's filtering; the visibility filter is covered by the Playwright
		// suite, which runs in a browser that actually lays elements out.
		const ids = focusableWithin(container).map((el) => el.id);
		expect(ids).not.toContain('off');
		expect(ids).not.toContain('text-off');
		expect(ids).not.toContain('skip');
	});
});

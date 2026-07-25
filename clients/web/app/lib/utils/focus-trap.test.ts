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

	/**
	 * jsdom performs no layout, so every element reports zero size and
	 * focusableWithin's visibility filter rejects all of them. Without this the
	 * assertions below pass against an empty array — proving nothing. Reporting a
	 * non-zero width leaves the selector and tab-order filtering as the only thing
	 * under test; real visibility is covered by the Playwright suite, which runs in
	 * a browser that lays elements out.
	 */
	function mount(html: string): HTMLElement {
		const container = document.createElement('div');
		container.innerHTML = html;
		document.body.appendChild(container);
		for (const el of Array.from(container.querySelectorAll<HTMLElement>('*'))) {
			Object.defineProperty(el, 'offsetWidth', { value: 10, configurable: true });
		}
		return container;
	}

	it('collects enabled, visible, tabbable controls in document order', () => {
		const container = mount(`
			<button id="ok">ok</button>
			<input id="text" />
			<div id="keep" tabindex="0">keep</div>
		`);

		expect(focusableWithin(container).map((el) => el.id)).toEqual(['ok', 'text', 'keep']);
	});

	it('skips disabled controls and tabindex="-1"', () => {
		const container = mount(`
			<button id="ok">ok</button>
			<button id="off" disabled>off</button>
			<input id="text" />
			<input id="text-off" disabled />
			<div id="skip" tabindex="-1">skip</div>
			<div id="keep" tabindex="0">keep</div>
		`);

		const ids = focusableWithin(container).map((el) => el.id);
		expect(ids).not.toContain('off');
		expect(ids).not.toContain('text-off');
		expect(ids).not.toContain('skip');
	});

	it('skips a button taken out of the tab order by tabindex="-1"', () => {
		// The selector's `[tabindex]:not([tabindex="-1"])` clause does not cover this:
		// the element still matches the `button` clause. IssueDetailModal renders
		// exactly this for its inactive tabs.
		const container = mount(`
			<button id="tab-active" tabindex="0">Active</button>
			<button id="tab-inactive" tabindex="-1">Inactive</button>
			<a id="link-inactive" href="#x" tabindex="-1">link</a>
			<input id="input-inactive" tabindex="-1" />
		`);

		expect(focusableWithin(container).map((el) => el.id)).toEqual(['tab-active']);
	});

	it('keeps the wrap boundary on the last real tab stop, not a roving-tabindex tab', () => {
		// The bug this guards: with an untabbable button collected last, `active === last`
		// never matched when Tab was pressed on the true final stop, so focus left the
		// dialog instead of wrapping.
		const container = mount(`
			<button id="close">close</button>
			<button id="tab-active" tabindex="0">Active</button>
			<button id="tab-inactive" tabindex="-1">Inactive</button>
		`);

		const focusable = focusableWithin(container);
		expect(focusable.map((el) => el.id)).toEqual(['close', 'tab-active']);

		at(focusable, focusable.length - 1).focus();
		const event = tabEvent();

		expect(cycleTabFocus(focusable, event)).toBe(true);
		expect(event.defaultPrevented).toBe(true);
		expect(document.activeElement?.id).toBe('close');
	});
});

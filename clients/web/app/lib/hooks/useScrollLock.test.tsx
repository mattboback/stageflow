import { afterEach, describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';

import { useScrollLock } from './useScrollLock';

function Dialog() {
	useScrollLock();
	return <div>dialog</div>;
}

afterEach(() => {
	document.body.style.overflow = '';
	document.body.style.paddingRight = '';
});

describe('useScrollLock', () => {
	it('locks the body while mounted and restores it on unmount', () => {
		const view = render(<Dialog />);
		expect(document.body.style.overflow).toBe('hidden');

		view.unmount();
		expect(document.body.style.overflow).toBe('');
	});

	it('restores whatever the body had before, not a hardcoded default', () => {
		document.body.style.overflow = 'scroll';

		const view = render(<Dialog />);
		expect(document.body.style.overflow).toBe('hidden');

		view.unmount();
		expect(document.body.style.overflow).toBe('scroll');
	});

	/*
	 * The reason the hook counts instead of holding a boolean: a report can have
	 * a confirm dialog open underneath an issue modal, and the inner one
	 * unmounting first must not hand the page back while the outer is still up.
	 */
	it('stays locked until the last of several dialogs closes', () => {
		const outer = render(<Dialog />);
		const inner = render(<Dialog />);

		inner.unmount();
		expect(document.body.style.overflow).toBe('hidden');

		outer.unmount();
		expect(document.body.style.overflow).toBe('');
	});
});

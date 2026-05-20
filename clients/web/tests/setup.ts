import '@testing-library/jest-dom/vitest';

// Polyfill Element.prototype.animate for JSDOM environments during Vitest runs
const proto = typeof Element !== 'undefined' ? (Element.prototype as Record<string, any>) : null;

if (proto && !proto.animate) {
	const noop = () => {
		// Mock empty function for tests
	};

	proto.animate = function () {
		return {
			addEventListener: noop,
			removeEventListener: noop,
			play: noop,
			pause: noop,
			cancel: noop,
			finish: noop,
			currentTime: 0,
			playState: 'finished',
			onfinish: null,
			oncancel: null,
			onremove: null
		} as unknown as Animation;
	};
}

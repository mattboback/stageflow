const sentry = await import('@sentry/node');
sentry.init({ enabled: false });

const lighthouse = await import('lighthouse');
if (typeof lighthouse.default !== 'function') {
	throw new TypeError('Lighthouse default export is unavailable');
}

process.stdout.write('Lighthouse/Sentry runtime imports initialized successfully.\n');

import { render, screen } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it } from 'vitest';

import { RouteFault } from './RouteFault';

function renderFault(props?: Partial<React.ComponentProps<typeof RouteFault>>) {
	const Stub = createRoutesStub([
		{
			path: '/',
			Component: () => (
				<RouteFault
					status={404}
					title="Page not found."
					detail="The page you requested doesn't exist."
					traceLine="route /missing not matched"
					traceHint="check the URL"
					{...props}
				/>
			)
		}
	]);
	return render(<Stub initialEntries={['/']} />);
}

describe('RouteFault', () => {
	it('renders the status, copy, and trace readout', () => {
		renderFault();

		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Page not found.');
		expect(screen.getByText("The page you requested doesn't exist.")).toBeTruthy();
		expect(screen.getAllByText('404').length).toBeGreaterThan(0);

		const trace = screen.getByRole('status');
		expect(trace.textContent).toContain('resolver');
		expect(trace.textContent).toContain('route /missing not matched');
		expect(trace.textContent).toContain('check the URL');
	});

	it('offers home and new-scan navigation by default', () => {
		renderFault();

		expect(screen.getByRole('link', { name: /back to home/i })).toBeTruthy();
		expect(screen.getByRole('link', { name: 'Configure a scan' })).toBeTruthy();
	});

	it('renders custom actions in place of the defaults', () => {
		renderFault({
			status: 500,
			actions: <a href="/scan/abc">Try again</a>
		});

		expect(screen.getByRole('link', { name: 'Try again' })).toBeTruthy();
		expect(screen.queryByRole('link', { name: /back to home/i })).toBeNull();
	});
});

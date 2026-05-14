import { Score, SeverityBar, StatusPill } from '$lib/components/ui';
import { render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { cleanup } from '@testing-library/svelte';

describe('Score', () => {
	afterEach(() => cleanup());

	it('renders rounded number and /100 denominator', () => {
		const { getByTestId } = render(Score, { props: { score: 87.4 } });
		expect(getByTestId('score-number')).toHaveTextContent('87');
		expect(getByTestId('score')).toHaveTextContent('/100');
	});

	it('renders em-dash placeholder when score is null', () => {
		const { getByTestId } = render(Score, { props: { score: null } });
		expect(getByTestId('score-number').textContent).toContain('—');
	});

	it('shows Strong pill for 94', () => {
		const { getByText } = render(Score, { props: { score: 94 } });
		expect(getByText('Strong')).toBeInTheDocument();
	});

	it('shows Failing pill for 42', () => {
		const { getByText } = render(Score, { props: { score: 42 } });
		expect(getByText('Failing')).toBeInTheDocument();
	});
});

describe('StatusPill', () => {
	afterEach(() => cleanup());

	it('renders default label for tone', () => {
		const { getByRole } = render(StatusPill, { props: { tone: 'strong' } });
		expect(getByRole('status')).toHaveTextContent('Strong');
	});

	it('renders custom label', () => {
		const { getByRole } = render(StatusPill, {
			props: { tone: 'failing', label: 'Critical' }
		});
		expect(getByRole('status')).toHaveTextContent('Critical');
	});
});

describe('SeverityBar', () => {
	afterEach(() => cleanup());

	it('renders aria label with all counts', () => {
		const { getByRole } = render(SeverityBar, {
			props: { counts: { critical: 3, serious: 5, moderate: 0, minor: 2, info: 1 } }
		});
		const img = getByRole('img');
		expect(img).toHaveAttribute('aria-label', expect.stringContaining('3 critical'));
		expect(img).toHaveAttribute('aria-label', expect.stringContaining('5 serious'));
	});

	it('renders empty-state with zero counts', () => {
		const { getByRole } = render(SeverityBar, { props: { counts: {} } });
		expect(getByRole('img')).toBeInTheDocument();
	});

	it('shows labels when showLabels=true', () => {
		const { getByText } = render(SeverityBar, {
			props: { counts: { critical: 7 }, showLabels: true }
		});
		expect(getByText('7')).toBeInTheDocument();
		expect(getByText('critical')).toBeInTheDocument();
	});
});

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ScoreHistory } from './ScoreHistory';

const points = [
	{ id: 'run-1', score: 64, at: '2026-09-20T14:00:00.000Z' },
	{ id: 'run-2', score: 60, at: '2026-09-23T14:00:00.000Z' },
	{ id: 'run-3', score: 57, at: '2026-09-26T14:00:00.000Z' }
];

/** The y coordinate of each point on the line, oldest first. */
function lineHeights(container: HTMLElement): number[] {
	const line = container.querySelector('polyline')?.getAttribute('points') ?? '';
	return line.split(' ').map((pair) => Number(pair.split(',')[1]));
}

describe('project score history', () => {
	it('names every score, oldest first, and which run is the baseline', () => {
		render(<ScoreHistory points={points} baselineId="run-2" baselineScore={60} />);
		expect(
			screen.getByRole('img', {
				name: 'Scores of the last 3 runs, oldest first: 64, 60 (baseline), 57'
			})
		).toBeTruthy();
	});

	it('marks the baseline and the latest run apart from the others', () => {
		const { container } = render(
			<ScoreHistory points={points} baselineId="run-2" baselineScore={60} />
		);
		const kinds = [...container.querySelectorAll('.score-history__run')].map(
			(run) => run.getAttribute('class')?.split('--')[1]
		);
		expect(kinds).toEqual(['run', 'baseline', 'latest']);
		expect(container.querySelector('.score-history__baseline')).not.toBeNull();
	});

	it('gives each run a hover title with its score', () => {
		const { container } = render(<ScoreHistory points={points} baselineId="run-2" />);
		const titles = [...container.querySelectorAll('title')].map((title) => title.textContent);
		expect(titles).toHaveLength(3);
		expect(titles[1]).toMatch(/score 60 · baseline$/);
		expect(titles[2]).toMatch(/score 57 · latest$/);
	});

	it('draws nothing until there are two scores to connect', () => {
		const { container } = render(<ScoreHistory points={points.slice(0, 1)} />);
		expect(container.innerHTML).toBe('');
	});

	it('keeps a one-point change a small step rather than the full height', () => {
		const { container } = render(
			<ScoreHistory
				points={[
					{ id: 'a', score: 60, at: '2026-09-20T14:00:00.000Z' },
					{ id: 'b', score: 61, at: '2026-09-23T14:00:00.000Z' }
				]}
			/>
		);
		const [first = 0, second = 0] = lineHeights(container);
		// The plot is 28px tall and spans at least ten points: 2.8px per point.
		expect(Math.abs(first - second)).toBeCloseTo(2.8, 1);
	});
});

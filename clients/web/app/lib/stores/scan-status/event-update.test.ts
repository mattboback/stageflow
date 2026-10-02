import { describe, expect, it } from 'vitest';

import type { ScanResult } from '../../types/scan';
import { applySseUpdate } from './event-update';
import { getLogMessage } from './log-messages';

const waiting: ScanResult = {
	id: 'job-1',
	state: 'PENDING',
	queue_position: 2,
	created_at: '2026-10-02T00:00:00Z',
	updated_at: '2026-10-02T00:00:00Z'
};

describe('queue updates', () => {
	it('moves a waiting job up the queue', () => {
		const next = applySseUpdate(waiting, { type: 'queue', state: 'PENDING', queue_position: 1 });
		expect(next.queue_position).toBe(1);
		expect(getLogMessage('PENDING', { type: 'queue', queue_position: 1 })).toBe('Queued, 1 ahead.');
	});

	it('keeps the position through other updates while the job waits', () => {
		expect(applySseUpdate(waiting, { type: 'status', state: 'PENDING' }).queue_position).toBe(2);
	});

	it('drops the position once the job is claimed', () => {
		const started = applySseUpdate(waiting, { type: 'status', state: 'EXTRACTING' });
		expect(started.state).toBe('EXTRACTING');
		expect('queue_position' in started).toBe(false);
	});

	it('names the next job in line', () => {
		expect(getLogMessage('PENDING', { type: 'queue', queue_position: 0 })).toBe(
			'Queued, next in line.'
		);
	});
});

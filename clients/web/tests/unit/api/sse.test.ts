import { createSSEStream } from '$lib/api/sse';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

type EventListenerFn = (event: Event) => void;

class MockEventSource {
	static CONNECTING = 0;
	static OPEN = 1;
	static CLOSED = 2;
	static instances: MockEventSource[] = [];

	readonly CONNECTING = MockEventSource.CONNECTING;
	readonly OPEN = MockEventSource.OPEN;
	readonly CLOSED = MockEventSource.CLOSED;
	readonly url: string;
	readonly withCredentials = false;

	readyState = MockEventSource.CONNECTING;
	onerror: ((event: Event) => void) | null = null;
	onmessage: ((event: MessageEvent<string>) => void) | null = null;
	onopen: ((event: Event) => void) | null = null;

	private readonly listeners: Record<string, EventListenerFn[]> = {};

	constructor(url: string) {
		this.url = url;
		MockEventSource.instances.push(this);
	}

	static reset() {
		MockEventSource.instances = [];
	}

	addEventListener(type: string, listener: EventListenerOrEventListenerObject | null) {
		if (!listener) {
			return;
		}
		const normalizedListener: EventListenerFn =
			typeof listener === 'function'
				? listener
				: (event: Event) => {
						listener.handleEvent(event);
					};
		this.listeners[type] = [...(this.listeners[type] ?? []), normalizedListener];
	}

	removeEventListener(type: string, listener: EventListenerOrEventListenerObject | null) {
		if (!listener) {
			return;
		}
		const handlers = this.listeners[type] ?? [];
		const normalizedListener: EventListenerFn =
			typeof listener === 'function'
				? listener
				: (event: Event) => {
						listener.handleEvent(event);
					};
		this.listeners[type] = handlers.filter((handler) => handler !== normalizedListener);
	}

	dispatchEvent(event: Event): boolean {
		for (const listener of this.listeners[event.type] ?? []) {
			listener(event);
		}
		return true;
	}

	close() {
		this.readyState = MockEventSource.CLOSED;
	}

	listenerCount(type: string): number {
		return (this.listeners[type] ?? []).length;
	}

	emitStatusRaw(data: string) {
		this.dispatchEvent(new MessageEvent('status', { data }));
	}

	emitDone() {
		this.dispatchEvent(new Event('done'));
	}

	emitError() {
		this.onerror?.(new Event('error'));
	}
}

const firstInstance = (): MockEventSource => {
	const instance = MockEventSource.instances.at(0);
	if (instance === undefined) {
		throw new Error('Expected EventSource instance');
	}
	return instance;
};

describe('SSE Stream', () => {
	beforeEach(() => {
		MockEventSource.reset();
		vi.stubGlobal('EventSource', MockEventSource);
	});

	afterEach(() => {
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it('initializes EventSource with correct URL', () => {
		createSSEStream('job-123', {
			onStatus: vi.fn(),
			onUpdate: vi.fn(),
			onDone: vi.fn(),
			onError: vi.fn()
		});

		expect(firstInstance().url).toContain('/api/v1/jobs/job-123/stream');
	});

	it('registers status/update/done listeners', () => {
		createSSEStream('job-123', {
			onStatus: vi.fn(),
			onUpdate: vi.fn(),
			onDone: vi.fn(),
			onError: vi.fn()
		});

		const instance = firstInstance();
		expect(instance.listenerCount('status')).toBe(1);
		expect(instance.listenerCount('update')).toBe(1);
		expect(instance.listenerCount('done')).toBe(1);
	});

	it('handles valid status events', () => {
		const onStatus = vi.fn();
		createSSEStream('job-123', {
			onStatus,
			onUpdate: vi.fn(),
			onDone: vi.fn(),
			onError: vi.fn()
		});

		const instance = firstInstance();
		instance.emitStatusRaw(JSON.stringify({ state: 'running' }));

		expect(onStatus).toHaveBeenCalledWith({ state: 'running' });
	});

	it('handles parse errors and closes stream after threshold', () => {
		const onError = vi.fn();
		createSSEStream(
			'job-123',
			{
				onStatus: vi.fn(),
				onUpdate: vi.fn(),
				onDone: vi.fn(),
				onError
			},
			{ onLog: vi.fn() }
		);

		const instance = firstInstance();
		instance.emitStatusRaw('invalid json');
		instance.emitStatusRaw('invalid json');
		instance.emitStatusRaw('invalid json');

		expect(onError).toHaveBeenCalledWith({
			message: 'Too many parse errors',
			kind: 'parse'
		});
		expect(instance.readyState).toBe(MockEventSource.CLOSED);
	});

	it('calls onOpen when the stream connects', () => {
		const onOpen = vi.fn();
		createSSEStream(
			'job-123',
			{
				onStatus: vi.fn(),
				onUpdate: vi.fn(),
				onDone: vi.fn(),
				onError: vi.fn()
			},
			{ onOpen }
		);

		const instance = firstInstance();
		instance.onopen?.(new Event('open'));

		expect(onOpen).toHaveBeenCalledTimes(1);
	});

	it('calls close on done event', () => {
		const onDone = vi.fn();
		createSSEStream('job-123', {
			onStatus: vi.fn(),
			onUpdate: vi.fn(),
			onDone,
			onError: vi.fn()
		});

		const instance = firstInstance();
		instance.emitDone();

		expect(onDone).toHaveBeenCalled();
		expect(instance.readyState).toBe(MockEventSource.CLOSED);
	});

	it('handles non-terminal connection errors', () => {
		const onError = vi.fn();
		createSSEStream('job-123', {
			onStatus: vi.fn(),
			onUpdate: vi.fn(),
			onDone: vi.fn(),
			onError
		});

		const instance = firstInstance();
		instance.readyState = MockEventSource.OPEN;
		instance.emitError();

		expect(onError).toHaveBeenCalledWith({
			message: 'Connection error',
			kind: 'transient'
		});
	});

	it('reports permanently closed streams with a closed error kind', () => {
		const onError = vi.fn();
		createSSEStream('job-123', {
			onStatus: vi.fn(),
			onUpdate: vi.fn(),
			onDone: vi.fn(),
			onError
		});

		const instance = firstInstance();
		instance.readyState = MockEventSource.CLOSED;
		instance.emitError();

		expect(onError).toHaveBeenCalledWith({
			message: 'Connection closed permanently',
			kind: 'closed'
		});
	});
});

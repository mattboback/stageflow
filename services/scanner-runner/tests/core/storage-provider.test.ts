import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import type { ScannerLogger, StorageConfig } from '../../src/core/types';

import { MinioStorageProvider, parseMinioEndpoint } from '../../src/core/storage-provider';

const fileMocks = vi.hoisted(() => ({
	getFileSize: vi.fn().mockResolvedValue(123),
	waitForFileReady: vi.fn().mockResolvedValue(true)
}));

vi.mock('../../src/core/storage-provider/files', () => fileMocks);

const mocks = vi.hoisted(() => {
	const client = {
		bucketExists: vi.fn(),
		makeBucket: vi.fn(),
		fPutObject: vi.fn(),
		putObject: vi.fn(),
		fGetObject: vi.fn(),
		statObject: vi.fn()
	};
	return {
		client,
		Client: vi.fn(function ClientMock() {
			return client;
		})
	};
});

vi.mock('minio', () => ({
	Client: mocks.Client
}));

const logger: ScannerLogger = {
	info: vi.fn(),
	warn: vi.fn(),
	error: vi.fn(),
	debug: vi.fn()
};

const baseConfig: StorageConfig = {
	endpoint: 'localhost:9000',
	accessKey: 'minio',
	secretKey: 'minio123',
	useSSL: false,
	bucket: 'scanner-artifacts'
};

describe('parseMinioEndpoint', () => {
	it('parses endpoints with explicit ports', () => {
		expect(parseMinioEndpoint('minio.local:9443', true)).toEqual({
			endPoint: 'minio.local',
			port: 9443
		});
	});

	it('defaults port based on SSL when not provided', () => {
		expect(parseMinioEndpoint('minio.local', false)).toEqual({
			endPoint: 'minio.local',
			port: 9000
		});
		expect(parseMinioEndpoint('minio.local', true)).toEqual({
			endPoint: 'minio.local',
			port: 443
		});
	});
});

describe('MinioStorageProvider', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('retries bucketExists on retryable errors before creating the bucket', async () => {
		const retryableError = { code: 'ECONNRESET', message: 'reset' };

		mocks.client.bucketExists
			.mockRejectedValueOnce(retryableError)
			.mockRejectedValueOnce(retryableError)
			.mockResolvedValue(false);
		mocks.client.makeBucket.mockResolvedValue(undefined);

		const provider = new MinioStorageProvider(baseConfig, logger);

		vi.useFakeTimers();
		try {
			const ensurePromise = provider.ensureBucket('scanner-artifacts');
			await vi.runAllTimersAsync();
			await ensurePromise;
		} finally {
			vi.useRealTimers();
		}

		expect(mocks.client.bucketExists).toHaveBeenCalledTimes(3);
		expect(mocks.client.makeBucket).toHaveBeenCalledWith('scanner-artifacts', '');
	});

	it('bails out on non-retryable errors', async () => {
		mocks.client.bucketExists.mockRejectedValueOnce({
			code: 'EINVAL',
			message: 'bad request'
		});

		const provider = new MinioStorageProvider(baseConfig, logger);

		await expect(provider.ensureBucket('scanner-artifacts')).rejects.toThrow(
			'MinIO bucketExists failed'
		);
		expect(mocks.client.bucketExists).toHaveBeenCalledTimes(1);
		expect(mocks.client.makeBucket).not.toHaveBeenCalled();
	});

	it('does nothing when the bucket already exists', async () => {
		mocks.client.bucketExists.mockResolvedValue(true);

		const provider = new MinioStorageProvider(baseConfig, logger);
		await provider.ensureBucket('scanner-artifacts');

		expect(mocks.client.bucketExists).toHaveBeenCalledTimes(1);
		expect(mocks.client.makeBucket).not.toHaveBeenCalled();
	});

	it('uploads buffers with a default content-type', async () => {
		mocks.client.putObject.mockResolvedValue(undefined);

		const provider = new MinioStorageProvider(baseConfig, logger);
		await provider.uploadBuffer('scanner-artifacts', 'job-1/results.json', Buffer.from('hi'));

		expect(mocks.client.putObject).toHaveBeenCalledWith(
			'scanner-artifacts',
			'job-1/results.json',
			expect.any(Buffer),
			2,
			{ 'Content-Type': 'application/octet-stream' }
		);
	});

	it('returns 0 when uploading an empty directory', async () => {
		const dir = await mkdtemp(join(tmpdir(), 'stageflow-empty-upload-'));

		try {
			const provider = new MinioStorageProvider(baseConfig, logger);
			const uploaded = await provider.uploadDirectory('scanner-artifacts', 'job-1/axe', dir);

			expect(uploaded).toBe(0);
			expect(mocks.client.fPutObject).not.toHaveBeenCalled();
		} finally {
			await rm(dir, { force: true, recursive: true });
		}
	});

	it('skips unreadable files and continues after upload failures', async () => {
		const dir = await mkdtemp(join(tmpdir(), 'stageflow-upload-'));
		await mkdir(join(dir, 'sub'));
		await writeFile(join(dir, 'a.txt'), 'a');
		await writeFile(join(dir, 'b.txt'), 'b');
		await writeFile(join(dir, 'sub', 'c.txt'), 'c');

		fileMocks.waitForFileReady.mockImplementation((filePath: string) =>
			Promise.resolve(!filePath.endsWith('a.txt'))
		);

		mocks.client.fPutObject.mockImplementation((_bucket: string, key: string) => {
			if (key.includes('/b.txt')) {
				return Promise.reject(new Error('upload failed'));
			}
			return Promise.resolve(undefined);
		});

		try {
			const provider = new MinioStorageProvider(baseConfig, logger);
			const uploaded = await provider.uploadDirectory('scanner-artifacts', 'job-1/axe', dir);

			expect(uploaded).toBe(1);
			expect(mocks.client.fPutObject).toHaveBeenCalledWith(
				'scanner-artifacts',
				'job-1/axe/sub/c.txt',
				join(dir, 'sub', 'c.txt'),
				{ 'Content-Type': 'text/plain' }
			);
		} finally {
			await rm(dir, { force: true, recursive: true });
		}
	});

	it('downloads files and wraps errors', async () => {
		mocks.client.fGetObject.mockResolvedValue(undefined);

		const provider = new MinioStorageProvider(baseConfig, logger);
		await provider.download('scanner-artifacts', 'job-1/results.json', '/tmp/results.json');

		expect(mocks.client.fGetObject).toHaveBeenCalledWith(
			'scanner-artifacts',
			'job-1/results.json',
			'/tmp/results.json'
		);

		mocks.client.fGetObject.mockRejectedValue({
			code: 'ECONNREFUSED',
			message: 'nope'
		});
		await expect(
			provider.download('scanner-artifacts', 'job-1/results.json', '/tmp/results.json')
		).rejects.toThrow('MinIO download failed');
	});

	it('checks existence via statObject', async () => {
		mocks.client.statObject.mockResolvedValue(undefined);
		const provider = new MinioStorageProvider(baseConfig, logger);
		await expect(provider.exists('scanner-artifacts', 'job-1/results.json')).resolves.toBe(true);

		mocks.client.statObject.mockRejectedValue(new Error('missing'));
		await expect(provider.exists('scanner-artifacts', 'job-1/results.json')).resolves.toBe(false);
	});
});

import { createServer, type Server } from 'node:http';
import type { AddressInfo } from 'node:net';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

import { BrowserManager } from '../../src/core/browser-manager';

describe('browser request routing lifecycle', () => {
	let server: Server;
	let origin: string;
	let blockedRequests = 0;
	const timers = new Set<ReturnType<typeof setTimeout>>();
	const manager = new BrowserManager({ deviceScaleFactor: 1 });

	beforeAll(async () => {
		server = createServer((req, res) => {
			if (req.url === '/redirect') {
				res.writeHead(302, { Location: '/fixture' }).end();
			} else if (req.url === '/private') {
				blockedRequests++;
				res.end('must not be reached');
			} else if (req.url?.startsWith('/slow')) {
				const timer = setTimeout(() => {
					timers.delete(timer);
					res.end('delayed response');
				}, 250);
				timers.add(timer);
			} else {
				res.setHeader('Content-Type', 'text/html');
				res.end(
					'<!doctype html><html lang="en"><title>Routing fixture</title><body>Ready</body></html>'
				);
			}
		});
		await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
		origin = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
		await manager.launch();
	});

	afterAll(async () => {
		await manager.close();
		for (const timer of timers) {
			clearTimeout(timer);
		}
		server.closeAllConnections();
		await new Promise<void>((resolve, reject) =>
			server.close((error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			})
		);
	});

	it('drains cancelled requests and delayed responses while repeatedly closing pages', async () => {
		const context = await manager.createContext();
		try {
			for (let run = 0; run < 20; run++) {
				const page = await context.newPage();
				await manager.navigateToPage(
					page,
					`${origin}/redirect`,
					{ type: 'domcontentloaded' },
					{
						allowedOrigins: [origin]
					}
				);
				await page.evaluate(() => {
					for (let i = 0; i < 12; i++) {
						const controller = new AbortController();
						void fetch(`/slow?cancel=${i}`, { signal: controller.signal }).catch(() => undefined);
						setTimeout(() => {
							controller.abort();
						}, 0);
						void fetch(`/slow?pending=${i}`).catch(() => undefined);
					}
					const frame = document.createElement('iframe');
					frame.src = '/redirect';
					document.body.append(frame);
					setTimeout(() => {
						frame.remove();
					}, 0);
				});
				await manager.closePage(page);
				expect(page.isClosed()).toBe(true);
				manager.assertPageHealthy(page);
			}
		} finally {
			await context.close();
		}
	}, 30_000);

	it('blocks redirects and subresources outside the allowed static origin', async () => {
		const context = await manager.createContext();
		const page = await context.newPage();
		const privateURL = origin.replace('127.0.0.1', 'localhost') + '/private';
		try {
			await manager.navigateToPage(
				page,
				`${origin}/fixture`,
				{ type: 'domcontentloaded' },
				{
					allowedOrigins: [origin]
				}
			);
			const fetched = await page.evaluate(async (url) => {
				try {
					await fetch(url);
					return true;
				} catch {
					return false;
				}
			}, privateURL);
			expect(fetched).toBe(false);
			expect(blockedRequests).toBe(0);
			await expect(
				manager.navigateToPage(page, privateURL, undefined, {
					allowedOrigins: [origin]
				})
			).rejects.toThrow('Blocked target URL');
			await manager.closePage(page);
		} finally {
			await context.close();
		}
	});
});

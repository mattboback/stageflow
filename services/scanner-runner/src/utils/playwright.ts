import fs from "node:fs";
import path from "node:path";

const DEFAULT_PLAYWRIGHT_BROWSERS_PATH = "/ms-playwright";

function sortChromiumDirsDesc(a: string, b: string): number {
	const aNum = Number.parseInt(a.replace("chromium-", ""), 10);
	const bNum = Number.parseInt(b.replace("chromium-", ""), 10);

	if (Number.isFinite(aNum) && Number.isFinite(bNum)) {
		return bNum - aNum;
	}

	return b.localeCompare(a);
}

/**
 * Resolve a Chromium executable path in Playwright's official Docker images.
 *
 * In those images, browsers are typically installed under `/ms-playwright` as:
 * `/ms-playwright/chromium-<rev>/chrome-linux/chrome`.
 *
 * This helper avoids hard-coding the browser revision number.
 */
export function resolvePlaywrightImageChromiumExecutablePath(): string | null {
	const configured = process.env.PLAYWRIGHT_BROWSERS_PATH?.trim();

	const roots = [
		configured && configured !== "0" ? configured : null,
		DEFAULT_PLAYWRIGHT_BROWSERS_PATH,
	].filter(Boolean) as string[];

	for (const root of roots) {
		if (!fs.existsSync(root)) {
			continue;
		}

		let entries: string[];
		try {
			entries = fs.readdirSync(root);
		} catch {
			continue;
		}

		const chromiumDirs = entries.filter((name) => /^chromium-\d+$/.test(name));
		chromiumDirs.sort(sortChromiumDirsDesc);

		for (const dir of chromiumDirs) {
			const candidate = path.join(root, dir, "chrome-linux", "chrome");
			if (fs.existsSync(candidate)) {
				return candidate;
			}

			const candidate64 = path.join(root, dir, "chrome-linux64", "chrome");
			if (fs.existsSync(candidate64)) {
				return candidate64;
			}
		}
	}

	return null;
}

import { compositeOver, contrastRatio, type Rgba } from '../app/lib/utils/contrast';
import { expect, test } from './fixtures';

// These are the 16 text nodes axe could not measure in the discovery scan.
// Verify their painted CSS colors independently; do not suppress axe's review
// findings. SVG overlap still needs the accompanying visual review.
const surfaces = [
	{
		path: '/',
		count: 11,
		selectors: [
			'.preview__stats > li > span',
			'.preview__stats > li > strong',
			'.preview__stats .delta',
			'.preview__stats small',
			'.preview__new',
			'.preview__evidence-title',
			'.preview__reg-note'
		]
	},
	{
		path: '/demo',
		count: 5,
		selectors: ['.rhead__dial .gauge__val', '.rhead__dial .gauge__cap', '.lh__cell .gauge__val']
	}
];

for (const width of [1440, 390]) {
	for (const theme of ['light', 'dark'] as const) {
		for (const surface of surfaces) {
			test(`${surface.path} reviewed text colors meet AA at ${width}px in ${theme}`, async ({
				page
			}) => {
				await page.setViewportSize({ width, height: 1000 });
				await page.emulateMedia({ colorScheme: theme });
				await page.goto(surface.path);
				await expect(page.locator('h1')).toBeVisible();
				await page.evaluate(() => document.fonts.ready);
				const text = page.locator(surface.selectors.join(', '));
				await expect(text).toHaveCount(surface.count);
				for (const target of await text.all()) await expect(target).toBeVisible();
				const samples = await text.evaluateAll((nodes) => {
					const canvas = document.createElement('canvas');
					canvas.width = canvas.height = 1;
					const context = canvas.getContext('2d', { willReadFrequently: true });
					if (!context) throw new Error('Canvas is required to resolve CSS colors');
					const color = (value: string) => {
						context.clearRect(0, 0, 1, 1);
						context.fillStyle = value;
						context.fillRect(0, 0, 1, 1);
						const [r = 0, g = 0, b = 0, a = 0] = context.getImageData(0, 0, 1, 1).data;
						return { r, g, b, a: a / 255 };
					};
					return nodes.map((node) => {
						const style = getComputedStyle(node);
						const backgrounds = [];
						for (let ancestor: Element | null = node; ancestor; ancestor = ancestor.parentElement) {
							const ancestorStyle = getComputedStyle(ancestor);
							if (
								ancestorStyle.backgroundImage !== 'none' ||
								ancestorStyle.opacity !== '1' ||
								ancestorStyle.mixBlendMode !== 'normal' ||
								ancestorStyle.filter !== 'none'
							) {
								throw new Error('Layer effects require a fresh visual contrast review');
							}
							backgrounds.unshift(color(ancestorStyle.backgroundColor));
						}
						return {
							text: node.textContent,
							foreground: color(style.color),
							backgrounds,
							fontSize: parseFloat(style.fontSize),
							fontWeight: parseFloat(style.fontWeight)
						};
					});
				});
				const evidence = samples.map((sample) => {
					const background = sample.backgrounds.reduce<Rgba>(compositeOverBackground, {
						r: 255,
						g: 255,
						b: 255,
						a: 1
					});
					const ratio = contrastRatio(sample.foreground, background);
					const large =
						sample.fontSize >= 24 || (sample.fontSize >= (14 * 4) / 3 && sample.fontWeight >= 700);
					const required = large ? 3 : 4.5;
					expect(
						ratio,
						`${sample.text ?? ''}: ${JSON.stringify(sample.foreground)} on ${JSON.stringify(background)}`
					).toBeGreaterThanOrEqual(required);
					return { ...sample, background, ratio, required };
				});
				const info = test.info();
				await info.attach('contrast-evidence', {
					body: JSON.stringify(evidence, null, 2),
					contentType: 'application/json'
				});
				await info.attach('visual-review', {
					body: await page.screenshot({
						fullPage: true,
						path: info.outputPath('visual-review.png')
					}),
					contentType: 'image/png'
				});
			});
		}
	}
}
function compositeOverBackground(background: Rgba, foreground: Rgba): Rgba {
	return compositeOver(foreground, background);
}

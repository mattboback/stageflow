export interface FixExample {
	title: string;
	before: string;
	after: string;
	notes?: string;
}

export function getFixExample(ruleId: string): FixExample | null {
	switch (ruleId) {
		case "image-alt":
			return {
				title: "Provide alternative text for images",
				before: '<img src="logo.gif">',
				after: '<img src="logo.gif" alt="Company logo">',
				notes:
					'Prefer meaningful alt text for informative images; use alt="" for purely decorative images.',
			};
		case "color-contrast":
			return {
				title: "Increase text contrast",
				before: ".subtitle { color: #999; background: #fff; }",
				after: ".subtitle { color: #111; background: #fff; }",
				notes: "Aim for at least 4.5:1 contrast for normal text (WCAG AA).",
			};
		case "html-has-lang":
		case "language-of-page":
		case "html-lang-valid":
		case "valid-lang":
			return {
				title: "Set the page language",
				before: "<html>",
				after: '<html lang="en">',
				notes:
					"Use a valid BCP 47 language tag that matches the primary language of the page.",
			};
		case "label":
			return {
				title: "Associate labels with form fields",
				before: '<input type="email" name="email">',
				after:
					'<label for="email">Email</label>\\n<input id="email" type="email" name="email">',
				notes:
					"Prefer visible labels; use aria-label only when a visible label is not possible.",
			};
		case "link-name":
			return {
				title: "Give links an accessible name",
				before: '<a href="/social"><i class="icon-facebook"></i></a>',
				after:
					'<a href="/social" aria-label="Follow us on Facebook"><i class="icon-facebook"></i></a>',
				notes:
					"Prefer visible link text; fall back to aria-label when the link is icon-only.",
			};
		case "button-name":
			return {
				title: "Give buttons an accessible name",
				before: '<button><svg aria-hidden="true">...</svg></button>',
				after:
					'<button aria-label="Close"><svg aria-hidden="true">...</svg></button>',
				notes: "If the button has visible text, aria-label is not needed.",
			};
		case "focus-visible":
			return {
				title: "Ensure focus is visible",
				before: "button:focus { outline: none; }",
				after:
					"button:focus-visible { outline: 2px solid #0ea5e9; outline-offset: 2px; }",
				notes:
					"Use :focus-visible to avoid distracting focus rings for mouse users.",
			};
		default:
			return null;
	}
}

/*
 * The StageFlow gauge.
 *
 * This was a 20px <span> with a rotated, half-transparent border on ::after —
 * which reads as a ring with a notch, not as a gauge, and looked nothing like
 * the favicon sitting a centimetre above it in the tab strip. Same geometry as
 * public/favicon.svg now, so the mark in the header and the mark in the tab are
 * the same drawing at two sizes.
 *
 * Colors come from tokens rather than the favicon's literals, because the
 * header sits on --surface in light and dark and the mark has to survive both.
 * The needle sweeps once on mount; the global prefers-reduced-motion rule in
 * instrument.css already collapses it, so there is no media query here.
 *
 * aria-hidden throughout: every call site wraps this in a link that already
 * carries the accessible name, and a second one would just be read twice.
 */
export function BrandMark() {
	return (
		<svg className="brandmark" viewBox="0 0 64 64" aria-hidden="true" focusable="false">
			<rect className="brandmark__field" width="64" height="64" rx="18" />
			<path
				className="brandmark__dial"
				d="M16 39a17 17 0 0 1 32 0"
				fill="none"
				strokeWidth="6"
				strokeLinecap="round"
			/>
			<path
				className="brandmark__needle"
				d="m32 37 10-11"
				fill="none"
				strokeWidth="5"
				strokeLinecap="round"
			/>
			<circle className="brandmark__hub" cx="32" cy="39" r="4" />
		</svg>
	);
}

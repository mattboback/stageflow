interface RecordValue {
	[key: string]: unknown;
}

function record(value: unknown): RecordValue | null {
	return typeof value === 'object' && value !== null && !Array.isArray(value)
		? (value as RecordValue)
		: null;
}

function cell(value: unknown): string {
	if (typeof value === 'string') return value.slice(0, 500);
	if (typeof value === 'number' || typeof value === 'boolean') return String(value);
	const object = record(value);
	if (!object) return '—';
	for (const key of ['snippet', 'selector', 'url', 'text', 'value']) {
		const candidate = object[key];
		if (typeof candidate === 'string' || typeof candidate === 'number')
			return String(candidate).slice(0, 500);
	}
	return '—';
}

/** Scanner content stays escaped text; no remote links, raw HTML, or unbounded JSON. */
export function LighthouseDetails({ details }: { details: unknown }) {
	const table = record(details);
	if (
		!table ||
		!['table', 'opportunity'].includes(String(table.type)) ||
		!Array.isArray(table.items)
	)
		return null;
	const rows = table.items
		.slice(0, 30)
		.map(record)
		.filter((row): row is RecordValue => row !== null);
	const headings = Array.isArray(table.headings)
		? table.headings.slice(0, 8).flatMap((value) => {
				const heading = record(value);
				return heading && typeof heading.key === 'string'
					? [{ key: heading.key, label: cell(heading.label ?? heading.key) }]
					: [];
			})
		: [];
	if (headings.length === 0 || rows.length === 0) return null;
	return (
		<section className="evidence__audit">
			<h3 className="imodal__pane-h">Lighthouse evidence</h3>
			<div className="evidence__table-wrap">
				<table>
					<caption>Audit details</caption>
					<thead>
						<tr>
							{headings.map((heading, index) => (
								<th key={index} scope="col">
									{heading.label}
								</th>
							))}
						</tr>
					</thead>
					<tbody>
						{rows.map((row, index) => (
							<tr key={index}>
								{headings.map((heading, column) => (
									<td key={column}>{cell(row[heading.key])}</td>
								))}
							</tr>
						))}
					</tbody>
				</table>
			</div>
			{table.items.length > 30 && <p>Showing 30 of {table.items.length} evidence rows.</p>}
		</section>
	);
}

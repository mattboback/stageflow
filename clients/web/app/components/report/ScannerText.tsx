import type { ReactNode } from 'react';

/*
 * Scanner prose (Lighthouse especially) arrives with markdown fragments:
 * `code spans` and [link text](https://…). Render just those two forms —
 * anything richer stays literal rather than risking a full parser.
 */
export function ScannerText({ text, links = true }: { text: string; links?: boolean }) {
	// Fresh regex per render: a shared /g regex carries lastIndex state.
	const mdToken = /\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)|`([^`]+)`/g;
	const nodes: ReactNode[] = [];
	let last = 0;
	let key = 0;
	for (let m = mdToken.exec(text); m !== null; m = mdToken.exec(text)) {
		if (m.index > last) nodes.push(text.slice(last, m.index));
		if (m[1] && m[2]) {
			/* links=false: contexts (e.g. inside a <button> row) where a nested
			   anchor is invalid — keep the readable text, drop the URL. */
			nodes.push(
				links ? (
					<a key={key++} href={m[2]} target="_blank" rel="noopener noreferrer">
						{m[1]}
					</a>
				) : (
					m[1]
				)
			);
		} else if (m[3]) {
			nodes.push(<code key={key++}>{m[3]}</code>);
		}
		last = m.index + m[0].length;
	}
	if (last < text.length) nodes.push(text.slice(last));
	return <>{nodes}</>;
}

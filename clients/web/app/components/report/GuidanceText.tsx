import { asSentence, failureGroupLabel, parseFailureSummary } from '../../lib/report';
import { ScannerText } from './ScannerText';

/*
 * Scanner guidance as a reader needs it. An axe failure summary ("Fix any of
 * the following:" plus one failed check per line) becomes labelled lists;
 * curated prose stays a paragraph. Both carry the given class, so inline code
 * and links look the same however many blocks the text has.
 */
export function GuidanceText({ text, className }: { text: string; className: string }) {
	const groups = parseFailureSummary(text);
	const only = groups?.length === 1 ? groups[0] : undefined;
	if (!groups || only?.checks.length === 1) {
		return (
			<p className={className}>
				<ScannerText text={only ? asSentence(only.checks[0] ?? '') : text} />
			</p>
		);
	}
	return (
		<div className={`imodal__checks ${className}`}>
			{groups.map((group, index) => (
				<div key={index}>
					<p>{failureGroupLabel(group, index, groups.length)}</p>
					<ul>
						{group.checks.map((check, checkIndex) => (
							<li key={checkIndex}>
								<ScannerText text={check} />
							</li>
						))}
					</ul>
				</div>
			))}
		</div>
	);
}

import {
	penaltyShareByScanner,
	scannerLabel,
	sharePercents,
	type ScoreModel
} from '../../lib/report';
import type { ScannerSummary } from '../../lib/types/unified-report';

interface Props {
	/** From explainScore. Without one (the score could not be reproduced), only the method is described. */
	model: ScoreModel | null | undefined;
	scanners: readonly ScannerSummary[];
	/**
	 * Whether the score is the assessment's, which leaves expected staging
	 * observations out. An older report scores from its summary, which counts them.
	 */
	excludesExpected: boolean;
}

/*
 * The site score next to Lighthouse's own category scores raised the obvious
 * question of why they disagree. This answers it where the number is, and
 * shows which scanners' findings cost the most. The breakdown appears only
 * when it reproduces the shown score exactly (explainScore); a breakdown that
 * did not add up to the number beside it would undermine both.
 */
export function ScoreExplainer({ model, scanners, excludesExpected }: Props) {
	const shares = model ? penaltyShareByScanner(model) : [];
	const percents = sharePercents(shares.map(({ share }) => share));
	return (
		<details className="rhead__how">
			<summary>How is this scored?</summary>
			<div className="rhead__how-body">
				<p>
					The site score starts at 100 and loses points for every actionable finding, weighted by
					severity (critical 20, serious 6, moderate 2, minor 0.5) and averaged over the pages
					scanned. A curve keeps many small findings from outweighing a few severe ones.
				</p>
				<p>
					It counts every scanner, so it differs from Lighthouse’s scores, which grade only
					Lighthouse’s own audits.{' '}
					{excludesExpected
						? 'Expected staging observations and findings that need a human check are left out.'
						: 'Findings that need a human check are left out.'}
				</p>
				{shares.length > 0 && (
					<table className="scorebd">
						<caption>What’s lowering the score</caption>
						<thead>
							<tr>
								<th scope="col">Scanner</th>
								<th scope="col">Share of penalty</th>
								<th scope="col">Score if all fixed</th>
							</tr>
						</thead>
						<tbody>
							{shares.map(({ scanner, share, scoreWithout }, index) => (
								<tr key={scanner}>
									<th scope="row">
										{scannerLabel(scanner, scanners.find((entry) => entry.id === scanner)?.name)}
									</th>
									<td>
										<span className="scorebd__share">
											<span
												className="scorebd__bar"
												style={{ width: `calc(${share} * (100% - 3.5rem))` }}
												aria-hidden="true"
											/>
											<span className="num">{percents[index]}%</span>
										</span>
									</td>
									<td className="num">{scoreWithout}</td>
								</tr>
							))}
						</tbody>
					</table>
				)}
			</div>
		</details>
	);
}

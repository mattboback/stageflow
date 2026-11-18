import type { RuleEvaluation } from '@stageflow/contracts-report';
import { boundRuleEvaluations } from '../../core/rule-evaluations';
import { occurrenceFingerprint } from '../../core/finding-identity';
import { extractAuditNodes } from './result-parser';
import type { LighthouseResult } from './types';

export function lighthouseRuleEvaluations(
	result: LighthouseResult,
	pageId: string
): RuleEvaluation[] {
	return boundRuleEvaluations(
		Object.entries(result.audits ?? {}).map(([ruleId, audit]) => {
			const outcome: RuleEvaluation['outcome'] =
				audit.scoreDisplayMode === 'error'
					? 'error'
					: audit.scoreDisplayMode === 'notApplicable'
						? 'inapplicable'
						: audit.scoreDisplayMode === 'manual' || audit.scoreDisplayMode === 'informative'
							? 'incomplete'
							: audit.score === 1
								? 'pass'
								: typeof audit.score === 'number'
									? 'fail'
									: 'error';
			const subjectIds = extractAuditNodes(audit)
				.slice(0, 32)
				.flatMap((node) =>
					node.selector
						? [occurrenceFingerprint('lighthouse', ruleId, pageId, [node.selector])]
						: []
				);
			return { ruleId, outcome, ...(subjectIds.length ? { subjectIds } : {}) };
		})
	);
}

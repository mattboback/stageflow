import type { RuleEvaluation } from '@stageflow/contracts-report';
import { occurrenceFingerprint } from './finding-identity';

const MAX_RULES = 512;
const MAX_SUBJECTS = 32;
const MAX_PAGE_SUBJECTS = 1024;

/** Only outcomes and bounded occurrence identities survive engine processing. */
export function boundRuleEvaluations(rules: RuleEvaluation[]): RuleEvaluation[] {
	let remaining = MAX_PAGE_SUBJECTS;
	return rules.slice(0, MAX_RULES).map((rule) => {
		const subjectIds = [...new Set(rule.subjectIds ?? [])].slice(
			0,
			Math.min(MAX_SUBJECTS, remaining)
		);
		remaining -= subjectIds.length;
		return {
			ruleId: rule.ruleId,
			outcome: rule.outcome,
			...(subjectIds.length ? { subjectIds } : {})
		};
	});
}

interface EngineRule {
	id: string;
	nodes?: { target?: unknown }[];
}
export function axeRuleEvaluations(
	results: {
		violations: EngineRule[];
		passes: EngineRule[];
		incomplete: EngineRule[];
		inapplicable: EngineRule[];
	},
	pageId: string
): RuleEvaluation[] {
	const groups: [EngineRule[], RuleEvaluation['outcome']][] = [
		[results.violations, 'fail'],
		[results.incomplete, 'incomplete'],
		[results.passes, 'pass'],
		[results.inapplicable, 'inapplicable']
	];
	return boundRuleEvaluations(
		groups.flatMap(([rules, outcome]) =>
			rules.map((rule) => ({
				ruleId: rule.id,
				outcome,
				...((outcome !== 'pass' || process.env.SCAN_PURPOSE === 'verification') &&
				rule.nodes?.length
					? {
							subjectIds: rule.nodes
								.slice(0, MAX_SUBJECTS)
								.map((node) =>
									occurrenceFingerprint(
										'axe',
										rule.id,
										pageId,
										Array.isArray(node.target)
											? node.target.filter((target): target is string => typeof target === 'string')
											: []
									)
								)
						}
					: {})
			}))
		)
	);
}

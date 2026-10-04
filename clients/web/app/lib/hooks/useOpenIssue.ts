import { useCallback } from 'react';
import { useSearchParams } from 'react-router';

/** Opens a finding over the Findings tab of the report on the current route. */
export function useOpenIssue(): (issueId: string) => void {
	const [, setSearchParams] = useSearchParams();
	return useCallback(
		(issueId: string) => {
			setSearchParams(
				(prev) => {
					const next = new URLSearchParams(prev);
					next.set('section', 'issues');
					next.set('issue', issueId);
					return next;
				},
				{ replace: true, preventScrollReset: true }
			);
		},
		[setSearchParams]
	);
}

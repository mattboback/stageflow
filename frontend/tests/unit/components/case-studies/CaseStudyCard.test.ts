import CaseStudyCard from '$lib/components/case-studies/CaseStudyCard.svelte';
import { stageflow } from '$lib/data/case-studies';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('CaseStudyCard', () => {
	it('renders the case study summary and links to the case study page', () => {
		const { getByRole, getByText } = render(CaseStudyCard, {
			props: { study: stageflow }
		});

		expect(getByRole('heading', { name: 'StageFlow', level: 2 })).toBeInTheDocument();
		expect(getByText('Read case study')).toBeInTheDocument();

		const link = getByRole('link');
		expect(link).toHaveAttribute('href', '/projects/stageflow');
	});
});

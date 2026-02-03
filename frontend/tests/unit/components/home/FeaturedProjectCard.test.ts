import FeaturedProjectCard from '$lib/components/home/FeaturedProjectCard.svelte';
import { stageflow } from '$lib/data/case-studies';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('FeaturedProjectCard', () => {
	it('renders core metadata and links to the case study page', () => {
		const { getByRole, getByText } = render(FeaturedProjectCard, {
			props: { study: stageflow }
		});

		expect(getByText('StageFlow')).toBeInTheDocument();
		expect(getByText('Read case study')).toBeInTheDocument();

		const link = getByRole('link');
		expect(link).toHaveAttribute('href', '/projects/stageflow');
	});
});

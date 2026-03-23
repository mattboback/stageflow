import type { SEOCheck } from '../types';

import { CONTENT_CHECKS } from './content';
import { HEADING_CHECKS } from './headings';
import { IMAGE_CHECKS } from './images';
import { META_CHECKS } from './meta';
import { SOCIAL_CHECKS } from './social';
import { TECHNICAL_CHECKS } from './technical';

export const SEO_CHECKS: SEOCheck[] = [
	...META_CHECKS,
	...HEADING_CHECKS,
	...IMAGE_CHECKS,
	...SOCIAL_CHECKS,
	...TECHNICAL_CHECKS,
	...CONTENT_CHECKS
];

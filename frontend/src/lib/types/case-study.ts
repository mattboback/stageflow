export interface CaseStudy {
	meta: {
		name: string;
		slug: string;
		tagline: string;
		status: 'production' | 'beta' | 'development' | 'archived';
		date: string;
		teamSize: string;
		repository: string;
		liveUrl: string | null;
		docsUrl: string | null;
		featured?: boolean;
	};

	hero: {
		description: string;
		badges: string[];
		stats: { value: string; label: string }[];
	};

	problem: {
		headline: string;
		body: string;
	};

	solution: {
		headline: string;
		body: string;
	};

	architecture: {
		summary: string;
		layers: {
			name: string;
			color: 'accent' | 'emerald' | 'blue' | 'rose' | 'amber' | 'slate';
			components: string[];
		}[];
	};

	features: {
		name: string;
		description: string;
		category: 'core' | 'security' | 'performance' | 'dx' | 'integration';
	}[];

	techStack: Record<string, { name: string; purpose: string }[]>;

	decisions: {
		title: string;
		rationale: string;
		alternatives: string[];
	}[];

	outcomes: string[];

	dataFlow: {
		type: 'event-driven' | 'request-response' | 'hybrid' | 'batch';
		messaging: string;
		streams: {
			name: string;
			events: string[];
		}[];
		stateMachine: {
			states: string[];
			happyPath: string;
		};
	};

	api: {
		style: 'REST' | 'GraphQL' | 'gRPC' | 'CLI';
		auth: string;
		keyEndpoints: {
			method: string;
			path: string;
			description: string;
		}[];
	};

	deployment: {
		target: string;
		runtime: string;
		ci: string;
	};

	diagrams?: {
		architecture?: string;
		flow?: string;
		deployment?: string;
	};

	skillsMatrix: string[];

	challenges: {
		problem: string;
		solution: string;
	}[];

	futurework: string[];
}

// Color mapping for architecture layers
export type ArchitectureLayerColor = CaseStudy['architecture']['layers'][number]['color'];

export const layerColors: Record<ArchitectureLayerColor, string> = {
	accent: 'from-accent to-accent-hover',
	emerald: 'from-emerald-500 to-emerald-600',
	blue: 'from-blue-500 to-blue-600',
	rose: 'from-rose-500 to-rose-600',
	amber: 'from-amber-500 to-amber-600',
	slate: 'from-slate-500 to-slate-600'
};

// Stream colors for event visualization
export const streamColors: Record<string, string> = {
	jobs: 'bg-accent',
	extraction: 'bg-ink-faint',
	scan: 'bg-accent-hover'
};

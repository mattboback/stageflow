<script lang="ts">
	import type { ScanResult, ScanStatus, ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import FailedView from '$lib/components/scan-status/FailedView.svelte';
	import ProcessingView from '$lib/components/scan-status/ProcessingView.svelte';
	import { Alert, PageSection, Panel } from '$lib/components/ui';
	import { buildOccurrenceModeReport, isIssueSortKey } from '$lib/report';
	import { AlertTriangle, ArrowLeft, Loader2 } from 'lucide-svelte';

	import ArtifactsView from './ArtifactsView.svelte';
	import ErrorsView from './ErrorsView.svelte';
	import IssueDetailModal from './IssueDetailModal.svelte';
	import IssuesView from './IssuesView.svelte';
	import OverviewDashboard from './OverviewDashboard.svelte';
	import PagesView from './PagesView.svelte';
	import ReportHeader from './ReportHeader.svelte';
	import ReportSectionNav from './ReportSectionNav.svelte';
	import ScannersView from './ScannersView.svelte';
	import VisualReviewPanel from './VisualReviewPanel.svelte';

	interface Props {
		jobId: string;
		status: ScanStatus;
		report: UnifiedReport | null;
		job: ScanResult | null;
		logs: string[];
		screenshots: ScreenshotArtifact[];
		error: string | null;
		onRefreshArtifacts?: () => void;
	}

	let { jobId, status, report, job, logs, screenshots, error, onRefreshArtifacts }: Props =
		$props();

	const section = $derived.by(() => {
		const value = page.url.searchParams.get('section') ?? 'overview';
		// Legacy 'errors' is folded into the combined Diagnostics (artifacts) section.
		if (value === 'errors') return 'artifacts';
		return ['overview', 'issues', 'pages', 'scanners', 'visual', 'artifacts'].includes(value)
			? value
			: 'overview';
	});

	// Strip legacy 'errors' from the URL so deep links remain canonical.
	$effect(() => {
		if (page.url.searchParams.get('section') === 'errors') {
			updateQueryParams({ section: 'artifacts' });
		}
	});

	const activeScanners = $derived.by(() => {
		const value = page.url.searchParams.get('scanner');
		if (!value || value === 'all') return [] as string[];
		return value.split(',').filter(Boolean);
	});

	const activePage = $derived.by(() => {
		const value = page.url.searchParams.get('page');
		return value && value !== 'all' ? value : null;
	});

	const activeSeverities = $derived.by(() => {
		const value = page.url.searchParams.get('severity');
		if (!value || value === 'all') return [] as string[];
		return value.split(',').filter(Boolean);
	});

	const activeCategory = $derived.by(() => {
		const value = page.url.searchParams.get('category');
		return value && value !== 'all' ? value : null;
	});

	const searchTerm = $derived.by(() => page.url.searchParams.get('q') ?? '');
	const activeIssueId = $derived.by(() => page.url.searchParams.get('issueId'));
	const activeElementId = $derived.by(() => page.url.searchParams.get('elementId'));
	const issueSort = $derived.by(() => page.url.searchParams.get('sort') ?? 'severity');
	const displayReport = $derived(report ? buildOccurrenceModeReport(report) : null);

	const activeScanner = $derived(activeScanners.length === 1 ? activeScanners[0] : null);

	const modalIssueList = $derived.by(() => {
		if (!displayReport) return [];
		const filtered = displayReport.issues.filter((issue) => {
			if (activeScanners.length && !activeScanners.includes(issue.scanner)) return false;
			if (activePage && issue.pageId !== activePage) return false;
			if (activeSeverities.length && !activeSeverities.includes(issue.severity)) return false;
			if (activeCategory && issue.category !== activeCategory) return false;
			if (searchTerm) {
				const haystack = `${issue.title} ${issue.description} ${issue.ruleId}`.toLowerCase();
				if (!haystack.includes(searchTerm.toLowerCase())) return false;
			}
			return true;
		});
		return filtered;
	});

	const selectedIssue = $derived.by(() => {
		if (!displayReport || !activeIssueId) return null;
		return displayReport.issues.find((issue) => issue.id === activeIssueId) ?? null;
	});

	function updateQueryParams(
		updates: Record<string, string | null>,
		options: { replaceState?: boolean } = {}
	) {
		const url = new URL(page.url);

		for (const [key, value] of Object.entries(updates)) {
			if (!value || value === 'all') {
				url.searchParams.delete(key);
				continue;
			}
			url.searchParams.set(key, value);
		}

		const query = url.searchParams.toString();
		const target = query ? `${url.pathname}?${query}` : url.pathname;
		void goto(target, {
			replaceState: options.replaceState ?? true,
			noScroll: true,
			keepFocus: true
		});
	}

	function setSection(value: string) {
		updateQueryParams({ section: value === 'overview' ? null : value }, { replaceState: false });
	}

	function handleIssueSelect(issue: IssueDetail, highlightedElementId?: string) {
		updateQueryParams(
			{ issueId: issue.id, elementId: highlightedElementId ?? null },
			{ replaceState: false }
		);
	}

	function closeIssueModal() {
		updateQueryParams({ issueId: null, elementId: null });
	}

	const normalizedIssueSort = $derived(isIssueSortKey(issueSort) ? issueSort : 'severity');
</script>

<PageSection class="pt-12 lg:pt-14" containerClass="max-w-7xl xl:max-w-[1400px]">
	<a
		href={`/scan/${jobId}`}
		class="text-ink-muted hover:text-accent mb-4 inline-flex items-center gap-2 text-sm font-medium transition-colors"
	>
		<ArrowLeft class="h-4 w-4" />
		Back to scan status
	</a>

	{#if displayReport}
		<ReportHeader
			{jobId}
			report={displayReport}
			{job}
			onJumpToIssues={() =>
				updateQueryParams(
					{ section: 'issues', q: null, scanner: null, page: null },
					{ replaceState: false }
				)}
			onJumpToPages={() =>
				updateQueryParams(
					{ section: 'pages', q: null, scanner: null, page: null },
					{ replaceState: false }
				)}
			{...onRefreshArtifacts ? { onRefreshArtifacts } : {}}
		/>
		<ReportSectionNav report={displayReport} {section} onSectionChange={setSection} />

		<svelte:boundary onerror={(e) => console.error('Report section render error:', e)}>
			{#snippet failed(error, reset)}
				{@const errorMessage = error instanceof Error ? error.message : 'Unknown error'}
				<Panel class="shadow-sm" padding="xl" rounded="2xl">
					<Alert variant="error">
						<div class="flex flex-col gap-4">
							<div class="flex items-start gap-3">
								<AlertTriangle class="mt-0.5 h-5 w-5 shrink-0" />
								<div>
									<p class="font-medium">Failed to render report section</p>
									<p class="mt-1 text-sm opacity-80">{errorMessage}</p>
								</div>
							</div>
							<button
								onclick={reset}
								class="bg-accent hover:bg-accent-hover self-start rounded-md px-4 py-2 text-sm font-medium text-white transition-colors"
							>
								Try Again
							</button>
						</div>
					</Alert>
				</Panel>
			{/snippet}
			{#if section === 'overview'}
				<section id="report-panel-overview" aria-labelledby="report-tab-overview">
					<OverviewDashboard
						report={displayReport}
						onSelectPage={(pageId) =>
							updateQueryParams({ section: 'pages', page: pageId }, { replaceState: false })}
						onSelectScanner={(scannerId) =>
							updateQueryParams(
								{ section: 'scanners', scanner: scannerId },
								{ replaceState: false }
							)}
						onSearchIssues={(query, scannerId) =>
							updateQueryParams(
								{
									section: 'issues',
									q: query,
									scanner: scannerId ?? null,
									page: null,
									severity: null,
									category: null
								},
								{ replaceState: false }
							)}
					/>
				</section>
			{:else if section === 'issues'}
				<section id="report-panel-issues" aria-labelledby="report-tab-issues">
					<IssuesView
						report={displayReport}
						{screenshots}
						{activeScanners}
						{activePage}
						{activeSeverities}
						{activeCategory}
						{searchTerm}
						issueSort={normalizedIssueSort}
						selectedIssueId={activeIssueId}
						onScannersChange={(values) =>
							updateQueryParams({ scanner: values.length ? values.join(',') : null })}
						onPageChange={(value) => updateQueryParams({ page: value })}
						onSeveritiesChange={(values) =>
							updateQueryParams({ severity: values.length ? values.join(',') : null })}
						onCategoryChange={(value) => updateQueryParams({ category: value })}
						onSearchChange={(value) => updateQueryParams({ q: value || null })}
						onSortChange={(value) =>
							updateQueryParams({ sort: value === 'severity' ? null : value })}
						onIssueSelect={handleIssueSelect}
						onClearFilters={() =>
							updateQueryParams({
								scanner: null,
								page: null,
								severity: null,
								category: null,
								q: null
							})}
					/>
				</section>
			{:else if section === 'pages'}
				<section id="report-panel-pages" aria-labelledby="report-tab-pages">
					<PagesView
						report={displayReport}
						{screenshots}
						{activeScanner}
						{activePage}
						onSelectPage={(pageId) => updateQueryParams({ page: pageId })}
						onIssueSelect={handleIssueSelect}
					/>
				</section>
			{:else if section === 'scanners'}
				<section id="report-panel-scanners" aria-labelledby="report-tab-scanners">
					<ScannersView
						report={displayReport}
						{job}
						{activeScanner}
						onSelectScanner={(scannerId) => updateQueryParams({ scanner: scannerId })}
					/>
				</section>
			{:else if section === 'visual'}
				<section id="report-panel-visual" aria-labelledby="report-tab-visual">
					<VisualReviewPanel
						report={displayReport}
						{screenshots}
						{activeScanner}
						{activePage}
						onSelectPage={(pageId) => updateQueryParams({ page: pageId })}
						onIssueSelect={handleIssueSelect}
					/>
				</section>
			{:else if section === 'artifacts'}
				<section id="report-panel-artifacts" aria-labelledby="report-tab-artifacts">
					<div class="grid gap-6 lg:grid-cols-2">
						<ArtifactsView {jobId} {job} {...onRefreshArtifacts ? { onRefreshArtifacts } : {}} />
						<ErrorsView errors={displayReport.errors} />
					</div>
				</section>
			{/if}
		</svelte:boundary>
	{:else if status === 'failed' || status === 'error'}
		<Panel class="shadow-sm" padding="xl" rounded="2xl">
			<FailedView result={job} />
		</Panel>
	{:else if error}
		<Alert variant="error">
			<div class="flex items-start gap-3">
				<AlertTriangle class="mt-0.5 h-5 w-5 shrink-0" />
				<div>
					<p class="font-medium">Failed to load report</p>
					<p class="mt-1 text-sm opacity-80">{error}</p>
				</div>
			</div>
		</Alert>
	{:else}
		<Panel class="shadow-sm" padding="xl" rounded="2xl">
			<div class="space-y-6">
				<div class="flex flex-col items-center justify-center gap-4">
					<Loader2 class="text-accent h-10 w-10 animate-spin" />
					<p class="text-ink-muted text-lg">Preparing report…</p>
					{#if status === 'complete'}
						<p class="text-ink-muted text-sm">Scan complete. Generating aggregated report…</p>
					{/if}
				</div>
				{#if status !== 'loading' && status !== 'complete'}
					<ProcessingView result={job} {logs} />
				{/if}
			</div>
		</Panel>
	{/if}
</PageSection>

{#if displayReport && selectedIssue}
	<IssueDetailModal
		issue={selectedIssue}
		page={displayReport.pages.find((p) => p.id === selectedIssue.pageId) ?? null}
		issues={modalIssueList}
		{screenshots}
		{...activeElementId ? { highlightedElementId: activeElementId } : {}}
		onClose={closeIssueModal}
		onNavigate={(issueId) => updateQueryParams({ issueId, elementId: null })}
	/>
{/if}

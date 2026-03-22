<script lang="ts">
import type {
	ScanResult,
	ScanStatus,
	ScreenshotArtifact,
} from "$lib/types/scan";
import type { IssueDetail, UnifiedReport } from "$lib/types/unified-report";

import { goto } from "$app/navigation";
import { page } from "$app/state";
import FailedView from "$lib/components/scan-status/FailedView.svelte";
import ProcessingView from "$lib/components/scan-status/ProcessingView.svelte";
import { Alert, PageSection, Panel } from "$lib/components/ui";
import { buildOccurrenceModeReport, isIssueSortKey } from "$lib/report";
import { parseReportAudience } from "$lib/types/report-audience";
import { AlertTriangle, ArrowLeft, Loader2 } from "lucide-svelte";

import ArtifactsView from "./ArtifactsView.svelte";
import ErrorsView from "./ErrorsView.svelte";
import IssueDetailModal from "./IssueDetailModal.svelte";
import IssuesView from "./IssuesView.svelte";
import OverviewDashboard from "./OverviewDashboard.svelte";
import PagesView from "./PagesView.svelte";
import ReportHeader from "./ReportHeader.svelte";
import ReportSectionNav from "./ReportSectionNav.svelte";
import ScannersView from "./ScannersView.svelte";

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

let {
	jobId,
	status,
	report,
	job,
	logs,
	screenshots,
	error,
	onRefreshArtifacts,
}: Props = $props();

const section = $derived.by(() => {
	const value = page.url.searchParams.get("section") ?? "overview";
	return [
		"overview",
		"issues",
		"pages",
		"scanners",
		"artifacts",
		"errors",
	].includes(value)
		? value
		: "overview";
});

const activeScanner = $derived.by(() => {
	const value = page.url.searchParams.get("scanner");
	return value && value !== "all" ? value : null;
});

const activePage = $derived.by(() => {
	const value = page.url.searchParams.get("page");
	return value && value !== "all" ? value : null;
});

const activeSeverity = $derived.by(() => {
	const value = page.url.searchParams.get("severity");
	return value && value !== "all" ? value : null;
});

const activeCategory = $derived.by(() => {
	const value = page.url.searchParams.get("category");
	return value && value !== "all" ? value : null;
});

const searchTerm = $derived.by(() => page.url.searchParams.get("q") ?? "");
const activeIssueId = $derived.by(() => page.url.searchParams.get("issueId"));
const activeElementId = $derived.by(() =>
	page.url.searchParams.get("elementId"),
);
const issueSort = $derived.by(
	() => page.url.searchParams.get("sort") ?? "severity",
);
const audience = $derived(
	parseReportAudience(page.url.searchParams.get("aud")),
);
const displayReport = $derived(
	report ? buildOccurrenceModeReport(report) : null,
);

const selectedIssue = $derived.by(() => {
	if (!displayReport || !activeIssueId) return null;
	return (
		displayReport.issues.find((issue) => issue.id === activeIssueId) ?? null
	);
});

function updateQueryParams(
	updates: Record<string, string | null>,
	options: { replaceState?: boolean } = {},
) {
	const url = new URL(page.url);

	for (const [key, value] of Object.entries(updates)) {
		if (!value || value === "all") {
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
		keepFocus: true,
	});
}

function setSection(value: string) {
	updateQueryParams(
		{ section: value === "overview" ? null : value },
		{ replaceState: false },
	);
}

function handleIssueSelect(issue: IssueDetail, highlightedElementId?: string) {
	updateQueryParams(
		{ issueId: issue.id, elementId: highlightedElementId ?? null },
		{ replaceState: false },
	);
}

function closeIssueModal() {
	updateQueryParams({ issueId: null, elementId: null });
}

const normalizedIssueSort = $derived(
	isIssueSortKey(issueSort) ? issueSort : "severity",
);
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
			{...(onRefreshArtifacts ? { onRefreshArtifacts } : {})}
		/>
		<ReportSectionNav
			report={displayReport}
			{section}
			{audience}
			onSectionChange={setSection}
			onAudienceChange={(value) => updateQueryParams({ aud: value === 'pm' ? null : value })}
		/>

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
				<OverviewDashboard
					report={displayReport}
					onSelectPage={(pageId) =>
						updateQueryParams({ section: 'pages', page: pageId }, { replaceState: false })}
					onSelectScanner={(scannerId) =>
						updateQueryParams({ section: 'scanners', scanner: scannerId }, { replaceState: false })}
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
			{:else if section === 'issues'}
				<IssuesView
					report={displayReport}
					{screenshots}
					{activeScanner}
					{activePage}
					{activeSeverity}
					{activeCategory}
					{searchTerm}
					issueSort={normalizedIssueSort}
					selectedIssueId={activeIssueId}
					onScannerChange={(value) => updateQueryParams({ scanner: value })}
					onPageChange={(value) => updateQueryParams({ page: value })}
					onSeverityChange={(value) => updateQueryParams({ severity: value })}
					onCategoryChange={(value) => updateQueryParams({ category: value })}
					onSearchChange={(value) => updateQueryParams({ q: value || null })}
					onSortChange={(value) => updateQueryParams({ sort: value === 'severity' ? null : value })}
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
			{:else if section === 'pages'}
				<PagesView
					report={displayReport}
					{screenshots}
					{activeScanner}
					{activePage}
					onSelectPage={(pageId) => updateQueryParams({ page: pageId })}
					onIssueSelect={handleIssueSelect}
				/>
			{:else if section === 'scanners'}
				<ScannersView
					report={displayReport}
					{job}
					{activeScanner}
					onSelectScanner={(scannerId) => updateQueryParams({ scanner: scannerId })}
				/>
			{:else if section === 'artifacts'}
				<ArtifactsView {jobId} {job} {...(onRefreshArtifacts ? { onRefreshArtifacts } : {})} />
			{:else if section === 'errors'}
				<ErrorsView errors={displayReport.errors} />
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
					<p class="text-ink-muted text-lg">Preparing report...</p>
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
			{audience}
			{screenshots}
			{...(activeElementId ? { highlightedElementId: activeElementId } : {})}
			onClose={closeIssueModal}
		/>
	{/if}

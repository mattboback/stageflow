import type { PageSummary, ScannerSummary } from '../../lib/types/unified-report';
import {
	SCANNER_META,
	SEVERITY_LEVELS,
	getSeverityChipClass
} from '../../lib/report';

interface Props {
	scanners: ScannerSummary[];
	pages: PageSummary[];
	categories: string[];
	activeScanner: string | null;
	activeSeverity: string | null;
	activePage: string | null;
	activeCategory: string | null;
	searchQuery: string;
	totalCount: number;
	filteredCount: number;
	onScannerChange: (next: string | null) => void;
	onSeverityChange: (next: string | null) => void;
	onPageChange: (next: string | null) => void;
	onCategoryChange: (next: string | null) => void;
	onSearchChange: (next: string) => void;
	onClear: () => void;
}

export function IssueFilterSidebar({
	scanners,
	pages,
	categories,
	activeScanner,
	activeSeverity,
	activePage,
	activeCategory,
	searchQuery,
	totalCount,
	filteredCount,
	onScannerChange,
	onSeverityChange,
	onPageChange,
	onCategoryChange,
	onSearchChange,
	onClear
}: Props) {
	const hasActiveFilters =
		!!activeScanner ||
		!!activeSeverity ||
		!!activePage ||
		!!activeCategory ||
		searchQuery.trim().length > 0;

	return (
		<aside className="iview__panel isidebar" aria-label="Finding filters">
			<div className="isidebar__group">
				<h3>Search</h3>
				<input
					type="search"
					className="isidebar__search"
					placeholder="Title · rule · page…"
					value={searchQuery}
					onChange={(e) => onSearchChange(e.target.value)}
					aria-label="Search findings"
				/>
			</div>

			<div className="isidebar__group" aria-label="Filter by severity">
				<h3>Severity</h3>
				<div className="isidebar__chips">
					<button
						type="button"
						className={`sev-chip${!activeSeverity ? ' active' : ''}`}
						aria-pressed={!activeSeverity}
						onClick={() => onSeverityChange(null)}
					>
						All
					</button>
					{SEVERITY_LEVELS.map((sev) => {
						const isActive = activeSeverity === sev;
						return (
							<button
								key={sev}
								type="button"
								className={getSeverityChipClass(sev, isActive)}
								aria-pressed={isActive}
								onClick={() => onSeverityChange(isActive ? null : sev)}
							>
								{sev}
							</button>
						);
					})}
				</div>
			</div>

			<div className="isidebar__group">
				<h3>Scanner</h3>
				<select
					className="isidebar__select"
					value={activeScanner ?? ''}
					onChange={(e) => onScannerChange(e.target.value || null)}
					aria-label="Filter by scanner"
				>
					<option value="">All scanners</option>
					{scanners.map((scanner) => {
						const label = SCANNER_META[scanner.id]?.label ?? scanner.name ?? scanner.id;
						return (
							<option key={scanner.id} value={scanner.id}>
								{label} ({scanner.issueCount ?? 0})
							</option>
						);
					})}
				</select>
			</div>

			{pages.length > 1 && (
				<div className="isidebar__group">
					<h3>Page</h3>
					<select
						className="isidebar__select"
						value={activePage ?? ''}
						onChange={(e) => onPageChange(e.target.value || null)}
						aria-label="Filter by page"
					>
						<option value="">All pages</option>
						{pages.map((page) => (
							<option key={page.id} value={page.id}>
								{page.path || page.url}
							</option>
						))}
					</select>
				</div>
			)}

			{categories.length > 0 && (
				<div className="isidebar__group">
					<h3>Category</h3>
					<select
						className="isidebar__select"
						value={activeCategory ?? ''}
						onChange={(e) => onCategoryChange(e.target.value || null)}
						aria-label="Filter by category"
					>
						<option value="">All categories</option>
						{categories.map((cat) => (
							<option key={cat} value={cat}>
								{cat}
							</option>
						))}
					</select>
				</div>
			)}

			<div className="isidebar__group" style={{ marginTop: 'auto' }}>
				<p
					style={{
						margin: '0 0 0.6rem',
						fontSize: '0.84rem',
						color: 'var(--ink-muted)'
					}}
				>
					Showing{' '}
					<b style={{ color: 'var(--ink-strong)', fontVariantNumeric: 'tabular-nums' }}>
						{filteredCount}
					</b>{' '}
					of {totalCount}
				</p>
				{hasActiveFilters && (
					<button type="button" className="isidebar__clear" onClick={onClear}>
						Clear filters
					</button>
				)}
			</div>
		</aside>
	);
}

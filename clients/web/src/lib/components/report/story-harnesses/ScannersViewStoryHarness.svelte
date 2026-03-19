<script lang="ts">
import type { ScanResult } from "$lib/types/scan";
import type { UnifiedReport } from "$lib/types/unified-report";

import ScannersView from "../ScannersView.svelte";

interface Props {
	report: UnifiedReport;
	job: ScanResult | null;
	initialScanner?: string | null;
	onSelectScanner?: (scannerId: string) => void;
}

const { report, job, initialScanner, onSelectScanner }: Props = $props();

let activeScanner = $derived<string | null>(
	initialScanner ?? report.scanners[0]?.id ?? null,
);

function handleSelectScanner(scannerId: string) {
	activeScanner = scannerId;
	onSelectScanner?.(scannerId);
}
</script>

<ScannersView {report} {job} {activeScanner} onSelectScanner={handleSelectScanner} />

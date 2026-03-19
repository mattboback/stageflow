import type { ScannerStatus } from "$lib/types/unified-report";

export type ScannerStatusTone = "success" | "danger" | "muted" | "warning";

export function getScannerStatusTone(
	status?: ScannerStatus | null,
): ScannerStatusTone {
	switch (status) {
		case "success":
			return "success";
		case "failed":
			return "danger";
		case "skipped":
			return "muted";
		default:
			return "warning";
	}
}

export function formatScannerStatus(status?: ScannerStatus | null): string {
	switch (status) {
		case "success":
			return "Success";
		case "failed":
			return "Failed";
		case "skipped":
			return "Skipped";
		default:
			return "Unknown";
	}
}

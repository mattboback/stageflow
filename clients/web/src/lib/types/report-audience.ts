export type ReportAudience = "pm" | "engineer" | "designer";

export function parseReportAudience(
	value: string | null | undefined,
): ReportAudience {
	switch (value) {
		case "pm":
		case "engineer":
		case "designer":
			return value;
		default:
			return "pm";
	}
}

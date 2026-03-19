export function resultsPath(jobId: string, scanner: string): string {
	return `${jobId}/${scanner}/results.json`;
}

export function reportPath(jobId: string, scanner: string): string {
	return `${jobId}/${scanner}/report.html`;
}

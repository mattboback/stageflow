export function resultsPath(jobId: string, scanner: string): string {
  return `${jobId}/${scanner}/results.json`;
}

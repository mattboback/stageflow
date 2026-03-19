// Helper to handle diverse API base URL environments
function getApiBase(): string {
	const rawApiBase =
		(import.meta.env.VITE_API_URL as string | undefined) ??
		"http://localhost:3000";
	return rawApiBase.endsWith("/") ? rawApiBase.slice(0, -1) : rawApiBase;
}

export const buildApiUrl = (path: string) => `${getApiBase()}${path}`;

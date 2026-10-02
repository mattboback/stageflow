/**
 * The variables Chromium may see. The scanner's own environment holds the
 * job's literal form credentials (PROVENANCE_AUTH_JSON) and its scanner
 * options, and a compromised renderer must not be able to read them. Chromium
 * runs with --no-sandbox, so the container is its sandbox.
 */
const BROWSER_ENV_KEYS = ['PATH', 'HOME', 'TMPDIR', 'LANG', 'LC_ALL', 'TZ'] as const;

export function browserEnv(source: NodeJS.ProcessEnv = process.env): Record<string, string> {
	const env: Record<string, string> = { DBUS_SESSION_BUS_ADDRESS: 'disabled' };
	for (const key of BROWSER_ENV_KEYS) {
		const value = source[key];
		if (value !== undefined) {
			env[key] = value;
		}
	}
	return env;
}

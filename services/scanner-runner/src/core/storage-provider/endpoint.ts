export function parseMinioEndpoint(
	endpoint: string,
	useSSL: boolean,
): { endPoint: string; port: number } {
	let host = endpoint.replace(/^https?:\/\//, "");
	let port = useSSL ? 443 : 9000;

	const colonIndex = host.lastIndexOf(":");
	if (colonIndex !== -1) {
		const portStr = host.substring(colonIndex + 1);
		const parsedPort = Number.parseInt(portStr, 10);
		if (!Number.isNaN(parsedPort) && parsedPort > 0 && parsedPort <= 65535) {
			port = parsedPort;
			host = host.substring(0, colonIndex);
		}
	}

	return { endPoint: host, port };
}

export const MAX_LOG_LINES = 50;
export const MAX_SSE_PARSE_ERRORS = 3;

/**
 * Maximum number of times the SSE connection can drop and reconnect
 * before we give up and fall back to a one-time status fetch.
 * EventSource auto-reconnects on error, but if the server is gone
 * (job completed, server restarted, etc.) we don't want to hammer it forever.
 */
export const MAX_SSE_RECONNECT_ATTEMPTS = 5;

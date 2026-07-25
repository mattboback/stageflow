/**
 * URL comparison used by the auth pipeline.
 *
 * `core/auth-hydrator.ts` and `core/auth-wall.ts` each carried a byte-identical
 * private copy of both functions, back to back. They decide whether the browser
 * is still sitting on the login page after a submit, so the two must agree.
 */

/** Drops trailing slashes, preserving root as "/". */
export function trimTrailingSlash(pathname: string): string {
	if (pathname === '/') {
		return pathname;
	}
	return pathname.replace(/\/+$/, '');
}

/**
 * Whether two URLs address the same origin and path, ignoring trailing slashes,
 * query, and fragment.
 *
 * Falls back to string equality when either side is not a parseable absolute URL,
 * which is what `page.url()` returns for about:blank and error pages.
 */
export function sameOriginAndPath(a: string, b: string): boolean {
	try {
		const left = new URL(a);
		const right = new URL(b);
		return (
			left.origin === right.origin &&
			trimTrailingSlash(left.pathname) === trimTrailingSlash(right.pathname)
		);
	} catch {
		return a === b;
	}
}

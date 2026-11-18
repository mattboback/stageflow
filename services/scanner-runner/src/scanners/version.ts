import { readFileSync } from 'node:fs';
import { join } from 'node:path';

// The worker runs under Node directly in production, without npm lifecycle env.
const manifest = JSON.parse(readFileSync(join(__dirname, '../../package.json'), 'utf8')) as {
	version: string;
};
export const SCANNER_VERSION = manifest.version;

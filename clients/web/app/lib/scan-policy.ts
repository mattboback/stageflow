import { IS_HOSTED_DEMO } from './site-metadata';

export const MAX_SCAN_PAGES = IS_HOSTED_DEMO ? 25 : 100;
export const ZIP_UPLOAD_ENABLED = !IS_HOSTED_DEMO;

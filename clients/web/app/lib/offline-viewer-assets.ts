// Vite raw imports put both strings in one hashed, lazily loaded module.
// https://vite.dev/guide/assets.html#importing-asset-as-string
export { default as javascript } from '../../dist/offline-viewer/offline-viewer.js?raw';
export { default as css } from '../../dist/offline-viewer/offline-viewer.css?raw';

import { cp, mkdir, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const projectRoot = path.resolve(scriptDir, "..");

const sourceDir = path.resolve(
	scriptDir,
	"..",
	"..",
	"..",
	"libs",
	"go",
	"scannercatalog",
	"manifests",
);

const destDir = path.resolve(projectRoot, "dist", "scanners");

await stat(sourceDir);
await mkdir(destDir, { recursive: true });

await cp(sourceDir, destDir, {
	recursive: true,
	force: true,
});

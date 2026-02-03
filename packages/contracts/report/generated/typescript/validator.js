"use strict";
/**
 * Runtime validation for UnifiedReportV2
 *
 * Provides type-safe validation using Ajv with full schema and data integrity checks.
 * DO NOT MODIFY - regenerate types with `bun run generate:ts`
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateReport = validateReport;
exports.isValidReport = isValidReport;
exports.assertValidReport = assertValidReport;
exports.parseReport = parseReport;
const ajv_1 = __importDefault(require("ajv"));
const ajv_formats_1 = __importDefault(require("ajv-formats"));
// Import schema - consumers must have ajv installed
const unified_report_v2_schema_json_1 = __importDefault(require("../../schema/unified-report.v2.schema.json"));
// Singleton Ajv instance
let ajvInstance = null;
let compiledValidator = null;
/**
 * Get or create the Ajv validator instance
 */
function getValidator() {
    if (!compiledValidator) {
        ajvInstance = new ajv_1.default({
            strict: false,
            allErrors: true,
            verbose: true,
        });
        (0, ajv_formats_1.default)(ajvInstance);
        compiledValidator = ajvInstance.compile(unified_report_v2_schema_json_1.default);
    }
    return compiledValidator;
}
/**
 * Convert Ajv errors to our ValidationError format
 */
function formatErrors(ajvErrors) {
    if (!ajvErrors)
        return [];
    return ajvErrors.map((err) => ({
        path: err.instancePath || 'root',
        message: err.message || 'Unknown error',
        keyword: err.keyword,
        params: err.params,
    }));
}
/**
 * Check data integrity constraints that can't be expressed in JSON Schema
 */
function checkDataIntegrity(report) {
    const errors = [];
    // Check totalIssues matches issues.length
    if (report.summary.totalIssues !== report.issues.length) {
        errors.push({
            field: 'summary.totalIssues',
            expected: report.issues.length,
            actual: report.summary.totalIssues,
            message: `summary.totalIssues (${report.summary.totalIssues}) does not match issues.length (${report.issues.length})`,
        });
    }
    // Check severity counts match actual issues
    const actualSeverity = {
        critical: 0,
        serious: 0,
        moderate: 0,
        minor: 0,
        info: 0,
    };
    for (const issue of report.issues) {
        actualSeverity[issue.severity]++;
    }
    const reportedSeverity = report.summary.bySeverity;
    for (const severity of ['critical', 'serious', 'moderate', 'minor']) {
        if (reportedSeverity[severity] !== actualSeverity[severity]) {
            errors.push({
                field: `summary.bySeverity.${severity}`,
                expected: actualSeverity[severity],
                actual: reportedSeverity[severity],
                message: `summary.bySeverity.${severity} (${reportedSeverity[severity]}) does not match actual count (${actualSeverity[severity]})`,
            });
        }
    }
    if (reportedSeverity.info !== undefined && reportedSeverity.info !== actualSeverity.info) {
        errors.push({
            field: 'summary.bySeverity.info',
            expected: actualSeverity.info,
            actual: reportedSeverity.info,
            message: `summary.bySeverity.info (${reportedSeverity.info}) does not match actual count (${actualSeverity.info})`,
        });
    }
    // Check pagesScanned matches pages.length
    if (report.summary.pagesScanned !== report.pages.length) {
        errors.push({
            field: 'summary.pagesScanned',
            expected: report.pages.length,
            actual: report.summary.pagesScanned,
            message: `summary.pagesScanned (${report.summary.pagesScanned}) does not match pages.length (${report.pages.length})`,
        });
    }
    // Check scanner IDs referenced in issues exist in scanners array
    const scannerIds = new Set(report.scanners.map((s) => s.id));
    const issueScannersNotFound = new Set();
    for (const issue of report.issues) {
        if (!scannerIds.has(issue.scanner)) {
            issueScannersNotFound.add(issue.scanner);
        }
    }
    if (issueScannersNotFound.size > 0) {
        errors.push({
            field: 'issues[].scanner',
            expected: `one of [${Array.from(scannerIds).join(', ')}]`,
            actual: Array.from(issueScannersNotFound).join(', '),
            message: `Issues reference non-existent scanner IDs: ${Array.from(issueScannersNotFound).join(', ')}`,
        });
    }
    // Check page IDs referenced in issues exist in pages array
    const pageIds = new Set(report.pages.map((p) => p.id));
    const issuePageIdsNotFound = new Set();
    for (const issue of report.issues) {
        if (!pageIds.has(issue.pageId)) {
            issuePageIdsNotFound.add(issue.pageId);
        }
    }
    if (issuePageIdsNotFound.size > 0) {
        errors.push({
            field: 'issues[].pageId',
            expected: `one of [${Array.from(pageIds).join(', ')}]`,
            actual: Array.from(issuePageIdsNotFound).join(', '),
            message: `Issues reference non-existent page IDs: ${Array.from(issuePageIdsNotFound).join(', ')}`,
        });
    }
    // Check artifact IDs referenced in occurrences exist in artifacts array
    if (report.artifacts && report.artifacts.length > 0) {
        const artifactIds = new Set(report.artifacts.map((a) => a.id));
        const missingArtifactIds = new Set();
        for (const issue of report.issues) {
            for (const occ of issue.occurrences || []) {
                for (const artId of occ.artifactIds || []) {
                    if (!artifactIds.has(artId)) {
                        missingArtifactIds.add(artId);
                    }
                }
            }
        }
        if (missingArtifactIds.size > 0) {
            errors.push({
                field: 'issues[].occurrences[].artifactIds',
                expected: `one of [${Array.from(artifactIds).join(', ')}]`,
                actual: Array.from(missingArtifactIds).join(', '),
                message: `Occurrences reference non-existent artifact IDs: ${Array.from(missingArtifactIds).join(', ')}`,
            });
        }
    }
    return errors;
}
/**
 * Validate data against the UnifiedReportV2 schema
 *
 * @param data - Unknown data to validate
 * @param options - Validation options
 * @returns ValidationResult with validity status and any errors
 *
 * @example
 * ```typescript
 * const result = validateReport(jsonData);
 * if (result.valid) {
 *   // data is valid UnifiedReportV2
 * } else {
 *   console.error('Schema errors:', result.errors);
 *   console.error('Integrity errors:', result.integrityErrors);
 * }
 * ```
 */
function validateReport(data, options = { checkIntegrity: true }) {
    const validate = getValidator();
    const schemaValid = validate(data);
    const result = {
        valid: schemaValid,
        errors: formatErrors(validate.errors),
        integrityErrors: [],
    };
    // Only check integrity if schema is valid
    if (schemaValid && options.checkIntegrity !== false) {
        result.integrityErrors = checkDataIntegrity(data);
        if (result.integrityErrors.length > 0) {
            result.valid = false;
        }
    }
    return result;
}
/**
 * Type guard to check if data is a valid UnifiedReportV2
 *
 * @param data - Unknown data to check
 * @returns True if data is valid UnifiedReportV2, with type narrowing
 *
 * @example
 * ```typescript
 * if (isValidReport(data)) {
 *   // TypeScript knows data is UnifiedReportV2 here
 *   console.log(data.version);
 *   console.log(data.issues.length);
 * }
 * ```
 */
function isValidReport(data) {
    return validateReport(data).valid;
}
/**
 * Assert that data is a valid UnifiedReportV2, throwing if invalid
 *
 * @param data - Unknown data to validate
 * @throws Error with validation details if invalid
 * @returns The validated data with proper type
 *
 * @example
 * ```typescript
 * try {
 *   const report = assertValidReport(jsonData);
 *   // report is typed as UnifiedReportV2
 * } catch (e) {
 *   console.error('Invalid report:', e.message);
 * }
 * ```
 */
function assertValidReport(data) {
    const result = validateReport(data);
    if (!result.valid) {
        const allErrors = [
            ...result.errors.map((e) => `${e.path}: ${e.message}`),
            ...result.integrityErrors.map((e) => e.message),
        ];
        throw new Error(`Invalid UnifiedReportV2:\n  ${allErrors.join('\n  ')}`);
    }
    return data;
}
/**
 * Parse and validate JSON string as UnifiedReportV2
 *
 * @param json - JSON string to parse and validate
 * @returns ValidationResult with parsed data if valid
 *
 * @example
 * ```typescript
 * const result = parseReport(jsonString);
 * if (result.valid) {
 *   // Use result.data
 * }
 * ```
 */
function parseReport(json) {
    let data;
    try {
        data = JSON.parse(json);
    }
    catch (e) {
        return {
            valid: false,
            errors: [
                {
                    path: 'root',
                    message: `JSON parse error: ${e instanceof Error ? e.message : 'Unknown error'}`,
                    keyword: 'parse',
                },
            ],
            integrityErrors: [],
        };
    }
    const result = validateReport(data);
    return {
        ...result,
        data: result.valid ? data : undefined,
    };
}

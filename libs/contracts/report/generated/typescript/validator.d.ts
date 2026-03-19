/**
 * Runtime validation for UnifiedReportV2
 *
 * Provides type-safe validation using Ajv with full schema and data integrity checks.
 * DO NOT MODIFY - regenerate types with `bun run generate:ts`
 */
import type { UnifiedReportV2 } from './unified-report.v2';
/**
 * Validation error details
 */
export interface ValidationError {
    path: string;
    message: string;
    keyword: string;
    params?: Record<string, unknown>;
}
/**
 * Data integrity error (business logic validation)
 */
export interface IntegrityError {
    field: string;
    expected: string | number;
    actual: string | number;
    message: string;
}
/**
 * Result of validation
 */
export interface ValidationResult {
    valid: boolean;
    errors: ValidationError[];
    integrityErrors: IntegrityError[];
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
export declare function validateReport(data: unknown, options?: {
    checkIntegrity?: boolean;
}): ValidationResult;
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
export declare function isValidReport(data: unknown): data is UnifiedReportV2;
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
export declare function assertValidReport(data: unknown): UnifiedReportV2;
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
export declare function parseReport(json: string): ValidationResult & {
    data?: UnifiedReportV2;
};
//# sourceMappingURL=validator.d.ts.map

"use strict";
/**
 * @stageflow/contracts-report
 *
 * Generated TypeScript types from JSON Schema.
 * DO NOT MODIFY - regenerate with `bun run generate:ts`
 */
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseReport = exports.assertValidReport = exports.isValidReport = exports.validateReport = void 0;
// Export all generated types
__exportStar(require("./unified-report.v2"), exports);
// Export validation functions
var validator_1 = require("./validator");
Object.defineProperty(exports, "validateReport", { enumerable: true, get: function () { return validator_1.validateReport; } });
Object.defineProperty(exports, "isValidReport", { enumerable: true, get: function () { return validator_1.isValidReport; } });
Object.defineProperty(exports, "assertValidReport", { enumerable: true, get: function () { return validator_1.assertValidReport; } });
Object.defineProperty(exports, "parseReport", { enumerable: true, get: function () { return validator_1.parseReport; } });
// NOTE: Avoid legacy aliases like `ScanResults`/`ScanArtifact` here to keep the
// canonical aggregate report type name unambiguous.

#!/usr/bin/env node
/**
 * Validation script for Unified Report v2 JSON Schema
 *
 * Usage:
 *   node validate.js <path-to-json-file>
 *   bun run validate -- <path-to-json-file>
 */

const Ajv = require('ajv');
const addFormats = require('ajv-formats');
const fs = require('fs');
const path = require('path');

// Load schema
const schemaPath = path.join(__dirname, 'unified-report.v2.schema.json');
const schema = JSON.parse(fs.readFileSync(schemaPath, 'utf8'));

// Create AJV instance with format support
const ajv = new Ajv({
  strict: false,
  allErrors: true,
  verbose: true
});
addFormats(ajv);

// Compile schema
const validate = ajv.compile(schema);

// Get file to validate from command line
const fileToValidate = process.argv[2];

if (!fileToValidate) {
  console.error('Usage: node validate.js <path-to-json-file>');
  process.exit(1);
}

if (!fs.existsSync(fileToValidate)) {
  console.error(`Error: File not found: ${fileToValidate}`);
  process.exit(1);
}

// Load and validate file
const data = JSON.parse(fs.readFileSync(fileToValidate, 'utf8'));
const valid = validate(data);

if (valid) {
  console.log(`✅ ${path.basename(fileToValidate)} is valid`);

  // Run data integrity checks
  const integrityErrors = checkDataIntegrity(data);
  if (integrityErrors.length > 0) {
    console.warn('\n⚠️  Data integrity warnings:');
    integrityErrors.forEach(err => console.warn(`   - ${err}`));
    process.exit(1);
  }

  process.exit(0);
} else {
  console.error(`❌ ${path.basename(fileToValidate)} is invalid\n`);
  console.error('Validation errors:');
  validate.errors.forEach(err => {
    console.error(`  • ${err.instancePath || 'root'}: ${err.message}`);
    if (err.params) {
      console.error(`    Params: ${JSON.stringify(err.params, null, 2)}`);
    }
  });
  process.exit(1);
}

/**
 * Check data integrity constraints that can't be expressed in JSON Schema
 */
function checkDataIntegrity(report) {
  const errors = [];

  // Check totalIssues matches issues.length
  if (report.summary.totalIssues !== report.issues.length) {
    errors.push(
      `summary.totalIssues (${report.summary.totalIssues}) does not match issues.length (${report.issues.length})`
    );
  }

  // Check severity counts match actual issues
  const actualSeverity = {
    critical: 0,
    serious: 0,
    moderate: 0,
    minor: 0,
    info: 0
  };

  report.issues.forEach(issue => {
    actualSeverity[issue.severity]++;
  });

  const reportedSeverity = report.summary.bySeverity;
  for (const severity of ['critical', 'serious', 'moderate', 'minor']) {
    if (reportedSeverity[severity] !== actualSeverity[severity]) {
      errors.push(
        `summary.bySeverity.${severity} (${reportedSeverity[severity]}) does not match actual count (${actualSeverity[severity]})`
      );
    }
  }

  if (reportedSeverity.info !== undefined && reportedSeverity.info !== actualSeverity.info) {
    errors.push(
      `summary.bySeverity.info (${reportedSeverity.info}) does not match actual count (${actualSeverity.info})`
    );
  }

  // Check pagesScanned matches pages.length
  if (report.summary.pagesScanned !== report.pages.length) {
    errors.push(
      `summary.pagesScanned (${report.summary.pagesScanned}) does not match pages.length (${report.pages.length})`
    );
  }

  // Check scanner IDs referenced in issues exist in scanners array
  const scannerIds = new Set(report.scanners.map(s => s.id));
  const issueScannersNotFound = new Set();

  report.issues.forEach(issue => {
    if (!scannerIds.has(issue.scanner)) {
      issueScannersNotFound.add(issue.scanner);
    }
  });

  if (issueScannersNotFound.size > 0) {
    errors.push(
      `Issues reference non-existent scanner IDs: ${Array.from(issueScannersNotFound).join(', ')}`
    );
  }

  // Check page IDs referenced in issues exist in pages array
  const pageIds = new Set(report.pages.map(p => p.id));
  const issuePageIdsNotFound = new Set();

  report.issues.forEach(issue => {
    if (!pageIds.has(issue.pageId)) {
      issuePageIdsNotFound.add(issue.pageId);
    }
  });

  if (issuePageIdsNotFound.size > 0) {
    errors.push(
      `Issues reference non-existent page IDs: ${Array.from(issuePageIdsNotFound).join(', ')}`
    );
  }

  // Check artifact IDs referenced in occurrences exist in artifacts array
  if (report.artifacts && report.artifacts.length > 0) {
    const artifactIds = new Set(report.artifacts.map(a => a.id));
    const missingArtifactIds = new Set();

    report.issues.forEach(issue => {
      (issue.occurrences || []).forEach(occ => {
        (occ.artifactIds || []).forEach(artId => {
          if (!artifactIds.has(artId)) {
            missingArtifactIds.add(artId);
          }
        });
      });
    });

    if (missingArtifactIds.size > 0) {
      errors.push(
        `Occurrences reference non-existent artifact IDs: ${Array.from(missingArtifactIds).join(', ')}`
      );
    }
  }

  return errors;
}

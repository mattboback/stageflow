import Ajv, { type ValidateFunction } from 'ajv';
import addFormats from 'ajv-formats';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

const contractsDir = join(__dirname, '..', '..', '..', '..', 'libs', 'contracts', 'provenance');
const schemaPath = join(contractsDir, 'schema', 'provenance.schema.json');
const fixturesDir = join(contractsDir, 'fixtures');

interface ProvenanceLike {
	version: string;
	job_id: string;
	base_url: string;
	auth?:
		| { mode: 'storage_state'; artifact_key: string }
		| {
				mode: 'form';
				login_url: string;
				steps: { type: string; value?: unknown }[];
				success: { type: string };
		  };
}

function loadSchema(): unknown {
	return JSON.parse(readFileSync(schemaPath, 'utf8')) as unknown;
}

function loadFixture(name: string): ProvenanceLike {
	return JSON.parse(readFileSync(join(fixturesDir, name), 'utf8')) as ProvenanceLike;
}

function makeValidator(): ValidateFunction<ProvenanceLike> {
	const ajv = new Ajv({ strict: false, allErrors: true, verbose: true });
	addFormats(ajv);
	return ajv.compile<ProvenanceLike>(loadSchema());
}

describe('Provenance contract', () => {
	const validate = makeValidator();

	it('accepts a no-auth document', () => {
		const fixture = loadFixture('provenance.no-auth.json');
		expect(validate(fixture)).toBe(true);
		expect(fixture.auth).toBeUndefined();
	});

	it('accepts a storage_state auth document', () => {
		const fixture = loadFixture('provenance.auth-storage-state.json');
		expect(validate(fixture)).toBe(true);
		expect(fixture.auth?.mode).toBe('storage_state');
		expect((fixture.auth as { mode: 'storage_state'; artifact_key: string }).artifact_key).toMatch(
			/storage-state\.json$/
		);
	});

	it('accepts a form auth document with from_env references', () => {
		const fixture = loadFixture('provenance.auth-form.json');
		expect(validate(fixture)).toBe(true);
		expect(fixture.auth?.mode).toBe('form');
		const formAuth = fixture.auth as Extract<typeof fixture.auth, { mode: 'form' }>;
		const fillSteps = formAuth.steps.filter(
			(s): s is typeof s & { value: { from_env: string } } =>
				typeof s.value === 'object' && s.value !== null && 'from_env' in s.value
		);
		expect(fillSteps.length).toBeGreaterThan(0);
		for (const step of fillSteps) {
			expect(step.value.from_env).toMatch(/^[A-Z][A-Z0-9_]*$/);
		}
	});

	it('accepts a form auth document with literal-string values', () => {
		const fixture = loadFixture('provenance.auth-form-literal.json');
		expect(validate(fixture)).toBe(true);
		expect(fixture.auth?.mode).toBe('form');
	});

	it('rejects a from_env reference whose name violates the env-var pattern', () => {
		const bad = {
			...loadFixture('provenance.auth-form.json')
		};
		const formAuth = bad.auth as Extract<typeof bad.auth, { mode: 'form' }>;
		formAuth.steps = formAuth.steps.map((s) =>
			'value' in s && typeof s.value === 'object' && s.value !== null && 'from_env' in s.value
				? { ...s, value: { from_env: 'lowercase-not-allowed' } }
				: s
		);
		expect(validate(bad)).toBe(false);
	});

	it('rejects a form auth missing the success strategy', () => {
		const fixture = loadFixture('provenance.auth-form.json') as Record<string, unknown>;
		const auth = { ...(fixture.auth as Record<string, unknown>) };
		delete auth.success;
		expect(validate({ ...fixture, auth })).toBe(false);
	});

	it('rejects unknown discriminator on auth.mode', () => {
		const fixture = loadFixture('provenance.no-auth.json') as Record<string, unknown>;
		expect(validate({ ...fixture, auth: { mode: 'oauth', token: 'x' } })).toBe(false);
	});
});

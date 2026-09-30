import { useEffect, useId, useRef, useState } from 'react';
import { useNavigate } from 'react-router';
import { PlaygroundAuthConfig } from '../playground/PlaygroundAuthConfig';
import { submitScanJob } from '../../lib/api/client';
import { useScrollLock } from '../../lib/hooks/useScrollLock';
import { cycleTabFocus, focusableWithin } from '../../lib/utils/focus-trap';
import {
	buildFormAuthConfig,
	isAuthConfigComplete,
	validateZipUploadFile,
	type AuthFormConfig
} from '../../lib/playground/playground-utils';
import {
	buildVerificationRequest,
	saveVerificationRequest,
	verificationUnavailableReason
} from '../../lib/verification';
import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import '../../styles/verification.css';

export function VerifyFindingDialog({
	report,
	issue,
	onClose
}: {
	report: UnifiedReport;
	issue: IssueDetail;
	onClose: () => void;
}) {
	const context = report.meta.scanContext;
	const zip = context?.inputType === 'zip';
	const authRequired = Boolean(context?.authenticated);
	const navigate = useNavigate();
	const titleId = useId();
	const panel = useRef<HTMLDivElement>(null);
	const cancel = useRef<HTMLButtonElement>(null);
	const [file, setFile] = useState<File | null>(null);
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [auth, setAuth] = useState<AuthFormConfig>({
		enabled: authRequired,
		loginUrl: '',
		username: '',
		password: '',
		usernameSelector: '',
		passwordSelector: '',
		submitSelector: '',
		successStrategy: 'auto',
		successSelector: ''
	});
	const unavailable = verificationUnavailableReason(report, issue);
	const freshAuthValid = !authRequired || (auth.enabled && isAuthConfigComplete(auth));
	const zipError = zip
		? file
			? validateZipUploadFile(file)
			: 'Upload a fresh ZIP build to verify this change.'
		: null;
	useScrollLock();
	useEffect(() => {
		const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
		cancel.current?.focus();
		return () => previous?.focus();
	}, []);
	useEffect(() => {
		const keys = (event: KeyboardEvent) => {
			if (event.key === 'Escape' && !busy) {
				event.preventDefault();
				onClose();
			}
			if (event.key === 'Tab' && panel.current)
				cycleTabFocus(focusableWithin(panel.current), event);
		};
		window.addEventListener('keydown', keys);
		return () => window.removeEventListener('keydown', keys);
	}, [onClose, busy]);
	async function verify() {
		if (unavailable || zipError || !freshAuthValid || !context) return;
		setBusy(true);
		setError(null);
		try {
			const seed = buildVerificationRequest(report, issue, `pending-${crypto.randomUUID()}`);
			if (!saveVerificationRequest(seed))
				throw new Error(
					'Browser storage is unavailable. Enable local storage before starting verification.'
				);
			const authPayload = authRequired ? buildFormAuthConfig(auth) : null;
			if (authRequired && !authPayload)
				throw new Error('Provide fresh authentication before verifying this protected page.');
			const { job_id } = await submitScanJob({
				mode: zip ? 'zip' : 'url',
				file: zip ? file : null,
				urls: zip ? [] : [issue.pageUrl],
				scanners: [
					{
						id: issue.scanner,
						enabled: true,
						config: context.scannerConfigurations?.[issue.scanner] ?? {}
					}
				],
				profile: report.meta.profile ?? 'production',
				screenshot: true,
				identityScope: context.identityScope,
				...(zip ? { pagePaths: [new URL(issue.pageUrl).pathname] } : {}),
				verification: {
					source_job_id: report.meta.jobId,
					page_id: issue.pageId,
					rule_id: issue.ruleId,
					scanner: issue.scanner,
					...(seed.subjectId ? { subject_id: seed.subjectId } : {})
				},
				auth: authPayload
			});
			saveVerificationRequest({ ...seed, jobId: job_id });
			void navigate(`/scan/${job_id}/report`);
		} catch (cause) {
			setError(cause instanceof Error ? cause.message : 'Verification could not be started.');
			setBusy(false);
		}
	}
	return (
		<div className="cdialog__backdrop" role="dialog" aria-modal="true" aria-labelledby={titleId}>
			<div className="panel verification-dialog" ref={panel}>
				<h2 id={titleId}>Verify this fix</h2>
				<p>
					<strong>{issue.title}</strong>
				</p>
				<p>{issue.pageUrl}</p>
				<p>
					Run {issue.scanner} on this page using the original scan settings. This verifies the
					selected finding; it does not replace a complete audit or baseline.
				</p>
				{unavailable && <p role="alert">{unavailable}</p>}
				{zip && (
					<div>
						<label htmlFor="verification-build">Fresh ZIP build</label>
						<input
							id="verification-build"
							type="file"
							accept=".zip,application/zip"
							onChange={(event) => setFile(event.target.files?.[0] ?? null)}
						/>
						{zipError && <p>{zipError}</p>}
					</div>
				)}
				{authRequired && (
					<>
						<p>
							Enter fresh credentials for this verification. Previous credentials are never saved
							with the report.
						</p>
						<PlaygroundAuthConfig config={auth} isValid={freshAuthValid} onConfigChange={setAuth} />
					</>
				)}
				{error && <p role="alert">{error}</p>}
				<div className="cdialog__actions">
					<button
						className="btn btn--ghost"
						type="button"
						ref={cancel}
						disabled={busy}
						onClick={onClose}
					>
						Cancel
					</button>
					<button
						className="btn btn--primary"
						type="button"
						disabled={busy || Boolean(unavailable || zipError) || !freshAuthValid}
						onClick={() => {
							void verify();
						}}
					>
						{busy ? 'Starting verification…' : 'Run verification'}
					</button>
				</div>
			</div>
		</div>
	);
}

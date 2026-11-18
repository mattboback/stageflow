import { Link } from 'react-router';
import { Pill } from './Pill';

/* Instrument-styled fault readout shared by the 404 page and route error
   boundaries. Styles live in app/styles/fault.css (imported from root.tsx). */
export function RouteFault({
	status,
	title,
	detail,
	traceKey = 'resolver',
	traceLine,
	traceHint,
	actions
}: {
	status: number;
	title: string;
	detail: string;
	traceKey?: string;
	traceLine: string;
	traceHint: string;
	actions?: React.ReactNode;
}) {
	const digits = String(status).split('');
	const accent = Math.floor(digits.length / 2);

	return (
		<main id="main" className="fault">
			<div className="panel fault__panel">
				<div className="fault__top">
					<span className="label">Fault</span>
					<Pill variant="error">{status}</Pill>
				</div>
				<div className="fault__code" aria-hidden="true">
					{digits.map((digit, i) => (i === accent ? <span key={i}>{digit}</span> : digit))}
				</div>
				<div className="fault__body">
					<h1>{title}</h1>
					<p>{detail}</p>
					<div className="fault__trace" role="status">
						<div>
							<span className="k">{traceKey}</span> {traceLine}
						</div>
						<div>
							<span className="warn">→</span> {traceHint}
						</div>
					</div>
					<div className="fault__actions">
						{actions ?? (
							<>
								<Link className="btn btn--primary" to="/">
									Back to home{' '}
									<span className="ar" aria-hidden="true">
										→
									</span>
								</Link>
								<Link className="btn btn--ghost" to="/playground">
									Configure a scan
								</Link>
							</>
						)}
					</div>
				</div>
			</div>
		</main>
	);
}

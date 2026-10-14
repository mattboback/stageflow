import { useState } from 'react';
import { Check, Copy } from 'lucide-react';

export function CopyButton({ value, label }: { value: string; label: string }) {
	const [copied, setCopied] = useState(false);
	return (
		<button
			type="button"
			className="codebox__copy"
			aria-label={label}
			onClick={() => {
				void navigator.clipboard
					?.writeText(value)
					.then(() => {
						setCopied(true);
						window.setTimeout(() => setCopied(false), 1600);
					})
					.catch(() => {
						setCopied(false);
					});
			}}
		>
			{copied ? <Check size={14} aria-hidden="true" /> : <Copy size={14} aria-hidden="true" />}
			{copied ? 'Copied' : 'Copy'}
		</button>
	);
}

/* Stacked label + full-width code box with Copy, shared by Evidence and Occurrence. */
export function EvidenceField({
	label,
	value,
	multiline = false
}: {
	label: string;
	value: string;
	multiline?: boolean;
}) {
	return (
		<div className="evidence__field">
			<dt>{label}</dt>
			<dd>
				<div className="codebox">
					{multiline ? (
						<pre className="code-well">
							<code>{value}</code>
						</pre>
					) : (
						<code className="code-well">{value}</code>
					)}
					<CopyButton value={value} label={`Copy ${label.toLowerCase()}`} />
				</div>
			</dd>
		</div>
	);
}

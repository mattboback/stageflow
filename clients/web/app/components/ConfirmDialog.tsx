import { useEffect, useId, useRef } from 'react';

import { cycleTabFocus, focusableWithin } from '../lib/utils/focus-trap';

/* Design-system replacement for window.confirm: a small modal panel with the
   same focus and Escape behavior as IssueDetailModal — literally the same, via
   the shared focus-trap helper. Cancel receives initial focus so a stray Enter
   never confirms a destructive action. */
export function ConfirmDialog({
	title,
	detail,
	confirmLabel,
	cancelLabel = 'Cancel',
	destructive = false,
	busy = false,
	onConfirm,
	onCancel
}: {
	title: string;
	detail: string;
	confirmLabel: string;
	cancelLabel?: string;
	destructive?: boolean;
	busy?: boolean;
	onConfirm: () => void;
	onCancel: () => void;
}) {
	const titleId = useId();
	const dialogRef = useRef<HTMLDivElement>(null);
	const cancelRef = useRef<HTMLButtonElement>(null);

	useEffect(() => {
		const previousActiveElement = document.activeElement as HTMLElement | null;
		cancelRef.current?.focus();
		return () => {
			if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
				previousActiveElement.focus();
			}
		};
	}, []);

	useEffect(() => {
		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				onCancel();
				return;
			}
			if (event.key === 'Tab' && dialogRef.current) {
				cycleTabFocus(focusableWithin(dialogRef.current), event);
			}
		};
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	}, [onCancel]);

	return (
		<div
			className="cdialog__backdrop"
			role="dialog"
			aria-modal="true"
			aria-labelledby={titleId}
			onClick={onCancel}
		>
			<div className="panel cdialog" ref={dialogRef} onClick={(event) => event.stopPropagation()}>
				<div className="cdialog__body">
					<h2 id={titleId}>{title}</h2>
					<p>{detail}</p>
				</div>
				<div className="cdialog__actions">
					<button className="btn btn--ghost" type="button" ref={cancelRef} onClick={onCancel}>
						{cancelLabel}
					</button>
					<button
						className={destructive ? 'btn btn--danger' : 'btn btn--primary'}
						type="button"
						disabled={busy}
						onClick={onConfirm}
					>
						{confirmLabel}
					</button>
				</div>
			</div>
		</div>
	);
}

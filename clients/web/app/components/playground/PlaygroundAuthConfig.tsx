import { useState } from 'react';

import type { AuthFormConfig } from '../../lib/components/playground/playground-utils';

interface Props {
	config: AuthFormConfig;
	isValid: boolean;
	onConfigChange: (config: AuthFormConfig) => void;
}

export function PlaygroundAuthConfig({ config, isValid, onConfigChange }: Props) {
	const [showPassword, setShowPassword] = useState(false);
	const [showAdvanced, setShowAdvanced] = useState(false);

	const update = (patch: Partial<AuthFormConfig>) => {
		onConfigChange({ ...config, ...patch });
	};

	return (
		<section className="pauth">
			<header className="pauth__head">
				<div>
					<h2 className="pauth__title">
						<span className="stepno num" aria-hidden="true">
							3
						</span>
						Authentication
					</h2>
					<p className="pauth__sub">
						Log in before scanning protected pages — the scan browser replays a form
						login with the credentials you provide.
					</p>
				</div>
				{/* Auth is a configuration step, not an on/off preference: show its
				    state and the action that moves it forward. */}
				{!config.enabled ? (
					<div className="pauth__state">
						<span className="pauth__status">Not configured</span>
						<button
							type="button"
							className="btn btn--ghost btn--sm"
							onClick={() => update({ enabled: true })}
						>
							Set up{' '}
							<span className="ar" aria-hidden="true">
								→
							</span>
						</button>
					</div>
				) : (
					<div className="pauth__state">
						{isValid ? (
							<span className="pauth__ok" role="status">
								Configured
							</span>
						) : (
							<span className="pauth__warn" role="status" id="pauth-incomplete">
								Required fields missing
							</span>
						)}
						<button
							type="button"
							className="pauth__remove"
							onClick={() => update({ enabled: false })}
						>
							Remove
						</button>
					</div>
				)}
			</header>

			{config.enabled && (
				<div className="pauth__body">
					<label className="pauth__field">
						<span className="label">
							Login URL <span className="pauth__req">*</span>
						</span>
						<input
							type="url"
							autoComplete="off"
							placeholder="https://app.example.com/login"
							value={config.loginUrl}
							aria-invalid={config.enabled && !config.loginUrl.trim() ? true : undefined}
							aria-required="true"
							onChange={(e) => update({ loginUrl: e.target.value })}
						/>
					</label>
					<div className="pauth__row">
						<label className="pauth__field">
							<span className="label">
								Username / email <span className="pauth__req">*</span>
							</span>
							<input
								type="text"
								autoComplete="off"
								value={config.username}
								aria-invalid={config.enabled && !config.username.trim() ? true : undefined}
								aria-required="true"
								onChange={(e) => update({ username: e.target.value })}
							/>
						</label>
						<label className="pauth__field">
							<span className="label">
								Password <span className="pauth__req">*</span>
							</span>
							<div className="pauth__pwd">
								<input
									type={showPassword ? 'text' : 'password'}
									autoComplete="off"
									value={config.password}
									aria-invalid={config.enabled && !config.password.trim() ? true : undefined}
									aria-required="true"
									onChange={(e) => update({ password: e.target.value })}
								/>
								<button
									type="button"
									onClick={() => setShowPassword((v) => !v)}
									className="pauth__pwd-eye"
									aria-label={showPassword ? 'Hide password' : 'Show password'}
								>
									{showPassword ? '◉' : '◌'}
								</button>
							</div>
						</label>
					</div>

					<button
						type="button"
						className="pauth__adv-toggle"
						onClick={() => setShowAdvanced((v) => !v)}
					>
						{showAdvanced ? '▾' : '▸'} Advanced selectors
					</button>
					{showAdvanced && (
						<div className="pauth__adv">
							<label className="pauth__field">
								<span className="label">Username selector</span>
								<input
									type="text"
									placeholder="auto:username"
									value={config.usernameSelector}
									onChange={(e) => update({ usernameSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="label">Password selector</span>
								<input
									type="text"
									placeholder="auto:password"
									value={config.passwordSelector}
									onChange={(e) => update({ passwordSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="label">Submit selector</span>
								<input
									type="text"
									placeholder="auto:submit"
									value={config.submitSelector}
									onChange={(e) => update({ submitSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="label">Success strategy</span>
								<select
									value={config.successStrategy}
									onChange={(e) =>
										update({
											successStrategy: e.target.value as 'auto' | 'selector'
										})
									}
								>
									<option value="auto">Auto (network idle)</option>
									<option value="selector">Wait for selector</option>
								</select>
							</label>
							{config.successStrategy === 'selector' && (
								<label className="pauth__field">
									<span className="label">Success selector</span>
									<input
										type="text"
										placeholder=".dashboard, [data-logged-in]"
										value={config.successSelector}
										onChange={(e) => update({ successSelector: e.target.value })}
									/>
								</label>
							)}
						</div>
					)}
				</div>
			)}
		</section>
	);
}

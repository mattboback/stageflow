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
					<h3 className="pauth__title">Authentication</h3>
					<p className="pauth__sub">Log in before scanning protected pages.</p>
				</div>
				{config.enabled && !isValid && (
					<span className="pauth__warn">Required fields missing</span>
				)}
				<label className="pauth__toggle">
					<input
						type="checkbox"
						checked={config.enabled}
						onChange={(e) => update({ enabled: e.target.checked })}
					/>
					<span className="pauth__toggle-track" aria-hidden="true" />
					<span className="sr-only">Enable authentication</span>
				</label>
			</header>

			{config.enabled && (
				<div className="pauth__body">
					<label className="pauth__field">
						<span className="pauth__label">
							Login URL <span className="pauth__req">*</span>
						</span>
						<input
							type="url"
							autoComplete="off"
							placeholder="https://app.example.com/login"
							value={config.loginUrl}
							onChange={(e) => update({ loginUrl: e.target.value })}
						/>
					</label>
					<div className="pauth__row">
						<label className="pauth__field">
							<span className="pauth__label">
								Username / email <span className="pauth__req">*</span>
							</span>
							<input
								type="text"
								autoComplete="off"
								value={config.username}
								onChange={(e) => update({ username: e.target.value })}
							/>
						</label>
						<label className="pauth__field">
							<span className="pauth__label">
								Password <span className="pauth__req">*</span>
							</span>
							<div className="pauth__pwd">
								<input
									type={showPassword ? 'text' : 'password'}
									autoComplete="off"
									value={config.password}
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
								<span className="pauth__label">Username selector</span>
								<input
									type="text"
									placeholder="auto:username"
									value={config.usernameSelector}
									onChange={(e) => update({ usernameSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="pauth__label">Password selector</span>
								<input
									type="text"
									placeholder="auto:password"
									value={config.passwordSelector}
									onChange={(e) => update({ passwordSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="pauth__label">Submit selector</span>
								<input
									type="text"
									placeholder="auto:submit"
									value={config.submitSelector}
									onChange={(e) => update({ submitSelector: e.target.value })}
								/>
							</label>
							<label className="pauth__field">
								<span className="pauth__label">Success strategy</span>
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
									<span className="pauth__label">Success selector</span>
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

<script lang="ts">
	import type { AuthFormConfig } from '$lib/components/playground/playground-utils';

	import { Label, Chip } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { AlertTriangle, ChevronDown, ChevronUp, Eye, EyeOff, Lock } from 'lucide-svelte';

	interface Props {
		config: AuthFormConfig;
		/** Passed in from parent derived state so the parent controls validity logic. */
		isValid: boolean;
		onConfigChange: (config: AuthFormConfig) => void;
	}

	let { config, isValid, onConfigChange }: Props = $props();

	let showAdvancedSettings = $state(false);
	let showPassword = $state(false);

	function update(patch: Partial<AuthFormConfig>) {
		onConfigChange({ ...config, ...patch });
	}
</script>

<div
	class="animate-fade-in border-line/80 bg-surface-muted/55 space-y-5 rounded-2xl border p-6 shadow-[var(--shadow-xs)]"
>
	<!-- Header -->
	<div class="mb-4 flex items-center gap-3">
		<div class="bg-accent/10 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl">
			<Lock class="text-accent h-5 w-5" />
		</div>
		<div class="min-w-0 flex-1">
			<h3 class="text-ink text-sm font-semibold">Authentication</h3>
			<p class="text-ink-muted text-xs">Log in before scanning protected pages</p>
		</div>
		{#if config.enabled && !isValid}
			<Chip tone="warning" size="xs" class="gap-1 font-medium">
				<AlertTriangle class="h-3 w-3" />
				Required
			</Chip>
		{/if}
		<!-- Toggle switch -->
		<button
			type="button"
			role="switch"
			aria-checked={config.enabled}
			aria-label="Enable authentication"
			onclick={() => update({ enabled: !config.enabled })}
			class={cn(
				'relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 transition-colors duration-200 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
				config.enabled
					? 'border-accent bg-accent focus-visible:ring-accent'
					: 'border-line bg-surface-muted focus-visible:ring-accent'
			)}
		>
			<span
				class={cn(
					'pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform duration-200',
					config.enabled ? 'translate-x-5' : 'translate-x-0.5'
				)}
			></span>
		</button>
	</div>

	{#if config.enabled}
		<!-- Login URL -->
		<div>
			<Label for="auth-login-url" class="mb-2 flex items-center gap-1 text-sm font-semibold">
				Login URL
				<span class="text-accent">*</span>
			</Label>
			<input
				id="auth-login-url"
				type="url"
				autocomplete="off"
				placeholder="https://app.example.com/login"
				value={config.loginUrl}
				oninput={(e) => update({ loginUrl: e.currentTarget.value })}
				class={cn(
					'border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2.5 text-sm focus:outline-none',
					!config.loginUrl.trim() && 'border-amber-300 focus:border-amber-500'
				)}
			/>
		</div>

		<!-- Username / Email -->
		<div>
			<Label for="auth-username" class="mb-2 flex items-center gap-1 text-sm font-semibold">
				Username / Email
				<span class="text-accent">*</span>
			</Label>
			<input
				id="auth-username"
				type="text"
				autocomplete="off"
				placeholder="user@example.com"
				value={config.username}
				oninput={(e) => update({ username: e.currentTarget.value })}
				class={cn(
					'border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2.5 text-sm focus:outline-none',
					!config.username.trim() && 'border-amber-300 focus:border-amber-500'
				)}
			/>
		</div>

		<!-- Password -->
		<div>
			<Label for="auth-password" class="mb-2 flex items-center gap-1 text-sm font-semibold">
				Password
				<span class="text-accent">*</span>
			</Label>
			<div class="relative">
				<input
					id="auth-password"
					type={showPassword ? 'text' : 'password'}
					autocomplete="off"
					placeholder="••••••••"
					value={config.password}
					oninput={(e) => update({ password: e.currentTarget.value })}
					class={cn(
						'border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2.5 pr-10 text-sm focus:outline-none',
						!config.password && 'border-amber-300 focus:border-amber-500'
					)}
				/>
				<button
					type="button"
					onclick={() => (showPassword = !showPassword)}
					aria-label={showPassword ? 'Hide password' : 'Show password'}
					class="text-ink-muted hover:text-ink absolute top-1/2 right-3 -translate-y-1/2"
				>
					{#if showPassword}
						<EyeOff class="h-4 w-4" />
					{:else}
						<Eye class="h-4 w-4" />
					{/if}
				</button>
			</div>
			<p class="text-ink-faint mt-1.5 text-xs">Use a demo or scoped account for hosted scans.</p>
		</div>

		<!-- Advanced Settings Toggle -->
		<button
			type="button"
			onclick={() => (showAdvancedSettings = !showAdvancedSettings)}
			class="border-line bg-surface hover:bg-surface-muted flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition-colors"
			aria-expanded={showAdvancedSettings}
			aria-controls="auth-advanced-settings"
		>
			<span class="text-ink text-sm font-medium">Advanced Settings</span>
			{#if showAdvancedSettings}
				<ChevronUp class="text-ink-muted h-4 w-4" />
			{:else}
				<ChevronDown class="text-ink-muted h-4 w-4" />
			{/if}
		</button>

		{#if showAdvancedSettings}
			<div id="auth-advanced-settings" class="animate-fade-in space-y-4 pt-2">
				<p class="text-ink-muted text-xs">
					Override CSS selectors if the defaults don't match your login form.
				</p>

				<div class="grid gap-4 sm:grid-cols-2">
					<!-- Username selector -->
					<div>
						<Label for="auth-username-selector" class="mb-2 block text-sm font-semibold">
							Username Field Selector
						</Label>
						<input
							id="auth-username-selector"
							type="text"
							autocomplete="off"
							placeholder={'input[type="email"]'}
							value={config.usernameSelector}
							oninput={(e) => update({ usernameSelector: e.currentTarget.value })}
							class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2 font-mono text-xs focus:outline-none"
						/>
					</div>

					<!-- Password selector -->
					<div>
						<Label for="auth-password-selector" class="mb-2 block text-sm font-semibold">
							Password Field Selector
						</Label>
						<input
							id="auth-password-selector"
							type="text"
							autocomplete="off"
							placeholder={'input[type="password"]'}
							value={config.passwordSelector}
							oninput={(e) => update({ passwordSelector: e.currentTarget.value })}
							class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2 font-mono text-xs focus:outline-none"
						/>
					</div>
				</div>

				<!-- Submit selector -->
				<div>
					<Label for="auth-submit-selector" class="mb-2 block text-sm font-semibold">
						Submit Button Selector
					</Label>
					<input
						id="auth-submit-selector"
						type="text"
						autocomplete="off"
						placeholder={'button[type="submit"]'}
						value={config.submitSelector}
						oninput={(e) => update({ submitSelector: e.currentTarget.value })}
						class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2 font-mono text-xs focus:outline-none"
					/>
				</div>

				<div class="animate-fade-in">
					<Label
						for="auth-success-selector"
						class="mb-2 flex items-center gap-1 text-sm font-semibold"
					>
						Success Selector
						<span class="text-accent">*</span>
					</Label>
					<input
						id="auth-success-selector"
						type="text"
						autocomplete="off"
						placeholder={'a[href="/dashboard"], [data-testid="welcome"]'}
						value={config.successSelector}
						oninput={(e) => update({ successSelector: e.currentTarget.value })}
						class={cn(
							'border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent w-full rounded-xl border px-3 py-2 font-mono text-xs focus:outline-none',
							!config.successSelector.trim() && 'border-amber-300 focus:border-amber-500'
						)}
					/>
					<p class="text-ink-faint mt-1 text-xs">
						Scan proceeds when this element becomes visible after login.
					</p>
				</div>
			</div>
		{/if}
	{/if}
</div>

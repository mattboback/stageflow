import { Monitor, Moon, Sun } from 'lucide-react';

import { useTheme } from '../lib/hooks/useTheme';
import { THEME_LABELS, THEMES, type Theme } from '../lib/theme';

const ICONS: Record<Theme, typeof Sun> = {
	light: Sun,
	dark: Moon,
	system: Monitor
};

/*
 * A native radio group, not the .toggle switch.
 *
 * Light / dark / system is three states, and aria-checked cannot say that
 * honestly. The existing .toggle is also a bare <button> with no name
 * mechanism, so using it here would trip axe's aria-toggle-field-name at
 * serious impact on every surface the header appears on. Radios give the
 * accessible name from <legend>, roving arrow-key focus, and correct group
 * semantics with no ARIA at all.
 */
export function ThemeToggle() {
	const { theme, setTheme } = useTheme();

	return (
		<fieldset className="themepick">
			<legend className="sr-only">Color theme</legend>
			{THEMES.map((option) => {
				const Icon = ICONS[option];
				return (
					<label className="themepick__opt" key={option}>
						<input
							className="themepick__input"
							type="radio"
							name="theme"
							value={option}
							checked={theme === option}
							onChange={() => {
								setTheme(option);
							}}
						/>
						<Icon aria-hidden="true" size={15} />
						<span className="themepick__label">{THEME_LABELS[option]}</span>
					</label>
				);
			})}
		</fieldset>
	);
}

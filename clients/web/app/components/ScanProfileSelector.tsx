import './ScanProfileSelector.css';

export function ScanProfileSelector({
	profile,
	onChange,
	id = 'scan-profile'
}: {
	profile: 'production' | 'staging';
	onChange: (profile: 'production' | 'staging') => void;
	id?: string;
}) {
	return (
		<div className="scan-profile">
			<label htmlFor={id}>Scan profile</label>
			<select
				id={id}
				className="input"
				value={profile}
				onChange={(event) => onChange(event.target.value === 'staging' ? 'staging' : 'production')}
			>
				<option value="production">Production</option>
				<option value="staging">Staging</option>
			</select>
			<p>
				{profile === 'staging'
					? 'Pages that block search indexing are expected here and not counted. Every other check still counts.'
					: 'Pages that block search indexing are reported as problems.'}
			</p>
		</div>
	);
}

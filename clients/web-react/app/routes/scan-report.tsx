import { useParams } from 'react-router';

// Phase 1 placeholder. Report route ported onto the Instrument system in Phase 4.
export default function ScanReport() {
	const { id } = useParams();
	return (
		<main style={{ padding: '4rem 1.5rem', maxWidth: '40rem', margin: '0 auto' }}>
			<h1>Report {id}</h1>
		</main>
	);
}

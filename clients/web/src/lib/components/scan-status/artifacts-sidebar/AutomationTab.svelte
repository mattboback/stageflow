<script lang="ts">
	import { buildApiUrl } from '$lib/api/utils';
	import { buildSiteUrl } from '$lib/config/site';

	interface Props {
		jobId: string | null;
	}

	let { jobId }: Props = $props();

	const snippet = $derived(
		jobId
			? `curl -s ${buildApiUrl(`/api/v1/jobs/${jobId}`)} | jq '.state, .violations'`
			: `suite-runner -suite suite.yml -api ${buildSiteUrl('/')}`
	);
	const artifactSnippet = $derived(
		jobId
			? `npx playwright test --config=scan.config.ts --job ${jobId}`
			: '# Upload a suite config, then trigger StageFlow via API'
	);

	let copiedSnippet = $state<string | null>(null);

	async function copySnippet(code: string, label: string) {
		try {
			await navigator.clipboard?.writeText(code);
			copiedSnippet = label;
			setTimeout(() => {
				copiedSnippet = null;
			}, 2000);
		} catch {
			copiedSnippet = null;
		}
	}
</script>

<div class="space-y-3">
	<div class="rounded-lg border border-slate-800 bg-slate-950/90 p-3 text-xs text-slate-100">
		<div
			class="mb-2 flex items-center justify-between text-[11px] tracking-wide text-slate-400 uppercase"
		>
			<span>Fetch latest job JSON</span>
			<button
				type="button"
				onclick={() => copySnippet(snippet, 'snippet1')}
				class="text-[11px] font-semibold text-slate-200 hover:text-white"
			>
				{copiedSnippet === 'snippet1' ? 'Copied' : 'Copy'}
			</button>
		</div>
		<pre class="font-mono text-[12px] leading-relaxed break-all whitespace-pre-wrap">{snippet}</pre>
	</div>

	<div class="rounded-lg border border-slate-800 bg-slate-950/90 p-3 text-xs text-slate-100">
		<div
			class="mb-2 flex items-center justify-between text-[11px] tracking-wide text-slate-400 uppercase"
		>
			<span>Pull artifacts in CI</span>
			<button
				type="button"
				onclick={() => copySnippet(artifactSnippet, 'snippet2')}
				class="text-[11px] font-semibold text-slate-200 hover:text-white"
			>
				{copiedSnippet === 'snippet2' ? 'Copied' : 'Copy'}
			</button>
		</div>
		<pre class="font-mono text-[12px] leading-relaxed break-all whitespace-pre-wrap">
			{artifactSnippet}
		</pre>
	</div>
</div>

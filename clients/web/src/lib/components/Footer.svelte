<script lang="ts">
	import { GithubIcon, LogoMark } from '$lib/components/icons';
	import { SITE } from '$lib/config/site';

	const year = new Date().getFullYear();

	const footerLinks = [
		{
			category: 'Product',
			links: [
				{ label: 'Playground', href: '/playground', external: false },
				{ label: 'Scan Report Demo', href: '/scan/demo', external: false }
			]
		},
		{
			category: 'Resources',
			links: [
				{ label: 'GitHub', href: SITE.githubUrl, external: true },
				{
					label: 'Documentation',
					href: `${SITE.githubUrl}/blob/main/README.md`,
					external: true
				}
			]
		}
	] as const;
</script>

<footer class="footer-shell border-line border-t">
	<div class="container-width py-12">
		<div class="grid gap-10 md:grid-cols-[minmax(0,1.4fr)_auto_auto]">
			<!-- Brand -->
			<div class="max-w-xs">
				<a href="/" class="inline-flex items-center gap-2.5">
					<LogoMark class="text-accent" size={20} />
					<span class="font-display text-paper text-[18px] leading-tight font-semibold"
						>{SITE.name}</span
					>
				</a>
				<p class="mt-3 text-sm leading-relaxed" style="color: rgba(250,249,247,0.5)">
					Self-hosted web scanning for accessibility, performance, SEO, and security workflows.
				</p>
				<div class="mt-5 flex items-center gap-3">
					<a
						href={SITE.githubUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="footer-link flex items-center gap-2"
						aria-label="GitHub"
					>
						<GithubIcon class="h-4 w-4" />
						<span>Open source</span>
					</a>
				</div>
			</div>

			<!-- Link columns -->
			{#each footerLinks as col (col.category)}
				<div>
					<p class="footer-category">{col.category}</p>
					<ul class="space-y-2.5">
						{#each col.links as link (link.label)}
							<li>
								{#if link.external}
									<a
										href={link.href}
										class="footer-link"
										target="_blank"
										rel="noopener noreferrer"
									>
										{link.label}
									</a>
								{:else}
									<a href={link.href} class="footer-link">
										{link.label}
									</a>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>

		<div class="mt-10 border-t pt-6" style="border-color: rgba(250,249,247,0.1)">
			<p class="text-xs" style="color: rgba(250,249,247,0.3)">
				&copy; {year} StageFlow. Open-source, operator-first, no vendor lock-in.
			</p>
		</div>
	</div>
</footer>

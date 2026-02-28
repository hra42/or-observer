<script lang="ts">
	import { page } from '$app/state';

	const links = [
		{ href: '/dashboard', label: 'Dashboard' },
		{ href: '/traces', label: 'Traces' },
		{ href: '/analytics', label: 'Analytics' }
	];
</script>

<nav class="border-b border-gray-800 bg-gray-900 px-4">
	<div class="mx-auto flex max-w-7xl items-center gap-6 py-3">
		<span class="text-lg font-semibold text-indigo-400">or-observer</span>
		{#each links as link}
			<a
				href={link.href}
				class="text-sm transition-colors {page.url.pathname.startsWith(link.href)
					? 'font-medium text-white'
					: 'text-gray-400 hover:text-white'}"
			>
				{link.label}
			</a>
		{/each}
	</div>
</nav>

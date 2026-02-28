<script lang="ts">
	import { page } from '$app/state';
	import { isDark, toggleTheme } from '$lib/stores/theme.svelte';

	const links = [
		{ href: '/dashboard', label: 'Dashboard' },
		{ href: '/traces', label: 'Traces' },
		{ href: '/analytics', label: 'Analytics' },
		{ href: '/alerts', label: 'Alerts' }
	];

	let menuOpen = $state(false);
</script>

<nav class="border-b border-gray-200 bg-gray-50 px-4 dark:border-gray-800 dark:bg-gray-900">
	<div class="mx-auto flex max-w-7xl items-center justify-between py-3">
		<div class="flex items-center gap-6">
			<span class="text-lg font-semibold text-indigo-600 dark:text-indigo-400">or-observer</span>
			<!-- Desktop links -->
			<div class="hidden sm:flex sm:gap-4">
				{#each links as link}
					<a
						href={link.href}
						class="text-sm transition-colors {page.url.pathname.startsWith(link.href)
							? 'font-medium text-gray-900 dark:text-white'
							: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
					>
						{link.label}
					</a>
				{/each}
			</div>
		</div>

		<div class="flex items-center gap-2">
			<!-- Theme toggle -->
			<button
				onclick={toggleTheme}
				class="rounded p-1.5 text-gray-600 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-800"
				aria-label="Toggle theme"
			>
				{#if isDark()}
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 2a1 1 0 011 1v1a1 1 0 11-2 0V3a1 1 0 011-1zm4 8a4 4 0 11-8 0 4 4 0 018 0zm-.464 4.95l.707.707a1 1 0 001.414-1.414l-.707-.707a1 1 0 00-1.414 1.414zm2.12-10.607a1 1 0 010 1.414l-.706.707a1 1 0 11-1.414-1.414l.707-.707a1 1 0 011.414 0zM17 11a1 1 0 100-2h-1a1 1 0 100 2h1zm-7 4a1 1 0 011 1v1a1 1 0 11-2 0v-1a1 1 0 011-1zM5.05 6.464A1 1 0 106.465 5.05l-.708-.707a1 1 0 00-1.414 1.414l.707.707zm1.414 8.486l-.707.707a1 1 0 01-1.414-1.414l.707-.707a1 1 0 011.414 1.414zM4 11a1 1 0 100-2H3a1 1 0 000 2h1z" clip-rule="evenodd" />
					</svg>
				{:else}
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
						<path d="M17.293 13.293A8 8 0 016.707 2.707a8.001 8.001 0 1010.586 10.586z" />
					</svg>
				{/if}
			</button>

			<!-- Mobile hamburger -->
			<button
				onclick={() => (menuOpen = !menuOpen)}
				class="rounded p-1.5 text-gray-600 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-800 sm:hidden"
				aria-label="Toggle menu"
			>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
					{#if menuOpen}
						<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
					{:else}
						<path fill-rule="evenodd" d="M3 5a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 10a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zM3 15a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clip-rule="evenodd" />
					{/if}
				</svg>
			</button>
		</div>
	</div>

	<!-- Mobile dropdown -->
	{#if menuOpen}
		<div class="border-t border-gray-200 pb-3 pt-2 dark:border-gray-800 sm:hidden">
			{#each links as link}
				<a
					href={link.href}
					onclick={() => (menuOpen = false)}
					class="block px-4 py-2 text-sm transition-colors {page.url.pathname.startsWith(link.href)
						? 'font-medium text-gray-900 dark:text-white'
						: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
				>
					{link.label}
				</a>
			{/each}
		</div>
	{/if}
</nav>

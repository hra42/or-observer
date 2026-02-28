<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import { browser } from '$app/environment';
	import Nav from '$lib/components/Nav.svelte';

	let { children } = $props();

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				enabled: browser,
				staleTime: 1000 * 60 * 5,
				gcTime: 1000 * 60 * 10
			}
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>or-observer</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div class="min-h-screen bg-gray-950 text-gray-100">
		<Nav />
		<main class="mx-auto max-w-7xl px-4 py-6">
			{@render children()}
		</main>
	</div>
</QueryClientProvider>

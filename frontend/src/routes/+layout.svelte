<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '../app.css';
	import { QueryClient, QueryClientProvider, HydrationBoundary } from '@tanstack/svelte-query';
	import { browser } from '$app/environment';
	import Nav from '$lib/components/Nav.svelte';
	import { getTheme } from '$lib/stores/theme.svelte';
	import { page } from '$app/state';

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

	let theme = $derived(getTheme());
	let dehydratedState = $derived(page.data?.dehydratedState);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>or-observer</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<HydrationBoundary state={dehydratedState}>
		<div class="min-h-screen bg-white text-gray-900 dark:bg-gray-950 dark:text-gray-100" class:dark={theme === 'dark'}>
			<Nav />
			<main class="mx-auto max-w-7xl px-4 py-6">
				{@render children()}
			</main>
		</div>
	</HydrationBoundary>
</QueryClientProvider>

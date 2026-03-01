<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import Nav from '$lib/components/Nav.svelte';

	const { data, children } = $props();
</script>

<QueryClientProvider client={data.queryClient}>
	<div class="min-h-screen bg-white text-gray-900 dark:bg-gray-950 dark:text-gray-100">
		<Nav />
		<main class="mx-auto max-w-7xl px-4 py-6">
			{@render children()}
		</main>
	</div>
</QueryClientProvider>

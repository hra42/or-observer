import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchTraces } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	const queryClient = new QueryClient();
	const apiKey = locals.apiKey;

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['traces', '', '', '', '', 50, 0],
			queryFn: () => fetchTraces({ limit: 50, offset: 0 }, apiKey)
		})
	]);

	return { dehydratedState: dehydrate(queryClient), apiKey };
};

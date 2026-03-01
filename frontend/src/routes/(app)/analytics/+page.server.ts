import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchCostsBreakdown } from '$lib/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	const queryClient = new QueryClient();
	const apiKey = locals.apiKey;

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily', '', ''],
			queryFn: () => fetchCostsBreakdown('model', 'daily', undefined, undefined, apiKey)
		})
	]);

	return { dehydratedState: dehydrate(queryClient), apiKey };
};

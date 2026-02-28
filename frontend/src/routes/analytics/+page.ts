import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchCostsBreakdown } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const queryClient = new QueryClient();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily', '', ''],
			queryFn: () => fetchCostsBreakdown('model', 'daily')
		})
	]);

	return { dehydratedState: dehydrate(queryClient) };
};

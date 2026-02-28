import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchTraces } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const queryClient = new QueryClient();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['traces', '', '', '', '', 50, 0],
			queryFn: () => fetchTraces({ limit: 50, offset: 0 })
		})
	]);

	return { dehydratedState: dehydrate(queryClient) };
};

import { fetchTraces } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { queryClient, apiKey } = await parent();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['traces', '', '', '', '', 50, 0],
			queryFn: () => fetchTraces({ limit: 50, offset: 0 }, apiKey)
		})
	]);
};

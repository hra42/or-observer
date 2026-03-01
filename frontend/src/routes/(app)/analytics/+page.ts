import { fetchCostsBreakdown } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { queryClient, apiKey } = await parent();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily', '', ''],
			queryFn: () => fetchCostsBreakdown('model', 'daily', undefined, undefined, apiKey)
		})
	]);
};

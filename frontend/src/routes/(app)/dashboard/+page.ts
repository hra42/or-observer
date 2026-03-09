import { fetchHealth, fetchMetricsHourly, fetchCostsBreakdown } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { queryClient, apiKey } = await parent();

	const defaultDays = 30;
	const now = new Date();
	const start = new Date(now.getTime() - defaultDays * 24 * 60 * 60 * 1000).toISOString();
	const end = now.toISOString();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['health'],
			queryFn: () => fetchHealth(apiKey)
		}),
		queryClient.prefetchQuery({
			queryKey: ['metrics', 'hourly', defaultDays],
			queryFn: () => fetchMetricsHourly(start, end, 'model', apiKey)
		}),
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily', defaultDays],
			queryFn: () => fetchCostsBreakdown('model', 'daily', start, end, apiKey)
		})
	]);
};

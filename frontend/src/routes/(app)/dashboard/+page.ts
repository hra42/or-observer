import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchHealth, fetchMetricsHourly, fetchCostsBreakdown } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent }) => {
	const { apiKey } = await parent();
	const queryClient = new QueryClient();

	const now = new Date();
	const start24h = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
	const end24h = now.toISOString();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['health'],
			queryFn: () => fetchHealth(apiKey)
		}),
		queryClient.prefetchQuery({
			queryKey: ['metrics', 'hourly', '24h'],
			queryFn: () => fetchMetricsHourly(start24h, end24h, 'model', apiKey)
		}),
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily'],
			queryFn: () => fetchCostsBreakdown('model', 'daily', undefined, undefined, apiKey)
		})
	]);

	return { dehydratedState: dehydrate(queryClient) };
};

import { QueryClient, dehydrate } from '@tanstack/svelte-query';
import { fetchHealth, fetchMetricsHourly, fetchCostsBreakdown } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const queryClient = new QueryClient();

	const now = new Date();
	const start24h = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
	const end24h = now.toISOString();

	await Promise.allSettled([
		queryClient.prefetchQuery({
			queryKey: ['health'],
			queryFn: fetchHealth
		}),
		queryClient.prefetchQuery({
			queryKey: ['metrics', 'hourly', '24h'],
			queryFn: () => fetchMetricsHourly(start24h, end24h, 'model')
		}),
		queryClient.prefetchQuery({
			queryKey: ['costs', 'model', 'daily'],
			queryFn: () => fetchCostsBreakdown('model', 'daily')
		})
	]);

	return { dehydratedState: dehydrate(queryClient) };
};

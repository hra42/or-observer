import { QueryClient } from '@tanstack/svelte-query';
import { browser } from '$app/environment';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = ({ data }) => {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				enabled: browser,
				staleTime: 1000 * 60 * 5,
				gcTime: 1000 * 60 * 10
			}
		}
	});

	return { queryClient, apiKey: data.apiKey, authRequired: data.authRequired };
};

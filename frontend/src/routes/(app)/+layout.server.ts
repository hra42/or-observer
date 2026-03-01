import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals }) => {
	return {
		apiKey: locals.apiKey,
		authRequired: locals.authRequired
	};
};

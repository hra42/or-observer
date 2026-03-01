import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
	cookies.delete('or_observer_api_key', { path: '/' });
	redirect(302, '/login');
};

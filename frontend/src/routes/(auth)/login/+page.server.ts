import { fail, redirect } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	if (!locals.authRequired) {
		redirect(302, '/dashboard');
	}
};

export const actions: Actions = {
	default: async ({ request, cookies, url }) => {
		const apiKey = env.API_KEY ?? '';
		if (!apiKey) {
			redirect(302, '/dashboard');
		}

		const data = await request.formData();
		const key = data.get('key');

		if (!key || key !== apiKey) {
			return fail(401, { error: 'Invalid API key' });
		}

		cookies.set('or_observer_api_key', String(key), {
			path: '/',
			httpOnly: true,
			sameSite: 'strict',
			secure: url.protocol === 'https:',
			maxAge: 60 * 60 * 24 * 30 // 30 days
		});

		const redirectTo = url.searchParams.get('redirect') || '/dashboard';
		redirect(302, redirectTo);
	}
};

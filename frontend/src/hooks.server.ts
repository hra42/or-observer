import { redirect, type Handle } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

export const handle: Handle = async ({ event, resolve }) => {
	const apiKey = env.API_KEY ?? '';
	const authRequired = apiKey !== '';

	event.locals.apiKey = apiKey;
	event.locals.authRequired = authRequired;

	if (!authRequired) {
		return resolve(event);
	}

	// Public paths that don't require auth
	if (event.url.pathname === '/login' || event.url.pathname === '/logout') {
		return resolve(event);
	}

	const cookie = event.cookies.get('or_observer_api_key');
	if (cookie !== apiKey) {
		const redirectParam = encodeURIComponent(event.url.pathname);
		redirect(302, `/login?redirect=${redirectParam}`);
	}

	return resolve(event);
};

import { browser } from '$app/environment';

const STORAGE_KEY = 'or-observer-theme';

function getInitial(): 'dark' | 'light' {
	if (!browser) return 'dark';
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored === 'light' || stored === 'dark') return stored;
	return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

let current = $state<'dark' | 'light'>(getInitial());

export function getTheme(): 'dark' | 'light' {
	return current;
}

export function isDark(): boolean {
	return current === 'dark';
}

export function toggleTheme() {
	current = current === 'dark' ? 'light' : 'dark';
	if (browser) {
		localStorage.setItem(STORAGE_KEY, current);
	}
}

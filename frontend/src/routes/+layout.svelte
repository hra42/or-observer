<script lang="ts">
	import '../app.css';
	import { getTheme } from '$lib/stores/theme.svelte';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
	let theme = $derived(getTheme());
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>or-observer</title>
</svelte:head>

<div class:dark={theme === 'dark'}>
	{@render children()}
</div>

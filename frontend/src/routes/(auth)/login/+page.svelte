<script lang="ts">
	import { page } from '$app/state';

	let { form } = $props();
</script>

<div class="w-full max-w-sm rounded-lg bg-white p-8 shadow-lg dark:bg-gray-800">
	<h1 class="mb-6 text-center text-2xl font-bold text-gray-900 dark:text-white">or-observer</h1>
	<p class="mb-6 text-center text-sm text-gray-600 dark:text-gray-400">Enter your API key to continue</p>

	{#if form?.error}
		<div class="mb-4 rounded bg-red-100 px-4 py-2 text-sm text-red-700 dark:bg-red-900/30 dark:text-red-400">
			{form.error}
		</div>
	{/if}

	<form method="POST">
		<input type="hidden" name="redirect" value={page.url.searchParams.get('redirect') ?? '/dashboard'} />
		<label for="key" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">API Key</label>
		<input
			id="key"
			name="key"
			type="password"
			required
			autofocus
			class="mb-4 w-full rounded border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
			placeholder="Enter API key"
		/>
		<button
			type="submit"
			class="w-full rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
		>
			Sign in
		</button>
	</form>
</div>

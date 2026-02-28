<script lang="ts">
	import { page } from '$app/state';
</script>

<div class="flex min-h-[50vh] flex-col items-center justify-center text-center">
	<p class="text-6xl font-bold text-gray-300 dark:text-gray-700">{page.status}</p>
	<p class="mt-4 text-lg text-gray-600 dark:text-gray-400">
		{page.error?.message ?? 'Something went wrong'}
	</p>
	<a
		href="/dashboard"
		class="mt-6 rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
	>
		Back to Dashboard
	</a>
</div>

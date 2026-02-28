<script lang="ts">
	let { message, onRetry }: { message: string; onRetry?: () => void } = $props();
</script>

<div class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20">
	<div class="flex items-start gap-3">
		<svg
			class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500 dark:text-red-400"
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 20 20"
			fill="currentColor"
		>
			<path
				fill-rule="evenodd"
				d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z"
				clip-rule="evenodd"
			/>
		</svg>
		<div class="flex-1">
			<p class="text-sm text-red-700 dark:text-red-300">{message}</p>
			{#if onRetry}
				<button
					onclick={onRetry}
					class="mt-2 text-sm font-medium text-red-600 hover:text-red-500 dark:text-red-400 dark:hover:text-red-300"
				>
					Retry
				</button>
			{/if}
		</div>
	</div>
</div>

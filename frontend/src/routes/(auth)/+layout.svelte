<script lang="ts">
	let { children } = $props();
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-100 dark:bg-gray-950">
	{@render children()}
</div>

<script lang="ts">
	import { browser } from '$app/environment';

	let { cost24h = 0, errors24h = 0 }: { cost24h?: number; errors24h?: number } = $props();
	let dismissed = $state<Set<string>>(new Set());

	const THRESHOLDS_KEY = 'or-observer-alert-thresholds';

	function getThresholds(): { costPerDay: number; maxErrors: number } {
		if (!browser) return { costPerDay: 5, maxErrors: 0 };
		try {
			const stored = localStorage.getItem(THRESHOLDS_KEY);
			if (stored) return JSON.parse(stored);
		} catch {
			// ignore
		}
		return { costPerDay: 5, maxErrors: 0 };
	}

	let thresholds = $derived(getThresholds());

	let alerts = $derived.by(() => {
		const list: { id: string; type: 'warning' | 'error'; message: string }[] = [];
		if (cost24h > thresholds.costPerDay) {
			list.push({
				id: 'cost',
				type: 'warning',
				message: `24h cost ($${cost24h.toFixed(2)}) exceeds threshold ($${thresholds.costPerDay})`
			});
		}
		if (errors24h > thresholds.maxErrors) {
			list.push({
				id: 'errors',
				type: 'error',
				message: `${errors24h} errors in the last 24h (threshold: ${thresholds.maxErrors})`
			});
		}
		return list.filter((a) => !dismissed.has(a.id));
	});

	function dismiss(id: string) {
		dismissed = new Set([...dismissed, id]);
	}
</script>

{#each alerts as alert}
	<div
		class="flex items-center justify-between rounded-lg px-4 py-3 text-sm {alert.type === 'error'
			? 'border border-red-200 bg-red-50 text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-300'
			: 'border border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-300'}"
	>
		<span>{alert.message}</span>
		<button
			onclick={() => dismiss(alert.id)}
			class="ml-4 flex-shrink-0 text-current opacity-60 hover:opacity-100"
			aria-label="Dismiss"
		>
			✕
		</button>
	</div>
{/each}

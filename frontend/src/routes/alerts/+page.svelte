<script lang="ts">
	import { browser } from '$app/environment';
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchMetricsHourly, type MetricRow } from '$lib/api';

	const THRESHOLDS_KEY = 'or-observer-alert-thresholds';

	function loadThresholds(): { costPerDay: number; maxErrors: number; maxLatencyMs: number } {
		if (!browser) return { costPerDay: 5, maxErrors: 0, maxLatencyMs: 5000 };
		try {
			const stored = localStorage.getItem(THRESHOLDS_KEY);
			if (stored) return { costPerDay: 5, maxErrors: 0, maxLatencyMs: 5000, ...JSON.parse(stored) };
		} catch {
			// ignore
		}
		return { costPerDay: 5, maxErrors: 0, maxLatencyMs: 5000 };
	}

	let thresholds = $state(loadThresholds());

	function save() {
		if (browser) {
			localStorage.setItem(THRESHOLDS_KEY, JSON.stringify(thresholds));
			saved = true;
			setTimeout(() => (saved = false), 2000);
		}
	}

	let saved = $state(false);

	// Fetch current stats to show status
	const now = new Date();
	const start24h = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();

	const metricsQuery = createQuery(() => ({
		queryKey: ['metrics', 'hourly', '24h-alerts'],
		queryFn: () => fetchMetricsHourly(start24h, now.toISOString(), '')
	}));

	let totalCost = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.total_cost, 0)
	);
	let totalErrors = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.error_count, 0)
	);
	let maxP95 = $derived(
		Math.max(0, ...(metricsQuery.data?.metrics ?? []).map((m: MetricRow) => m.p95_latency_ms))
	);

	const inputClass =
		'w-32 rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white';
</script>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">Alert Configuration</h1>
	<p class="text-gray-600 dark:text-gray-400">
		Configure thresholds for dashboard alert banners. Alerts are checked against the last 24 hours
		of data and stored locally in your browser.
	</p>

	<div class="space-y-4 rounded-lg bg-gray-100 p-6 dark:bg-gray-800">
		<!-- Cost threshold -->
		<div class="flex flex-wrap items-center justify-between gap-4 border-b border-gray-200 pb-4 dark:border-gray-700">
			<div>
				<p class="font-medium">Daily cost threshold</p>
				<p class="text-sm text-gray-600 dark:text-gray-400">Alert when 24h cost exceeds this amount</p>
			</div>
			<div class="flex items-center gap-3">
				<div class="flex items-center gap-1">
					<span class="text-sm text-gray-600 dark:text-gray-400">$</span>
					<input
						type="number"
						step="0.5"
						min="0"
						bind:value={thresholds.costPerDay}
						class={inputClass}
					/>
				</div>
				<div class="text-sm {totalCost > thresholds.costPerDay ? 'text-amber-600 dark:text-amber-400 font-medium' : 'text-gray-500'}">
					Current: ${totalCost.toFixed(2)}
				</div>
			</div>
		</div>

		<!-- Error threshold -->
		<div class="flex flex-wrap items-center justify-between gap-4 border-b border-gray-200 pb-4 dark:border-gray-700">
			<div>
				<p class="font-medium">Error count threshold</p>
				<p class="text-sm text-gray-600 dark:text-gray-400">Alert when error count exceeds this in 24h</p>
			</div>
			<div class="flex items-center gap-3">
				<input
					type="number"
					step="1"
					min="0"
					bind:value={thresholds.maxErrors}
					class={inputClass}
				/>
				<div class="text-sm {totalErrors > thresholds.maxErrors ? 'text-red-600 dark:text-red-400 font-medium' : 'text-gray-500'}">
					Current: {totalErrors}
				</div>
			</div>
		</div>

		<!-- Latency threshold -->
		<div class="flex flex-wrap items-center justify-between gap-4 pb-2">
			<div>
				<p class="font-medium">P95 latency threshold (ms)</p>
				<p class="text-sm text-gray-600 dark:text-gray-400">Alert when P95 latency exceeds this</p>
			</div>
			<div class="flex items-center gap-3">
				<input
					type="number"
					step="100"
					min="0"
					bind:value={thresholds.maxLatencyMs}
					class={inputClass}
				/>
				<div class="text-sm {maxP95 > thresholds.maxLatencyMs ? 'text-amber-600 dark:text-amber-400 font-medium' : 'text-gray-500'}">
					Current P95: {maxP95.toFixed(0)}ms
				</div>
			</div>
		</div>
	</div>

	<div class="flex items-center gap-4">
		<button
			onclick={save}
			class="rounded bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
		>
			Save thresholds
		</button>
		{#if saved}
			<span class="text-sm text-green-600 dark:text-green-400">Saved!</span>
		{/if}
	</div>
</div>

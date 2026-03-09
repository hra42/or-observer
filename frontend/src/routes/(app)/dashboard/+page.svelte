<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchMetricsHourly, fetchCostsBreakdown, fetchHealth, type MetricRow } from '$lib/api';
	import { LineChart } from 'layerchart';
	import { scaleBand } from 'd3-scale';
	import * as Chart from '$lib/components/ui/chart/index.js';
	import Spinner from '$lib/components/Spinner.svelte';
	import ErrorAlert from '$lib/components/ErrorAlert.svelte';
	import AlertBanner from '$lib/components/AlertBanner.svelte';

	let apiKey = $derived(page.data.apiKey ?? '');

	// Time range configuration
	const RANGE_STORAGE_KEY = 'or-observer-dashboard-range';
	const rangeOptions = [
		{ value: 1, label: '24h' },
		{ value: 7, label: '7d' },
		{ value: 14, label: '14d' },
		{ value: 30, label: '30d' }
	] as const;

	let rangeDays = $state(30);

	function loadRangePrefs() {
		if (!browser) return;
		try {
			const stored = localStorage.getItem(RANGE_STORAGE_KEY);
			if (stored) rangeDays = JSON.parse(stored);
		} catch { /* ignore */ }
	}

	loadRangePrefs();

	$effect(() => {
		if (!browser) return;
		localStorage.setItem(RANGE_STORAGE_KEY, JSON.stringify(rangeDays));
	});

	let rangeStart = $derived(new Date(Date.now() - rangeDays * 24 * 60 * 60 * 1000).toISOString());
	let rangeEnd = $derived(new Date().toISOString());
	let rangeLabel = $derived(rangeOptions.find((o) => o.value === rangeDays)?.label ?? `${rangeDays}d`);

	const healthQuery = createQuery(() => ({
		queryKey: ['health'],
		queryFn: () => fetchHealth(apiKey),
		refetchInterval: 30_000
	}));

	const metricsQuery = createQuery(() => ({
		queryKey: ['metrics', 'hourly', rangeDays],
		queryFn: () => fetchMetricsHourly(rangeStart, rangeEnd, 'model', apiKey)
	}));

	const costsQuery = createQuery(() => ({
		queryKey: ['costs', 'model', 'daily', rangeDays],
		queryFn: () => fetchCostsBreakdown('model', 'daily', rangeStart, rangeEnd, apiKey)
	}));

	let totalCost = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.total_cost, 0)
	);
	let totalRequests = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.request_count, 0)
	);
	let totalErrors = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.error_count, 0)
	);
	let maxP95Latency = $derived(
		Math.max(0, ...(metricsQuery.data?.metrics ?? []).map((m: MetricRow) => m.p95_latency_ms))
	);

	let chartData = $derived.by(() => {
		const byHour = new Map<string, { cost: number; requests: number }>();
		for (const m of metricsQuery.data?.metrics ?? []) {
			const fmt = rangeDays <= 1
				? new Date(m.hour).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
				: new Date(m.hour).toLocaleDateString([], { month: 'short', day: 'numeric' });
			const prev = byHour.get(fmt) ?? { cost: 0, requests: 0 };
			byHour.set(fmt, { cost: prev.cost + m.total_cost, requests: prev.requests + m.request_count });
		}
		return Array.from(byHour.entries())
			.map(([hour, v]) => ({ hour, cost: +v.cost.toFixed(4), requests: v.requests }))
			.reverse();
	});

	const chartConfig = {
		cost: { label: 'Cost ($)', color: '#818cf8' },
		requests: { label: 'Requests', color: '#34d399' }
	} satisfies Chart.ChartConfig;
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">Dashboard</h1>
		<div class="flex items-center gap-1 rounded-lg bg-gray-100 p-1 dark:bg-gray-800">
			{#each rangeOptions as opt}
				<button
					onclick={() => (rangeDays = opt.value)}
					class="rounded-md px-3 py-1 text-sm font-medium transition-colors {rangeDays === opt.value
						? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white'
						: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
				>
					{opt.label}
				</button>
			{/each}
		</div>
	</div>

	<AlertBanner cost24h={totalCost} errors24h={totalErrors} p95LatencyMs={maxP95Latency} />

	<!-- Summary cards -->
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<p class="text-sm text-gray-600 dark:text-gray-400">Total traces</p>
			{#if healthQuery.isLoading}
				<div class="mt-2"><Spinner size="sm" /></div>
			{:else if healthQuery.isError}
				<p class="mt-1 text-sm text-red-500 dark:text-red-400">Error</p>
			{:else}
				<p class="mt-1 text-2xl font-bold">{healthQuery.data?.traces_ingested.toLocaleString()}</p>
			{/if}
		</div>

		<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<p class="text-sm text-gray-600 dark:text-gray-400">Requests ({rangeLabel})</p>
			{#if metricsQuery.isLoading}
				<div class="mt-2"><Spinner size="sm" /></div>
			{:else}
				<p class="mt-1 text-2xl font-bold">{totalRequests.toLocaleString()}</p>
			{/if}
		</div>

		<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<p class="text-sm text-gray-600 dark:text-gray-400">Cost ({rangeLabel})</p>
			{#if metricsQuery.isLoading}
				<div class="mt-2"><Spinner size="sm" /></div>
			{:else}
				<p class="mt-1 text-2xl font-bold">${totalCost.toFixed(4)}</p>
			{/if}
		</div>

		<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<p class="text-sm text-gray-600 dark:text-gray-400">DB status</p>
			<p
				class="mt-1 text-lg font-semibold {healthQuery.data?.database === 'connected'
					? 'text-green-600 dark:text-green-400'
					: 'text-red-500 dark:text-red-400'}"
			>
				{healthQuery.data?.database ?? '...'}
			</p>
		</div>
	</div>

	<!-- Trend chart -->
	<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
		<h2 class="mb-4 text-lg font-semibold">Hourly trend ({rangeLabel})</h2>
		{#if metricsQuery.isLoading}
			<div class="flex h-48 items-center justify-center"><Spinner /></div>
		{:else if metricsQuery.isError}
			<ErrorAlert message="Failed to load metrics" onRetry={() => metricsQuery.refetch()} />
		{:else if chartData.length === 0}
			<div class="flex h-48 items-center justify-center text-gray-500 dark:text-gray-500">No data yet</div>
		{:else}
			<Chart.Container config={chartConfig} class="min-h-[240px] w-full">
				<LineChart
					data={chartData}
					x="hour"
					xScale={scaleBand()}
					axis="x"
					legend
					series={[
						{ key: 'cost', label: chartConfig.cost.label, color: chartConfig.cost.color },
						{ key: 'requests', label: chartConfig.requests.label, color: chartConfig.requests.color }
					]}
					props={{
						spline: { strokeWidth: 2 },
						xAxis: { format: (d: string) => d }
					}}
				>
					{#snippet tooltip()}
						<Chart.Tooltip />
					{/snippet}
				</LineChart>
			</Chart.Container>
		{/if}
	</div>

	<!-- Top models table -->
	<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
		<h2 class="mb-4 text-lg font-semibold">Top models by cost ({rangeLabel})</h2>
		{#if costsQuery.isLoading}
			<div class="py-4"><Spinner /></div>
		{:else if costsQuery.isError}
			<ErrorAlert message="Failed to load cost data" onRetry={() => costsQuery.refetch()} />
		{:else if (costsQuery.data?.breakdown ?? []).length === 0}
			<p class="text-gray-500">No data yet</p>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-200 text-left text-gray-600 dark:border-gray-700 dark:text-gray-400">
							<th class="pb-2 pr-4">Model</th>
							<th class="pb-2 pr-4 text-right">Requests</th>
							<th class="pb-2 pr-4 text-right">Total cost</th>
							<th class="pb-2 text-right">Avg cost</th>
						</tr>
					</thead>
					<tbody>
						{#each costsQuery.data?.breakdown ?? [] as row}
							<tr class="border-b border-gray-200/50 dark:border-gray-700/50">
								<td class="py-2 pr-4 font-mono text-indigo-600 dark:text-indigo-300">{row.dimension}</td>
								<td class="py-2 pr-4 text-right">{row.request_count.toLocaleString()}</td>
								<td class="py-2 pr-4 text-right">${row.total_cost.toFixed(4)}</td>
								<td class="py-2 text-right">${row.avg_cost.toFixed(6)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

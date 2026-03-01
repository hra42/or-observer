<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchMetricsHourly, fetchCostsBreakdown, fetchHealth, type MetricRow } from '$lib/api';
	import {
		LineChart,
		Line,
		XAxis,
		YAxis,
		CartesianGrid,
		Legend,
		ResponsiveContainer
	} from 'recharts';
	import { Chart_Tooltip as Tooltip } from '$lib/recharts';
	import Spinner from '$lib/components/Spinner.svelte';
	import ErrorAlert from '$lib/components/ErrorAlert.svelte';
	import AlertBanner from '$lib/components/AlertBanner.svelte';
	import { isDark } from '$lib/stores/theme.svelte';

	let apiKey = $derived(page.data.apiKey ?? '');

	const now = new Date();
	const start24h = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
	const end24h = now.toISOString();

	const healthQuery = createQuery(() => ({
		queryKey: ['health'],
		queryFn: () => fetchHealth(apiKey),
		refetchInterval: 30_000
	}));

	const metricsQuery = createQuery(() => ({
		queryKey: ['metrics', 'hourly', '24h'],
		queryFn: () => fetchMetricsHourly(start24h, end24h, 'model', apiKey)
	}));

	const costsQuery = createQuery(() => ({
		queryKey: ['costs', 'model', 'daily'],
		queryFn: () => fetchCostsBreakdown('model', 'daily', undefined, undefined, apiKey)
	}));

	let totalCost24h = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.total_cost, 0)
	);
	let totalRequests24h = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.request_count, 0)
	);
	let totalErrors24h = $derived(
		(metricsQuery.data?.metrics ?? []).reduce((s: number, m: MetricRow) => s + m.error_count, 0)
	);
	let maxP95Latency24h = $derived(
		Math.max(0, ...(metricsQuery.data?.metrics ?? []).map((m: MetricRow) => m.p95_latency_ms))
	);

	let chartData = $derived.by(() => {
		const byHour = new Map<string, { cost: number; requests: number }>();
		for (const m of metricsQuery.data?.metrics ?? []) {
			const h = new Date(m.hour).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
			const prev = byHour.get(h) ?? { cost: 0, requests: 0 };
			byHour.set(h, { cost: prev.cost + m.total_cost, requests: prev.requests + m.request_count });
		}
		return Array.from(byHour.entries())
			.map(([hour, v]) => ({ hour, cost: +v.cost.toFixed(4), requests: v.requests }))
			.reverse();
	});

	let gridStroke = $derived(isDark() ? '#374151' : '#e5e7eb');
	let axisStroke = $derived(isDark() ? '#9ca3af' : '#6b7280');
</script>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">Dashboard</h1>

	<AlertBanner cost24h={totalCost24h} errors24h={totalErrors24h} p95LatencyMs={maxP95Latency24h} />

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
			<p class="text-sm text-gray-600 dark:text-gray-400">Requests (24h)</p>
			{#if metricsQuery.isLoading}
				<div class="mt-2"><Spinner size="sm" /></div>
			{:else}
				<p class="mt-1 text-2xl font-bold">{totalRequests24h.toLocaleString()}</p>
			{/if}
		</div>

		<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<p class="text-sm text-gray-600 dark:text-gray-400">Cost (24h)</p>
			{#if metricsQuery.isLoading}
				<div class="mt-2"><Spinner size="sm" /></div>
			{:else}
				<p class="mt-1 text-2xl font-bold">${totalCost24h.toFixed(4)}</p>
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
		<h2 class="mb-4 text-lg font-semibold">Hourly trend (24h)</h2>
		{#if metricsQuery.isLoading}
			<div class="flex h-48 items-center justify-center"><Spinner /></div>
		{:else if metricsQuery.isError}
			<ErrorAlert message="Failed to load metrics" onRetry={() => metricsQuery.refetch()} />
		{:else if chartData.length === 0}
			<div class="flex h-48 items-center justify-center text-gray-500 dark:text-gray-500">No data yet</div>
		{:else if browser}
			<ResponsiveContainer width="100%" height={240}>
				<LineChart data={chartData}>
					<CartesianGrid strokeDasharray="3 3" stroke={gridStroke} />
					<XAxis dataKey="hour" stroke={axisStroke} tick={{ fontSize: 11 }} />
					<YAxis yAxisId="left" stroke={axisStroke} tick={{ fontSize: 11 }} />
					<YAxis yAxisId="right" orientation="right" stroke={axisStroke} tick={{ fontSize: 11 }} />
					<Tooltip />
					<Legend />
					<Line
						yAxisId="left"
						type="monotone"
						dataKey="cost"
						stroke="#818cf8"
						name="Cost ($)"
						dot={false}
					/>
					<Line
						yAxisId="right"
						type="monotone"
						dataKey="requests"
						stroke="#34d399"
						name="Requests"
						dot={false}
					/>
				</LineChart>
			</ResponsiveContainer>
		{/if}
	</div>

	<!-- Top models table -->
	<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
		<h2 class="mb-4 text-lg font-semibold">Top models by cost (today)</h2>
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

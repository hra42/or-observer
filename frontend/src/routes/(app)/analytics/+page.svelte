<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchCostsBreakdown, fetchMetricsHourly, type MetricRow, type BreakdownRow } from '$lib/api';
	import { BarChart, LineChart } from 'layerchart';
	import { scaleBand } from 'd3-scale';
	import * as Chart from '$lib/components/ui/chart/index.js';
	import Spinner from '$lib/components/Spinner.svelte';
	import ErrorAlert from '$lib/components/ErrorAlert.svelte';

	let apiKey = $derived(page.data.apiKey ?? '');

	let groupBy = $state<'model' | 'user'>('model');
	let costPeriod = $state<'hourly' | 'daily' | 'overall'>('daily');
	let activeTab = $state<'costs' | 'latency'>('costs');
	let latencyGroupBy = $state<'model' | 'user' | ''>('');

	function toRFC3339(local: string): string {
		if (!local) return '';
		return new Date(local).toISOString();
	}

	const now = new Date();
	function toLocalInput(d: Date): string {
		return d.toISOString().slice(0, 16);
	}
	let rangeStart = $state(toLocalInput(new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)));
	let rangeEnd = $state(toLocalInput(now));
	let costRangeStart = $state('');
	let costRangeEnd = $state('');

	const costsQuery = createQuery(() => ({
		queryKey: ['costs', groupBy, costPeriod, costRangeStart, costRangeEnd],
		queryFn: () =>
			fetchCostsBreakdown(
				groupBy,
				costPeriod,
				costRangeStart ? toRFC3339(costRangeStart) : undefined,
				costRangeEnd ? toRFC3339(costRangeEnd) : undefined,
				apiKey
			)
	}));

	const latencyQuery = createQuery(() => ({
		queryKey: ['metrics', 'latency', rangeStart, rangeEnd, latencyGroupBy],
		queryFn: () => fetchMetricsHourly(toRFC3339(rangeStart), toRFC3339(rangeEnd), latencyGroupBy, apiKey)
	}));

	let latencyData = $derived.by(() => {
		const raw = latencyQuery.data?.metrics ?? [];
		// Aggregate into daily buckets when range > 2 days for readability
		const rangeMs = new Date(rangeEnd).getTime() - new Date(rangeStart).getTime();
		const rangeDays = rangeMs / (24 * 60 * 60 * 1000);

		if (rangeDays > 2) {
			const byDay = new Map<string, { avgWeighted: number; totalReqs: number; p95: number[]; p99: number[] }>();
			for (const m of raw) {
				// Use ISO date as key to avoid cross-year collisions
				const isoKey = new Date(m.hour).toISOString().slice(0, 10);
				const bucket = byDay.get(isoKey) ?? { avgWeighted: 0, totalReqs: 0, p95: [], p99: [] };
				bucket.avgWeighted += m.avg_latency_ms * m.request_count;
				bucket.totalReqs += m.request_count;
				bucket.p95.push(m.p95_latency_ms);
				bucket.p99.push(m.p99_latency_ms);
				byDay.set(isoKey, bucket);
			}
			return Array.from(byDay.entries())
				.sort(([a], [b]) => a.localeCompare(b))
				.map(([isoKey, v]) => ({
					hour: new Date(isoKey + 'T00:00:00').toLocaleDateString([], { month: 'short', day: 'numeric' }),
					avg: +(v.totalReqs > 0 ? v.avgWeighted / v.totalReqs : 0).toFixed(1),
					p95: +Math.max(...v.p95).toFixed(1),
					p99: +Math.max(...v.p99).toFixed(1)
				}));
		}

		return raw
			.map((m: MetricRow) => ({
				hour: new Date(m.hour).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
				avg: +m.avg_latency_ms.toFixed(1),
				p95: +m.p95_latency_ms.toFixed(1),
				p99: +m.p99_latency_ms.toFixed(1)
			}))
			.reverse();
	});

	let costData = $derived(
		(costsQuery.data?.breakdown ?? []).map((b: BreakdownRow) => ({
			name: b.dimension.length > 20 ? b.dimension.slice(0, 20) + '…' : b.dimension,
			cost: +b.total_cost.toFixed(4),
			requests: b.request_count
		}))
	);

	const costChartConfig = {
		cost: { label: 'Cost ($)', color: '#818cf8' }
	} satisfies Chart.ChartConfig;

	const latencyChartConfig = {
		avg: { label: 'Avg (ms)', color: '#6366f1' },
		p95: { label: 'P95 (ms)', color: '#f59e0b' },
		p99: { label: 'P99 (ms)', color: '#ef4444' }
	} satisfies Chart.ChartConfig;

	// Thin x-axis labels: show at most ~maxTicks labels evenly spaced
	function thinLabels(data: { hour: string }[], maxTicks = 12): (d: string) => string {
		const step = Math.max(1, Math.ceil(data.length / maxTicks));
		const show = new Set(data.filter((_, i) => i % step === 0).map((d) => d.hour));
		return (d: string) => (show.has(d) ? d : '');
	}

	let latencyTickFormat = $derived(thinLabels(latencyData));
	let costTickFormat = $derived(thinLabels(costData.map((d) => ({ hour: d.name }))));

	const inputClass =
		'rounded bg-gray-200 px-3 py-1 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white';
</script>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">Analytics</h1>

	<!-- Tabs -->
	<div class="flex w-fit gap-1 rounded-lg bg-gray-200 p-1 dark:bg-gray-800">
		<button
			onclick={() => (activeTab = 'costs')}
			class="rounded px-4 py-1.5 text-sm font-medium transition-colors {activeTab === 'costs'
				? 'bg-indigo-600 text-white'
				: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
		>
			Cost Breakdown
		</button>
		<button
			onclick={() => (activeTab = 'latency')}
			class="rounded px-4 py-1.5 text-sm font-medium transition-colors {activeTab === 'latency'
				? 'bg-indigo-600 text-white'
				: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white'}"
		>
			Latency
		</button>
	</div>

	{#if activeTab === 'costs'}
		<!-- Controls -->
		<div class="flex flex-wrap gap-3 rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<div class="flex items-center gap-2">
				<label for="cost-group-by" class="text-sm text-gray-600 dark:text-gray-400">Group by</label>
				<select id="cost-group-by" bind:value={groupBy} class={inputClass}>
					<option value="model">Model</option>
					<option value="user">User</option>
				</select>
			</div>
			<div class="flex items-center gap-2">
				<label for="cost-period" class="text-sm text-gray-600 dark:text-gray-400">Period</label>
				<select id="cost-period" bind:value={costPeriod} class={inputClass}>
					<option value="hourly">Last hour</option>
					<option value="daily">Last 24h</option>
					<option value="overall">All time</option>
				</select>
			</div>
			<div class="flex items-center gap-2">
				<label for="cost-range-start" class="text-sm text-gray-600 dark:text-gray-400">Start</label>
				<input id="cost-range-start" bind:value={costRangeStart} type="datetime-local" class={inputClass} />
			</div>
			<div class="flex items-center gap-2">
				<label for="cost-range-end" class="text-sm text-gray-600 dark:text-gray-400">End</label>
				<input id="cost-range-end" bind:value={costRangeEnd} type="datetime-local" class={inputClass} />
			</div>
		</div>

		{#if costsQuery.isLoading}
			<div class="py-12 text-center"><Spinner /></div>
		{:else if costsQuery.isError}
			<ErrorAlert message="Failed to load cost data" onRetry={() => costsQuery.refetch()} />
		{:else if costData.length === 0}
			<div class="py-12 text-center text-gray-500">No data for this period</div>
		{:else}
			<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
				<h2 class="mb-4 text-lg font-semibold">Cost by {groupBy}</h2>
				<Chart.Container config={costChartConfig} class="min-h-[280px] w-full">
					<BarChart
						data={costData}
						xScale={scaleBand().padding(0.25)}
						x="name"
						axis="x"
						series={[{ key: 'cost', label: costChartConfig.cost.label, color: costChartConfig.cost.color }]}
						props={{
							bars: { stroke: 'none', rounded: 'all', radius: 4 },
							highlight: { area: { fill: 'none' } },
							xAxis: { format: costTickFormat }
						}}
					>
						{#snippet tooltip()}
							<Chart.Tooltip />
						{/snippet}
					</BarChart>
				</Chart.Container>
			</div>

			<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
				<div class="overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr class="border-b border-gray-200 text-left text-gray-600 dark:border-gray-700 dark:text-gray-400">
								<th class="pb-2 pr-4">{groupBy === 'model' ? 'Model' : 'User'}</th>
								<th class="pb-2 pr-4 text-right">Requests</th>
								<th class="pb-2 pr-4 text-right">Total cost</th>
								<th class="pb-2 pr-4 text-right">Avg cost</th>
								<th class="pb-2 text-right">Total tokens</th>
							</tr>
						</thead>
						<tbody>
							{#each costsQuery.data?.breakdown ?? [] as row}
								<tr class="border-b border-gray-200/50 dark:border-gray-700/50">
									<td class="py-2 pr-4 font-mono text-indigo-600 dark:text-indigo-300">{row.dimension}</td>
									<td class="py-2 pr-4 text-right">{row.request_count.toLocaleString()}</td>
									<td class="py-2 pr-4 text-right">${row.total_cost.toFixed(4)}</td>
									<td class="py-2 pr-4 text-right">${row.avg_cost.toFixed(6)}</td>
									<td class="py-2 text-right">{row.total_tokens.toLocaleString()}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	{:else}
		<!-- Latency controls -->
		<div class="flex flex-wrap gap-3 rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
			<div class="flex items-center gap-2">
				<label for="latency-range-start" class="text-sm text-gray-600 dark:text-gray-400">Start</label>
				<input id="latency-range-start" bind:value={rangeStart} type="datetime-local" class={inputClass} />
			</div>
			<div class="flex items-center gap-2">
				<label for="latency-range-end" class="text-sm text-gray-600 dark:text-gray-400">End</label>
				<input id="latency-range-end" bind:value={rangeEnd} type="datetime-local" class={inputClass} />
			</div>
			<div class="flex items-center gap-2">
				<label for="latency-group-by" class="text-sm text-gray-600 dark:text-gray-400">Group by</label>
				<select id="latency-group-by" bind:value={latencyGroupBy} class={inputClass}>
					<option value="">All</option>
					<option value="model">Model</option>
					<option value="user">User</option>
				</select>
			</div>
		</div>

		{#if latencyQuery.isLoading}
			<div class="py-12 text-center"><Spinner /></div>
		{:else if latencyQuery.isError}
			<ErrorAlert message="Failed to load latency data" onRetry={() => latencyQuery.refetch()} />
		{:else if latencyData.length === 0}
			<div class="py-12 text-center text-gray-500">No data for this range</div>
		{:else}
			<div class="rounded-lg bg-gray-100 p-4 dark:bg-gray-800">
				<h2 class="mb-4 text-lg font-semibold">Latency distribution over time</h2>
				<Chart.Container config={latencyChartConfig} class="min-h-[300px] w-full">
					<LineChart
						data={latencyData}
						x="hour"
						xScale={scaleBand()}
						axis="x"
						legend
						series={[
							{ key: 'avg', label: latencyChartConfig.avg.label, color: latencyChartConfig.avg.color },
							{ key: 'p95', label: latencyChartConfig.p95.label, color: latencyChartConfig.p95.color },
							{ key: 'p99', label: latencyChartConfig.p99.label, color: latencyChartConfig.p99.color }
						]}
						props={{
							spline: { strokeWidth: 2 },
							xAxis: { format: latencyTickFormat }
						}}
					>
						{#snippet tooltip()}
							<Chart.Tooltip />
						{/snippet}
					</LineChart>
				</Chart.Container>
			</div>
		{/if}
	{/if}
</div>

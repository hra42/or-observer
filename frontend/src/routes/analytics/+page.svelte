<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchCostsBreakdown, fetchMetricsHourly, type MetricRow, type BreakdownRow } from '$lib/api';
	import {
		BarChart,
		Bar,
		XAxis,
		YAxis,
		CartesianGrid,
		Legend,
		ResponsiveContainer,
		ComposedChart,
		Line
	} from 'recharts';
	import { Chart_Tooltip as Tooltip } from '$lib/recharts';
	import Spinner from '$lib/components/Spinner.svelte';
	import ErrorAlert from '$lib/components/ErrorAlert.svelte';
	import { isDark } from '$lib/stores/theme.svelte';

	let groupBy = $state<'model' | 'user'>('model');
	let costPeriod = $state<'hourly' | 'daily' | 'overall'>('daily');
	let activeTab = $state<'costs' | 'latency'>('costs');
	let latencyGroupBy = $state<'model' | 'user' | ''>('');

	// Convert a datetime-local string (YYYY-MM-DDTHH:mm, no tz) to RFC3339 UTC.
	function toRFC3339(local: string): string {
		if (!local) return '';
		return new Date(local).toISOString();
	}

	const now = new Date();
	// datetime-local inputs expect "YYYY-MM-DDTHH:mm" format.
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
				costRangeEnd ? toRFC3339(costRangeEnd) : undefined
			)
	}));

	const latencyQuery = createQuery(() => ({
		queryKey: ['metrics', 'latency', rangeStart, rangeEnd, latencyGroupBy],
		queryFn: () => fetchMetricsHourly(toRFC3339(rangeStart), toRFC3339(rangeEnd), latencyGroupBy)
	}));

	let latencyData = $derived(
		(latencyQuery.data?.metrics ?? [])
			.map((m: MetricRow) => ({
				hour: new Date(m.hour).toLocaleDateString([], {
					month: 'short',
					day: 'numeric',
					hour: '2-digit'
				}),
				avg: +m.avg_latency_ms.toFixed(1),
				p95: +m.p95_latency_ms.toFixed(1),
				p99: +m.p99_latency_ms.toFixed(1)
			}))
			.reverse()
	);

	let costData = $derived(
		(costsQuery.data?.breakdown ?? []).map((b: BreakdownRow) => ({
			name: b.dimension.length > 20 ? b.dimension.slice(0, 20) + '…' : b.dimension,
			cost: +b.total_cost.toFixed(4),
			requests: b.request_count
		}))
	);

	let gridStroke = $derived(isDark() ? '#374151' : '#e5e7eb');
	let axisStroke = $derived(isDark() ? '#9ca3af' : '#6b7280');

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
				<ResponsiveContainer width="100%" height={280}>
					<BarChart data={costData} margin={{ top: 4, right: 16, left: 0, bottom: 40 }}>
						<CartesianGrid strokeDasharray="3 3" stroke={gridStroke} />
						<XAxis
							dataKey="name"
							stroke={axisStroke}
							tick={{ fontSize: 10, angle: -30, textAnchor: 'end' }}
						/>
						<YAxis stroke={axisStroke} tick={{ fontSize: 11 }} />
						<Tooltip />
						<Legend />
						<Bar dataKey="cost" name="Cost ($)" fill="#818cf8" />
					</BarChart>
				</ResponsiveContainer>
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
				<ResponsiveContainer width="100%" height={300}>
					<ComposedChart data={latencyData}>
						<CartesianGrid strokeDasharray="3 3" stroke={gridStroke} />
						<XAxis
							dataKey="hour"
							stroke={axisStroke}
							tick={{ fontSize: 10, angle: -30, textAnchor: 'end' }}
							height={50}
						/>
						<YAxis stroke={axisStroke} tick={{ fontSize: 11 }} unit="ms" />
						<Tooltip />
						<Legend />
						<Bar dataKey="avg" name="Avg (ms)" fill="#6366f1" opacity={0.6} />
						<Line
							type="monotone"
							dataKey="p95"
							name="P95 (ms)"
							stroke="#f59e0b"
							dot={false}
							strokeWidth={2}
						/>
						<Line
							type="monotone"
							dataKey="p99"
							name="P99 (ms)"
							stroke="#ef4444"
							dot={false}
							strokeWidth={2}
						/>
					</ComposedChart>
				</ResponsiveContainer>
			</div>
		{/if}
	{/if}
</div>

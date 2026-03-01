<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { fetchTraces, type TraceRow } from '$lib/api';
	import Spinner from '$lib/components/Spinner.svelte';
	import ErrorAlert from '$lib/components/ErrorAlert.svelte';
	import TraceMessages from '$lib/components/TraceMessages.svelte';
	import TraceMetadata from '$lib/components/TraceMetadata.svelte';

	let apiKey = $derived(page.data.apiKey ?? '');

	let userID = $state('');
	let model = $state('');
	let startDate = $state('');
	let endDate = $state('');
	let limit = $state(50);
	let offset = $state(0);
	let selected = $state<TraceRow | null>(null);
	let activeTab = $state<'messages' | 'metadata'>('messages');

	let parsedMetadata = $derived<Record<string, unknown>>(
		selected ? (() => { try { return JSON.parse(selected.metadata || '{}'); } catch { return {}; } })() : {}
	);

	function toRFC3339(local: string): string {
		if (!local) return '';
		return new Date(local).toISOString();
	}

	const query = createQuery(() => ({
		queryKey: ['traces', userID, model, startDate, endDate, limit, offset],
		queryFn: () =>
			fetchTraces({
				user_id: userID || undefined,
				model: model || undefined,
				start_date: startDate ? toRFC3339(startDate) : undefined,
				end_date: endDate ? toRFC3339(endDate) : undefined,
				limit,
				offset
			}, apiKey)
	}));

	let total = $derived(query.data?.total ?? 0);
	let traces = $derived(query.data?.traces ?? []);

	function applyFilters() {
		offset = 0;
	}

	function prevPage() {
		if (offset >= limit) offset -= limit;
	}

	function nextPage() {
		if (offset + limit < total) offset += limit;
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleString();
	}
</script>

<div class="space-y-4">
	<h1 class="text-2xl font-bold">Trace Explorer</h1>

	<!-- Filter bar -->
	<form
		onsubmit={(e) => {
			e.preventDefault();
			applyFilters();
		}}
		class="flex flex-wrap gap-3 rounded-lg bg-gray-100 p-4 dark:bg-gray-800"
	>
		<input
			bind:value={userID}
			placeholder="User ID"
			aria-label="User ID"
			class="w-full rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:w-auto"
		/>
		<input
			bind:value={model}
			placeholder="Model"
			aria-label="Model"
			class="w-full rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:w-auto"
		/>
		<input
			bind:value={startDate}
			type="datetime-local"
			aria-label="Start date"
			class="w-full rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:w-auto"
		/>
		<input
			bind:value={endDate}
			type="datetime-local"
			aria-label="End date"
			class="w-full rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:w-auto"
		/>
		<select
			bind:value={limit}
			aria-label="Results per page"
			class="w-full rounded bg-gray-200 px-3 py-1.5 text-sm text-gray-900 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:w-auto"
		>
			<option value={25}>25 / page</option>
			<option value={50}>50 / page</option>
			<option value={100}>100 / page</option>
		</select>
		<button type="submit" class="rounded bg-indigo-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-indigo-500">
			Filter
		</button>
		<button
			type="button"
			onclick={() => {
				userID = '';
				model = '';
				startDate = '';
				endDate = '';
				applyFilters();
			}}
			class="rounded bg-gray-200 px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
		>
			Clear
		</button>
	</form>

	{#if query.isLoading}
		<div class="py-12 text-center"><Spinner /></div>
	{:else if query.isError}
		<ErrorAlert message="Failed to load traces: {query.error?.message}" onRetry={() => query.refetch()} />
	{:else}
		<div class="rounded-lg bg-gray-100 dark:bg-gray-800">
			<div class="border-b border-gray-200 px-4 py-3 text-sm text-gray-600 dark:border-gray-700 dark:text-gray-400">
				{total.toLocaleString()} traces
				{#if total > 0} — showing {offset + 1}–{Math.min(offset + limit, total)}{/if}
			</div>
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-200 text-left text-gray-600 dark:border-gray-700 dark:text-gray-400">
							<th class="px-4 py-2">Trace ID</th>
							<th class="px-4 py-2">Model</th>
							<th class="px-4 py-2 text-right">Tokens</th>
							<th class="px-4 py-2 text-right">Cost</th>
							<th class="px-4 py-2 text-right">Duration</th>
							<th class="px-4 py-2">User</th>
							<th class="px-4 py-2">Time</th>
						</tr>
					</thead>
					<tbody>
						{#each traces as trace}
							<tr
								class="cursor-pointer border-b border-gray-200/50 transition-colors hover:bg-gray-200/50 dark:border-gray-700/50 dark:hover:bg-gray-700/50"
								onclick={() => { selected = trace; activeTab = 'messages'; }}
								onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); selected = trace; activeTab = 'messages'; } }}
								tabindex="0"
								role="button"
							>
								<td class="px-4 py-2 font-mono text-xs text-indigo-600 dark:text-indigo-300"
									>{trace.trace_id.slice(0, 8)}…</td
								>
								<td class="px-4 py-2 text-gray-700 dark:text-gray-300">{trace.model || '—'}</td>
								<td class="px-4 py-2 text-right">{trace.total_tokens.toLocaleString()}</td>
								<td class="px-4 py-2 text-right">${trace.cost.toFixed(6)}</td>
								<td class="px-4 py-2 text-right">{trace.duration_ms}ms</td>
								<td class="px-4 py-2 text-gray-600 dark:text-gray-400">{trace.user_id || '—'}</td>
								<td class="px-4 py-2 text-xs text-gray-600 dark:text-gray-400">{formatDate(trace.created_at)}</td>
							</tr>
						{:else}
							<tr>
								<td colspan="7" class="py-12 text-center text-gray-500">No traces found</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			{#if total > limit}
				<div class="flex items-center justify-between border-t border-gray-200 px-4 py-3 dark:border-gray-700">
					<button
						onclick={prevPage}
						disabled={offset === 0}
						class="rounded bg-gray-200 px-3 py-1 text-sm disabled:opacity-40 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600"
					>
						← Prev
					</button>
					<span class="text-sm text-gray-600 dark:text-gray-400">
						Page {Math.floor(offset / limit) + 1} of {Math.ceil(total / limit)}
					</span>
					<button
						onclick={nextPage}
						disabled={offset + limit >= total}
						class="rounded bg-gray-200 px-3 py-1 text-sm disabled:opacity-40 hover:bg-gray-300 dark:bg-gray-700 dark:hover:bg-gray-600"
					>
						Next →
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Detail modal -->
{#if selected}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
		onclick={() => (selected = null)}
		onkeydown={(e) => { if (e.key === 'Escape') selected = null; }}
		role="button"
		tabindex="-1"
	>
		<div
			class="max-h-[85vh] w-full max-w-4xl overflow-y-auto rounded-lg bg-white shadow-xl dark:bg-gray-800"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			tabindex="-1"
		>
			<div class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700">
				<h2 class="font-semibold">Trace detail</h2>
				<button onclick={() => (selected = null)} class="text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white">✕</button>
			</div>
			<div class="space-y-4 p-6 text-sm">
				<div class="grid grid-cols-2 gap-4 sm:grid-cols-3">
					{#each [
						['Trace ID', selected.trace_id],
						['Span ID', selected.span_id],
						['Span name', selected.span_name],
						['Model', selected.model],
						['User ID', selected.user_id],
						['Session ID', selected.session_id],
						['Prompt tokens', String(selected.prompt_tokens)],
						['Completion tokens', String(selected.completion_tokens)],
						['Total tokens', String(selected.total_tokens)],
						['Cost', `$${selected.cost.toFixed(8)}`],
						['Duration', `${selected.duration_ms}ms`],
						['Created at', formatDate(selected.created_at)],
					] as [label, value]}
						<div>
							<p class="text-gray-600 dark:text-gray-400">{label}</p>
							<p class="font-mono">{value || '—'}</p>
						</div>
					{/each}
				</div>

				<!-- Tabs -->
				<div class="border-b border-gray-200 dark:border-gray-700">
					<div class="flex gap-4">
						<button
							onclick={() => (activeTab = 'messages')}
							class="border-b-2 px-1 pb-2 text-sm font-medium transition-colors {activeTab === 'messages' ? 'border-indigo-500 text-indigo-600 dark:text-indigo-400' : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'}"
						>
							Messages
						</button>
						<button
							onclick={() => (activeTab = 'metadata')}
							class="border-b-2 px-1 pb-2 text-sm font-medium transition-colors {activeTab === 'metadata' ? 'border-indigo-500 text-indigo-600 dark:text-indigo-400' : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'}"
						>
							Metadata
						</button>
					</div>
				</div>

				{#if activeTab === 'messages'}
					<TraceMessages metadata={parsedMetadata} />
				{:else}
					<TraceMetadata metadata={parsedMetadata} />
				{/if}
			</div>
		</div>
	</div>
{/if}

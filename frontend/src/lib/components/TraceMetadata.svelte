<script lang="ts">
	interface Props {
		metadata: Record<string, unknown>;
	}

	interface MetadataGroup {
		label: string;
		prefix: string;
		entries: [string, unknown][];
	}

	let { metadata }: Props = $props();

	const consumedKeys = new Set([
		'gen_ai.prompt',
		'gen_ai.completion',
		'span.input',
		'span.output',
		'trace.input',
		'trace.output',
		'span.type',
		'span.level'
	]);

	const consumedPrefixes = [
		'gen_ai.request.',
		'gen_ai.usage.',
		'gen_ai.response.',
		'openrouter.'
	];

	function isConsumed(key: string): boolean {
		if (consumedKeys.has(key)) return true;
		return consumedPrefixes.some((p) => key.startsWith(p));
	}

	function groupEntries(meta: Record<string, unknown>): MetadataGroup[] {
		const groups: MetadataGroup[] = [
			{ label: 'Model Parameters', prefix: 'gen_ai.request.', entries: [] },
			{ label: 'Usage & Cost', prefix: 'gen_ai.usage.', entries: [] },
			{ label: 'Response Info', prefix: 'gen_ai.response.', entries: [] },
			{ label: 'Provider Info', prefix: 'openrouter.', entries: [] }
		];
		const other: [string, unknown][] = [];

		for (const [key, value] of Object.entries(meta)) {
			if (consumedKeys.has(key)) continue;

			let matched = false;
			for (const group of groups) {
				if (key.startsWith(group.prefix)) {
					group.entries.push([key.slice(group.prefix.length), value]);
					matched = true;
					break;
				}
			}

			if (!matched) {
				// Also categorize gen_ai.provider.name, gen_ai.system, gen_ai.operation.name into provider
				if (key === 'gen_ai.provider.name' || key === 'gen_ai.system' || key === 'gen_ai.operation.name') {
					groups[3].entries.push([key.replace('gen_ai.', ''), value]);
				} else {
					other.push([key, value]);
				}
			}
		}

		if (other.length > 0) {
			groups.push({ label: 'Other', prefix: '', entries: other });
		}

		return groups.filter((g) => g.entries.length > 0);
	}

	let groups = $derived(groupEntries(metadata));

	let collapsed = $state<Record<string, boolean>>({});

	function toggle(label: string) {
		collapsed[label] = !collapsed[label];
	}

	function formatValue(val: unknown): string {
		if (val === null || val === undefined) return '—';
		if (typeof val === 'object') return JSON.stringify(val, null, 2);
		return String(val);
	}
</script>

{#if groups.length > 0}
	<div class="space-y-3">
		{#each groups as group}
			<div class="rounded-lg border border-gray-200 dark:border-gray-700">
				<button
					onclick={() => toggle(group.label)}
					class="flex w-full items-center justify-between px-4 py-2 text-left text-sm font-medium text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-700/50"
				>
					<span>{group.label}</span>
					<span class="text-xs text-gray-400">{collapsed[group.label] ? '▸' : '▾'} {group.entries.length}</span>
				</button>
				{#if !collapsed[group.label]}
					<div class="border-t border-gray-200 dark:border-gray-700">
						<dl class="divide-y divide-gray-100 dark:divide-gray-700/50">
							{#each group.entries as [key, value]}
								<div class="grid grid-cols-3 gap-2 px-4 py-2">
									<dt class="text-xs font-medium text-gray-500 dark:text-gray-400">{key}</dt>
									<dd class="col-span-2 break-all font-mono text-xs text-gray-800 dark:text-gray-200">
										{formatValue(value)}
									</dd>
								</div>
							{/each}
						</dl>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{:else}
	<p class="text-sm text-gray-500 dark:text-gray-400">No metadata attributes found.</p>
{/if}

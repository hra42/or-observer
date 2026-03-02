<script lang="ts">
	interface Props {
		metadata: Record<string, unknown>;
	}

	interface ChatMessage {
		role: string;
		content: string;
	}

	let { metadata }: Props = $props();

	const roleBadgeClasses: Record<string, string> = {
		system: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200',
		user: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
		assistant: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
		tool: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200'
	};

	function badgeClass(role: string): string {
		return roleBadgeClasses[role] ?? 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200';
	}

	function tryParseJSON(val: unknown): unknown | null {
		if (typeof val === 'string') {
			try {
				return JSON.parse(val);
			} catch {
				return null;
			}
		}
		if (typeof val === 'object' && val !== null) return val;
		return null;
	}

	function extractInputMessages(meta: Record<string, unknown>): ChatMessage[] {
		for (const key of ['gen_ai.prompt', 'span.input', 'trace.input']) {
			const raw = meta[key];
			if (raw === undefined || raw === null || raw === '') continue;

			const parsed = tryParseJSON(raw);

			// Parsed as a JSON object — check structured formats
			if (parsed && typeof parsed === 'object') {
				const obj = parsed as Record<string, unknown>;
				// { messages: [{role, content}, ...] }
				if (Array.isArray(obj.messages)) {
					return obj.messages.filter(
						(m: unknown): m is ChatMessage =>
							typeof m === 'object' && m !== null && 'role' in m && 'content' in m
					);
				}
				// Direct array
				if (Array.isArray(parsed)) {
					const msgs = (parsed as unknown[]).filter(
						(m: unknown): m is ChatMessage =>
							typeof m === 'object' && m !== null && 'role' in m && 'content' in m
					);
					if (msgs.length > 0) return msgs;
				}
				// { role, content } directly
				if ('role' in obj && 'content' in obj) {
					return [{ role: String(obj.role), content: String(obj.content) }];
				}
			}

			// Plain string content
			if (typeof raw === 'string') {
				const content = typeof parsed === 'string' ? parsed : raw;
				return [{ role: 'user', content }];
			}
		}
		return [];
	}

	function extractOutputMessages(meta: Record<string, unknown>): ChatMessage[] {
		for (const key of ['gen_ai.completion', 'span.output', 'trace.output']) {
			const raw = meta[key];
			if (raw === undefined || raw === null || raw === '') continue;

			const parsed = tryParseJSON(raw);

			// Parsed as a JSON object — check structured formats
			if (parsed && typeof parsed === 'object') {
				const obj = parsed as Record<string, unknown>;
				// { choices: [{message: {role, content}}] }
				if (Array.isArray(obj.choices)) {
					return obj.choices
						.map((c: Record<string, unknown>) => c.message as ChatMessage | undefined)
						.filter((m): m is ChatMessage => !!m && typeof m.content === 'string');
				}
				// Array of {role, content}
				if (Array.isArray(parsed)) {
					const msgs = (parsed as unknown[]).filter(
						(m: unknown): m is ChatMessage =>
							typeof m === 'object' && m !== null && 'role' in m && 'content' in m
					);
					if (msgs.length > 0) return msgs;
				}
				// { role, content } directly
				if ('role' in obj && 'content' in obj) {
					return [{ role: String(obj.role), content: String(obj.content) }];
				}
			}

			// Plain string content (not valid JSON, or a JSON string primitive)
			if (typeof raw === 'string') {
				const content = typeof parsed === 'string' ? parsed : raw;
				return [{ role: 'assistant', content }];
			}
		}
		return [];
	}

	let inputMessages = $derived(extractInputMessages(metadata));
	let outputMessages = $derived(extractOutputMessages(metadata));
	let hasMessages = $derived(inputMessages.length > 0 || outputMessages.length > 0);
</script>

{#if hasMessages}
	<div class="space-y-4">
		{#if inputMessages.length > 0}
			<div>
				<h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Input</h3>
				<div class="space-y-2">
					{#each inputMessages as msg}
						<div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-900">
							<span class="mb-1 inline-block rounded px-2 py-0.5 text-xs font-medium {badgeClass(msg.role)}">
								{msg.role}
							</span>
							<p class="mt-1 whitespace-pre-wrap text-sm text-gray-800 dark:text-gray-200">{msg.content}</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		{#if outputMessages.length > 0}
			<div>
				<h3 class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Output</h3>
				<div class="space-y-2">
					{#each outputMessages as msg}
						<div class="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900 dark:bg-green-950">
							<span class="mb-1 inline-block rounded px-2 py-0.5 text-xs font-medium {badgeClass(msg.role)}">
								{msg.role}
							</span>
							<p class="mt-1 whitespace-pre-wrap text-sm text-gray-800 dark:text-gray-200">{msg.content}</p>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{:else}
	<p class="text-sm text-gray-500 dark:text-gray-400">No message data found in metadata.</p>
{/if}

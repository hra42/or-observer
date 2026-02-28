const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

export interface TraceRow {
	id: string;
	trace_id: string;
	span_id: string;
	span_name: string;
	model: string;
	prompt_tokens: number;
	completion_tokens: number;
	total_tokens: number;
	cost: number;
	duration_ms: number;
	user_id: string;
	session_id: string;
	metadata: string;
	created_at: string;
}

export interface TracesResponse {
	total: number;
	limit: number;
	offset: number;
	traces: TraceRow[];
}

export interface TracesQuery {
	user_id?: string;
	model?: string;
	start_date?: string;
	end_date?: string;
	limit?: number;
	offset?: number;
}

export async function fetchTraces(params: TracesQuery = {}): Promise<TracesResponse> {
	const url = new URL('/api/traces', API_BASE);
	Object.entries(params).forEach(([k, v]) => {
		if (v != null && v !== '') url.searchParams.set(k, String(v));
	});
	const res = await fetch(url.toString());
	if (!res.ok) throw new Error(`/api/traces: ${res.status}`);
	return res.json();
}

export interface MetricRow {
	hour: string;
	dimension: string;
	request_count: number;
	avg_latency_ms: number;
	p95_latency_ms: number;
	p99_latency_ms: number;
	total_tokens: number;
	total_cost: number;
	error_count: number;
}

export interface MetricsResponse {
	metrics: MetricRow[];
}

export async function fetchMetricsHourly(
	start?: string,
	end?: string,
	groupBy?: 'model' | 'user' | ''
): Promise<MetricsResponse> {
	const url = new URL('/api/metrics/hourly', API_BASE);
	if (start) url.searchParams.set('start', start);
	if (end) url.searchParams.set('end', end);
	if (groupBy) url.searchParams.set('groupBy', groupBy);
	const res = await fetch(url.toString());
	if (!res.ok) throw new Error(`/api/metrics/hourly: ${res.status}`);
	return res.json();
}

export interface BreakdownRow {
	dimension: string;
	request_count: number;
	total_cost: number;
	avg_cost: number;
	total_tokens: number;
}

export interface CostsResponse {
	period: string;
	group_by: string;
	breakdown: BreakdownRow[];
}

export async function fetchCostsBreakdown(
	groupBy: 'model' | 'user' = 'model',
	period: 'hourly' | 'daily' | 'overall' = 'daily',
	start?: string,
	end?: string
): Promise<CostsResponse> {
	const url = new URL('/api/costs/breakdown', API_BASE);
	url.searchParams.set('groupBy', groupBy);
	url.searchParams.set('period', period);
	if (start) url.searchParams.set('start', start);
	if (end) url.searchParams.set('end', end);
	const res = await fetch(url.toString());
	if (!res.ok) throw new Error(`/api/costs/breakdown: ${res.status}`);
	return res.json();
}

export async function fetchHealth(): Promise<{
	status: string;
	database: string;
	traces_ingested: number;
	uptime_seconds: number;
}> {
	const res = await fetch(`${API_BASE}/health`);
	if (!res.ok) throw new Error(`/health: ${res.status}`);
	return res.json();
}

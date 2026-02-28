// Re-export recharts components with `any` casts to work around Svelte 5 type incompatibility.
// Recharts hasn't updated its typedefs for Svelte 5's Component signature yet.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export {
	LineChart,
	Line,
	BarChart,
	Bar,
	ComposedChart,
	XAxis,
	YAxis,
	CartesianGrid,
	Tooltip as _Tooltip,
	Legend,
	ResponsiveContainer
} from 'recharts';

import { Tooltip } from 'recharts';
// Cast to any so Svelte 5 template type-checker is happy
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const Chart_Tooltip = Tooltip as any;

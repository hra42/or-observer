// Re-export recharts components with `any` casts to work around Svelte 5 type incompatibility.
// Recharts components are React forwardRef objects, not callable functions.
// Without the `as any` cast, Svelte 5 SSR tries to call them as functions and crashes.
/* eslint-disable @typescript-eslint/no-explicit-any */
import {
	LineChart as _LineChart,
	Line as _Line,
	BarChart as _BarChart,
	Bar as _Bar,
	ComposedChart as _ComposedChart,
	XAxis as _XAxis,
	YAxis as _YAxis,
	CartesianGrid as _CartesianGrid,
	Tooltip as _Tooltip,
	Legend as _Legend,
	ResponsiveContainer as _ResponsiveContainer
} from 'recharts';

export const LineChart = _LineChart as any;
export const Line = _Line as any;
export const BarChart = _BarChart as any;
export const Bar = _Bar as any;
export const ComposedChart = _ComposedChart as any;
export const XAxis = _XAxis as any;
export const YAxis = _YAxis as any;
export const CartesianGrid = _CartesianGrid as any;
export const Chart_Tooltip = _Tooltip as any;
export const Legend = _Legend as any;
export const ResponsiveContainer = _ResponsiveContainer as any;

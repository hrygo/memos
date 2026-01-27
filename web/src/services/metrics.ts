export interface MetricsOverview {
  total_requests: number;
  success_rate: number;
  avg_latency_ms: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  error_count: number;
  time_range: string;
  is_mock: boolean; // 标记是否为模拟数据
}

export const metricsService = {
  getOverview: async (timeRange = "24h"): Promise<MetricsOverview> => {
    const response = await fetch(`/api/v1/system/metrics/overview?range=${timeRange}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch metrics: ${response.statusText}`);
    }
    return response.json();
  },
};

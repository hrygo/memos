import { useEffect, useState } from "react";
import { Activity, BarChart3, Clock, TrendingUp } from "lucide-react";
import { useTranslate } from "@/utils/i18n";
import { cn } from "@/lib/utils";
import { metricsService, type MetricsOverview } from "@/services/metrics";

export { MetricsDashboard };

function MetricsDashboard() {
  const t = useTranslate();
  const [timeRange, setTimeRange] = useState("24h");
  const [metrics, setMetrics] = useState<MetricsOverview | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setIsLoading(true);
    setError(null);
    metricsService
      .getOverview(timeRange)
      .then(setMetrics)
      .catch((err) => {
        console.error("Failed to fetch metrics:", err);
        setError(err instanceof Error ? err.message : "Failed to load metrics");
      })
      .finally(() => setIsLoading(false));
  }, [timeRange]);

  const timeRanges = [
    { value: "1h", label: "1H" },
    { value: "24h", label: "24H" },
    { value: "7d", label: "7D" },
    { value: "30d", label: "30D" },
  ];

  // Translation keys (must exist in i18n files)
  const tTitle = t("setting.metrics-section.overview");
  const tRequests = t("setting.metrics-section.requests");
  const tSuccessRate = t("setting.metrics-section.success-rate");
  const tLatency = t("setting.metrics-section.latency");
  const tP95 = t("setting.metrics-section.p95");
  const tErrors = t("setting.metrics-section.errors");

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <BarChart3 className="w-5 h-5 text-muted-foreground" />
          <h3 className="text-lg font-semibold">{tTitle}</h3>
        </div>
        <div className="flex gap-1 bg-muted rounded-lg p-1">
          {timeRanges.map((range) => (
            <button
              key={range.value}
              onClick={() => setTimeRange(range.value)}
              className={cn(
                "px-3 py-1 text-xs font-medium rounded-md transition-colors",
                timeRange === range.value
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {range.label}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-xl">
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="h-28 bg-muted/30 rounded-xl animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <MetricCard
            title={tRequests}
            value={metrics?.total_requests ?? 0}
            icon={<Activity className="w-4 h-4" />}
            color="text-blue-600"
            bgColor="bg-blue-500/10"
          />
          <MetricCard
            title={tSuccessRate}
            value={`${((metrics?.success_rate ?? 0) * 100).toFixed(1)}%`}
            icon={<TrendingUp className="w-4 h-4" />}
            color="text-green-600"
            bgColor="bg-green-500/10"
          />
          <MetricCard
            title={tLatency}
            value={`${metrics?.p50_latency_ms ?? 0}ms`}
            icon={<Clock className="w-4 h-4" />}
            color="text-amber-600"
            bgColor="bg-amber-500/10"
          />
          <MetricCard
            title={tP95}
            value={`${metrics?.p95_latency_ms ?? 0}ms`}
            icon={<Clock className="w-4 h-4" />}
            color="text-purple-600"
            bgColor="bg-purple-500/10"
          />
        </div>
      )}

      {metrics && metrics.error_count > 0 && (
        <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-xl">
          <p className="text-sm text-destructive">
            {tErrors}: {metrics.error_count}
          </p>
        </div>
      )}

      {metrics && metrics.is_mock && (
        <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-xl">
          <p className="text-xs text-amber-600 dark:text-amber-400">
            Mock data - Metrics service not yet implemented
          </p>
        </div>
      )}
    </div>
  );
}

interface MetricCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
  bgColor: string;
}

function MetricCard({ title, value, icon, color, bgColor }: MetricCardProps) {
  return (
    <div className="bg-card border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between mb-2">
        <span className="text-sm text-muted-foreground">{title}</span>
        <div className={cn("p-1.5 rounded-lg", bgColor)}>
          <div className={color}>{icon}</div>
        </div>
      </div>
      <p className="text-2xl font-semibold text-foreground">{value}</p>
    </div>
  );
}

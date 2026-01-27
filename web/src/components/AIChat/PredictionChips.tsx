import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  CalendarDays,
  CalendarPlus,
  Clock,
  FileSearch,
  LineChart,
  Lightbulb,
  PenSquare,
  Bell,
  Sparkles,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

// Action types matching backend prediction engine
type ActionType =
  | "view_week_schedule"
  | "view_tomorrow"
  | "create_schedule"
  | "set_reminder"
  | "search_related"
  | "view_weekly_report"
  | "monthly_review"
  | "quick_note";

interface Prediction {
  type: "action" | "query" | "reminder";
  label: string;
  confidence: number;
  action: ActionType;
  payload?: unknown;
  reason?: string;
}

interface PredictionChipsProps {
  predictions: Prediction[];
  onSelect: (prediction: Prediction) => void;
  loading?: boolean;
  className?: string;
  maxVisible?: number;
}

// Icon mapping for action types
const ACTION_ICONS: Record<ActionType, LucideIcon> = {
  view_week_schedule: CalendarDays,
  view_tomorrow: Clock,
  create_schedule: CalendarPlus,
  set_reminder: Bell,
  search_related: FileSearch,
  view_weekly_report: LineChart,
  monthly_review: LineChart,
  quick_note: PenSquare,
};

// Color mapping based on action category
const ACTION_COLORS: Record<ActionType, string> = {
  view_week_schedule: "bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800",
  view_tomorrow: "bg-cyan-50 text-cyan-700 border-cyan-200 hover:bg-cyan-100 dark:bg-cyan-900/20 dark:text-cyan-300 dark:border-cyan-800",
  create_schedule: "bg-green-50 text-green-700 border-green-200 hover:bg-green-100 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800",
  set_reminder: "bg-amber-50 text-amber-700 border-amber-200 hover:bg-amber-100 dark:bg-amber-900/20 dark:text-amber-300 dark:border-amber-800",
  search_related: "bg-purple-50 text-purple-700 border-purple-200 hover:bg-purple-100 dark:bg-purple-900/20 dark:text-purple-300 dark:border-purple-800",
  view_weekly_report: "bg-indigo-50 text-indigo-700 border-indigo-200 hover:bg-indigo-100 dark:bg-indigo-900/20 dark:text-indigo-300 dark:border-indigo-800",
  monthly_review: "bg-violet-50 text-violet-700 border-violet-200 hover:bg-violet-100 dark:bg-violet-900/20 dark:text-violet-300 dark:border-violet-800",
  quick_note: "bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100 dark:bg-emerald-900/20 dark:text-emerald-300 dark:border-emerald-800",
};

export function PredictionChips({
  predictions,
  onSelect,
  loading = false,
  className,
  maxVisible = 4,
}: PredictionChipsProps) {
  const { t } = useTranslation();
  const [visiblePredictions, setVisiblePredictions] = useState<Prediction[]>([]);
  const [isAnimating, setIsAnimating] = useState(false);

  useEffect(() => {
    if (predictions.length > 0) {
      setIsAnimating(true);
      // Stagger animation
      const sorted = [...predictions]
        .sort((a, b) => b.confidence - a.confidence)
        .slice(0, maxVisible);
      
      setVisiblePredictions([]);
      sorted.forEach((pred, index) => {
        setTimeout(() => {
          setVisiblePredictions((prev) => [...prev, pred]);
          if (index === sorted.length - 1) {
            setIsAnimating(false);
          }
        }, index * 100);
      });
    } else {
      setVisiblePredictions([]);
    }
  }, [predictions, maxVisible]);

  const handleSelect = useCallback(
    (prediction: Prediction) => {
      onSelect(prediction);
    },
    [onSelect]
  );

  const getActionLabel = (action: ActionType): string => {
    const labelMap: Record<ActionType, string> = {
      view_week_schedule: t("ai.prediction.view-week"),
      view_tomorrow: t("ai.prediction.view-tomorrow"),
      create_schedule: t("ai.prediction.create-schedule"),
      set_reminder: t("ai.prediction.set-reminder"),
      search_related: t("ai.prediction.search-related"),
      view_weekly_report: t("ai.prediction.weekly-report"),
      monthly_review: t("ai.prediction.monthly-review"),
      quick_note: t("ai.prediction.quick-note"),
    };
    return labelMap[action] || action;
  };

  if (loading) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-8 w-24 rounded-full bg-muted animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (visiblePredictions.length === 0 && !isAnimating) {
    return null;
  }

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mr-1">
        <Sparkles className="w-3.5 h-3.5" />
        <span>{t("ai.prediction.suggested")}</span>
      </div>
      {visiblePredictions.map((prediction, index) => {
        const Icon = ACTION_ICONS[prediction.action] || Lightbulb;
        const colorClass = ACTION_COLORS[prediction.action] || ACTION_COLORS.quick_note;

        return (
          <button
            key={`${prediction.action}-${index}`}
            onClick={() => handleSelect(prediction)}
            className={cn(
              "inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium",
              "border transition-all duration-200 ease-out",
              "hover:shadow-sm active:scale-95",
              "animate-in fade-in slide-in-from-bottom-2",
              colorClass
            )}
            style={{
              animationDelay: `${index * 50}ms`,
              animationDuration: "200ms",
            }}
            title={prediction.reason}
          >
            <Icon className="w-3.5 h-3.5" />
            <span>{prediction.label || getActionLabel(prediction.action)}</span>
            {prediction.confidence >= 0.8 && (
              <span className="ml-0.5 w-1.5 h-1.5 rounded-full bg-current opacity-60" />
            )}
          </button>
        );
      })}
    </div>
  );
}

// Hook for fetching predictions
export function usePredictions(userID: number | undefined) {
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetchPredictions = useCallback(async (context?: unknown[]) => {
    if (!userID) return;
    
    setLoading(true);
    setError(null);
    
    try {
      const response = await fetch("/api/v1/ai/predictions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: userID,
          context: context || [],
        }),
      });
      
      if (!response.ok) {
        throw new Error("Failed to fetch predictions");
      }
      
      const data = await response.json();
      setPredictions(data.predictions || []);
    } catch (err) {
      setError(err instanceof Error ? err : new Error("Unknown error"));
      setPredictions([]);
    } finally {
      setLoading(false);
    }
  }, [userID]);

  const clearPredictions = useCallback(() => {
    setPredictions([]);
  }, []);

  return {
    predictions,
    loading,
    error,
    fetchPredictions,
    clearPredictions,
  };
}

// Static predictions for demo/fallback
export function useStaticPredictions() {
  const { t } = useTranslation();
  
  const getTimeBased = useCallback((): Prediction[] => {
    const hour = new Date().getHours();
    const predictions: Prediction[] = [];
    
    // Morning predictions
    if (hour >= 8 && hour < 10) {
      predictions.push({
        type: "action",
        label: t("ai.prediction.view-today"),
        confidence: 0.9,
        action: "view_tomorrow",
        reason: t("ai.prediction.reason-morning"),
      });
    }
    
    // End of week
    const dayOfWeek = new Date().getDay();
    if (dayOfWeek === 5) {
      predictions.push({
        type: "action",
        label: t("ai.prediction.weekly-report"),
        confidence: 0.85,
        action: "view_weekly_report",
        reason: t("ai.prediction.reason-friday"),
      });
    }
    
    // Default suggestions
    predictions.push({
      type: "action",
      label: t("ai.prediction.view-week"),
      confidence: 0.7,
      action: "view_week_schedule",
    });
    
    predictions.push({
      type: "action",
      label: t("ai.prediction.create-schedule"),
      confidence: 0.6,
      action: "create_schedule",
    });
    
    return predictions.slice(0, 4);
  }, [t]);

  return { getTimeBased };
}

export default PredictionChips;

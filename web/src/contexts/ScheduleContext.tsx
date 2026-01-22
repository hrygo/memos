import { createContext, ReactNode, useContext, useState } from "react";
import type { QuickInputPreferences } from "@/components/ScheduleQuickInput/types";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

interface ScheduleContextType {
  selectedDate: string | undefined;
  setSelectedDate: (date: string | undefined) => void;
  // Search state
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  filteredSchedules: Schedule[];
  setFilteredSchedules: (schedules: Schedule[]) => void;
  hasSearchFilter: boolean;
  setHasSearchFilter: (hasFilter: boolean) => void;
  // Quick input state
  quickInputOpen: boolean;
  setQuickInputOpen: (open: boolean) => void;
  quickInputPreferences: QuickInputPreferences;
  updateQuickInputPreferences: (preferences: Partial<QuickInputPreferences>) => void;
}

const ScheduleContext = createContext<ScheduleContextType | undefined>(undefined);

export const useScheduleContext = () => {
  const context = useContext(ScheduleContext);
  if (!context) {
    throw new Error("useScheduleContext must be used within ScheduleProvider");
  }
  return context;
};

interface ScheduleProviderProps {
  children: ReactNode;
}

const DEFAULT_QUICK_INPUT_PREFERENCES: QuickInputPreferences = {
  defaultDuration: 60,
  defaultAllDay: false,
  enableLocalParsing: true,
  enableAIParsing: true,
  parseDebounceMs: 600,
  showTemplates: true,
  collapseOnCreate: true,
};

export const ScheduleProvider = ({ children }: ScheduleProviderProps) => {
  const [selectedDate, setSelectedDate] = useState<string | undefined>();

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [filteredSchedules, setFilteredSchedules] = useState<Schedule[]>([]);
  const [hasSearchFilter, setHasSearchFilter] = useState(false);

  // Quick input state
  const [quickInputOpen, setQuickInputOpen] = useState(false);
  const [quickInputPreferences, setQuickInputPreferences] = useState<QuickInputPreferences>(DEFAULT_QUICK_INPUT_PREFERENCES);

  const updateQuickInputPreferences = (preferences: Partial<QuickInputPreferences>) => {
    setQuickInputPreferences((prev) => ({ ...prev, ...preferences }));
  };

  return (
    <ScheduleContext.Provider
      value={{
        selectedDate,
        setSelectedDate,
        searchQuery,
        setSearchQuery,
        filteredSchedules,
        setFilteredSchedules,
        hasSearchFilter,
        setHasSearchFilter,
        quickInputOpen,
        setQuickInputOpen,
        quickInputPreferences,
        updateQuickInputPreferences,
      }}
    >
      {children}
    </ScheduleContext.Provider>
  );
};

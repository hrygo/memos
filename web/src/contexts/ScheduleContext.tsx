import dayjs from "dayjs";
import { createContext, ReactNode, useContext, useState } from "react";
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

export const ScheduleProvider = ({ children }: ScheduleProviderProps) => {
  const [selectedDate, setSelectedDate] = useState<string | undefined>(dayjs().format("YYYY-MM-DD"));

  // Search state
  const [searchQuery, setSearchQuery] = useState("");
  const [filteredSchedules, setFilteredSchedules] = useState<Schedule[]>([]);
  const [hasSearchFilter, setHasSearchFilter] = useState(false);

  // Quick input state
  const [quickInputOpen, setQuickInputOpen] = useState(false);

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
      }}
    >
      {children}
    </ScheduleContext.Provider>
  );
};

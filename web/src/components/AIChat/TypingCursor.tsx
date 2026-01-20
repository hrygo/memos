import { cn } from "@/lib/utils";

interface TypingCursorProps {
  active?: boolean;
}

const TypingCursor = ({ active = true }: TypingCursorProps) => {
  return <span className={cn("inline-block w-0.5 h-4 bg-primary align-middle ml-0.5", "animate-pulse", !active && "opacity-0")} />;
};

export default TypingCursor;

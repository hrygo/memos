import dayjs from "dayjs";
import { ArrowRight, Clock, Edit3, Loader2, Sparkles, Trash } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "react-hot-toast";
import { useScheduleAgentStreamingChat } from "@/hooks/useScheduleQueries";
import { Popover, PopoverContent, PopoverAnchor } from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useTranslate } from "@/utils/i18n";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";

interface QuickEditPopoverProps {
    schedule: Schedule;
    children: React.ReactNode;
    onOpenChange?: (open: boolean) => void;
}

export function QuickEditPopover({ schedule, children, onOpenChange }: QuickEditPopoverProps) {
    const t = useTranslate();
    const [isOpen, setIsOpen] = useState(false);
    const [input, setInput] = useState("");

    const streamingChat = useScheduleAgentStreamingChat();
    const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleOpenChange = (open: boolean) => {
        setIsOpen(open);
        onOpenChange?.(open);
        if (!open) {
            setInput("");
            streamingChat.reset();
        }
    };

    const handleContextMenu = (e: React.MouseEvent) => {
        e.preventDefault();
        handleOpenChange(true);
    };

    const handleTouchStart = () => {
        longPressTimer.current = setTimeout(() => {
            handleOpenChange(true);
        }, 500); // 500ms long press
    };

    const handleTouchEnd = () => {
        if (longPressTimer.current) {
            clearTimeout(longPressTimer.current);
            longPressTimer.current = null;
        }
    };

    const handleQuickAction = async (action: string) => {
        if (streamingChat.isStreaming) return;

        let prompt = "";
        switch (action) {
            case "delay_30m":
                prompt = "Delay this schedule by 30 minutes";
                setInput(t("schedule.quick.quick-edit.delay-30m") || "Delay 30m");
                break;
            case "move_tomorrow":
                prompt = "Move this schedule to tomorrow same time";
                setInput(t("schedule.quick.quick-edit.move-tomorrow") || "Move to tomorrow");
                break;
            case "cancel":
                prompt = "Cancel this schedule";
                setInput(t("schedule.quick.quick-edit.cancel-schedule") || "Cancel schedule");
                break;
            default:
                return;
        }

        await processEdit(prompt);
    };

    const handleSubmit = async () => {
        if (!input.trim() || streamingChat.isStreaming) return;
        await processEdit(input);
    };

    const processEdit = async (instruction: string) => {
        const timeStr = `${dayjs.unix(Number(schedule.startTs)).format("YYYY-MM-DD HH:mm")}-${dayjs.unix(Number(schedule.endTs)).format("HH:mm")}`;
        const context = `Task: Update schedule "${schedule.title}" (${timeStr}). Instruction: ${instruction}`;

        try {
            const response = await streamingChat.startChat(context);
            // Simple heuristic to check success if the response is not just an error
            if (response && !response.toLowerCase().includes("error")) {
                // If the agent performed tool calls (which it should), we consider it a success or at least processed.
                // In a real scenario, we might want to check the tool events, but here checking text is a proxy.
                toast.success(t("schedule.update-success") || "Schedule updated");
                setIsOpen(false);
            }
        } catch (error) {
            console.error("Quick edit error:", error);
            toast.error(t("schedule.update-failed") || "Failed to update");
        }
    };

    const isProcessing = streamingChat.isStreaming;

    return (
        <Popover open={isOpen} onOpenChange={handleOpenChange}>
            <PopoverAnchor asChild>
                <div
                    onContextMenu={handleContextMenu}
                    onTouchStart={handleTouchStart}
                    onTouchEnd={handleTouchEnd}
                    onTouchMove={handleTouchEnd} // Cancel on drag
                    className="w-full"
                >
                    {children}
                </div>
            </PopoverAnchor>
            <PopoverContent className="w-80 p-0" align="start">
                <div className="p-3 border-b bg-muted/20">
                    <h4 className="font-medium text-sm flex items-center gap-2">
                        <Edit3 className="w-3.5 h-3.5 text-muted-foreground" />
                        {t("schedule.quick.quick-edit.title") || "Quick Edit"}
                    </h4>
                </div>

                <div className="p-3 space-y-3">
                    <Textarea
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        placeholder={t("schedule.quick.quick-edit.placeholder") || "e.g., Delay 30 mins, Rename to..."}
                        className="resize-none min-h-[60px] text-sm"
                        onKeyDown={(e) => {
                            if (e.key === "Enter" && !e.shiftKey) {
                                e.preventDefault();
                                handleSubmit();
                            }
                        }}
                        disabled={isProcessing}
                    />

                    <div className="flex flex-wrap gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs"
                            onClick={() => handleQuickAction("delay_30m")}
                            disabled={isProcessing}
                        >
                            <Clock className="w-3 h-3 mr-1" />
                            +30m
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs"
                            onClick={() => handleQuickAction("move_tomorrow")}
                            disabled={isProcessing}
                        >
                            <ArrowRight className="w-3 h-3 mr-1" />
                            Tomorrow
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            className="h-7 text-xs text-red-500 hover:text-red-600 hover:bg-red-50"
                            onClick={() => handleQuickAction("cancel")}
                            disabled={isProcessing}
                        >
                            <Trash className="w-3 h-3 mr-1" />
                            Cancel
                        </Button>
                    </div>

                    <div className="flex justify-end pt-2">
                        <Button
                            size="sm"
                            onClick={handleSubmit}
                            disabled={!input.trim() || isProcessing}
                            className="h-8"
                        >
                            {isProcessing ? (
                                <Loader2 className="w-3.5 h-3.5 animate-spin mr-1" />
                            ) : (
                                <Sparkles className="w-3.5 h-3.5 mr-1" />
                            )}
                            {t("common.apply") || "Apply"}
                        </Button>
                    </div>

                    {streamingChat.currentStep && (
                        <div className="text-xs text-muted-foreground flex items-center gap-1.5 bg-muted/30 p-1.5 rounded">
                            <Loader2 className="w-3 h-3 animate-spin text-primary" />
                            {streamingChat.currentStep}
                        </div>
                    )}
                </div>
            </PopoverContent>
        </Popover>
    );
}

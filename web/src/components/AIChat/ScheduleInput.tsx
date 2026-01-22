import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQueryClient } from "@tanstack/react-query";
import dayjs from "dayjs";
import { AlertTriangle, Bot, Calendar, Clock, Loader2, MapPin, Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { toast } from "react-hot-toast";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  useCheckConflict,
  useCreateSchedule,
  useDeleteSchedule,
  useScheduleAgentChat,
  useUpdateSchedule,
} from "@/hooks/useScheduleQueries";
import { cn } from "@/lib/utils";
import type { Schedule } from "@/types/proto/api/v1/schedule_service_pb";
import { useTranslate } from "@/utils/i18n";
import { ScheduleConflictAlert } from "./ScheduleConflictAlert";
import { ScheduleErrorBoundary } from "./ScheduleErrorBoundary";

interface ScheduleInputProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialText?: string;
  editSchedule?: Schedule | null;
  onSuccess?: (schedule: Schedule) => void;
}

// Type definitions for conversation history
type ConversationRole = "user" | "assistant";

interface ConversationMessage {
  role: ConversationRole;
  content: string;
}

// Constants
const MAX_CONVERSATION_ROUNDS = 5;
const SUCCESS_AUTO_CLOSE_DELAY_MS = 1500;
const MAX_INPUT_LENGTH = 500;

export const ScheduleInput = ({ open, onOpenChange, initialText = "", editSchedule, onSuccess }: ScheduleInputProps) => {
  const t = useTranslate();
  const queryClient = useQueryClient();
  const createSchedule = useCreateSchedule();
  const updateSchedule = useUpdateSchedule();
  const deleteSchedule = useDeleteSchedule();
  const checkConflict = useCheckConflict();
  const agentChat = useScheduleAgentChat();
  const isEditMode = !!editSchedule;

  const [input, setInput] = useState(initialText);
  const [parsedSchedule, setParsedSchedule] = useState<Schedule | null>(editSchedule || null);
  const [conflicts, setConflicts] = useState<Schedule[]>([]);
  const [showConflictAlert, setShowConflictAlert] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  // Agent mode states
  const [agentResponse, setAgentResponse] = useState<string | null>(null);
  const [isProcessingAgent, setIsProcessingAgent] = useState(false);
  const [conversationHistory, setConversationHistory] = useState<ConversationMessage[]>([]);

  // Ref for auto-close timeout to prevent memory leaks
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout>>();

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (closeTimeoutRef.current) {
        clearTimeout(closeTimeoutRef.current);
      }
    };
  }, []);

  // Initialize with editSchedule when it changes
  useEffect(() => {
    if (editSchedule) {
      setParsedSchedule(editSchedule);
      setInput(editSchedule.title || "");
    } else {
      setParsedSchedule(null);
      setInput(initialText);
    }
  }, [editSchedule, initialText]);

  // Handle Agent-based parsing
  const handleAgentParse = async () => {
    if (!input.trim()) return;

    // Validate input length
    if (input.length > MAX_INPUT_LENGTH) {
      toast.error((t("schedule.input-too-long") as string) || "Input too long");
      return;
    }

    setIsProcessingAgent(true);

    // Limit conversation history to prevent excessive context
    const trimmedHistory = conversationHistory.slice(-MAX_CONVERSATION_ROUNDS * 2);

    // Add user message to history
    const newHistory: ConversationMessage[] = [...trimmedHistory, { role: "user", content: input }];

    try {
      // Build full conversation context using StringBuilder pattern for better performance
      const parts: string[] = [];
      for (const msg of newHistory) {
        parts.push(`${msg.role}: ${msg.content}`);
      }
      const conversationContext = parts.join("\n");

      const result = await agentChat.mutateAsync({
        message: conversationContext,
        userTimezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Shanghai",
      });

      if (result.response) {
        // Add assistant response to history
        const updatedHistory: ConversationMessage[] = [...newHistory, { role: "assistant", content: result.response }];
        setConversationHistory(updatedHistory);
        setAgentResponse(result.response);

        // Check if agent successfully created a schedule
        const createdRegex = new RegExp(t("schedule.quick-input.created-regex") as string, "i");
        const createdSchedule = createdRegex.test(result.response);

        if (createdSchedule) {
          toast.success(t("schedule.quick-input.schedule-created-success") as string);
          // Refresh schedules
          queryClient.invalidateQueries({ queryKey: ["schedules"] });
          // Clear history after successful creation
          setConversationHistory([]);
          // Clear input
          setInput("");
          // Close dialog after short delay with cleanup
          if (closeTimeoutRef.current) {
            clearTimeout(closeTimeoutRef.current);
          }
          closeTimeoutRef.current = setTimeout(() => {
            handleClose();
          }, SUCCESS_AUTO_CLOSE_DELAY_MS);
        } else {
          // Agent is asking for clarification
          // Don't show toast - response is already visible in UI
          // Keep input empty for user's response
          setInput("");
        }
      }
    } catch (error) {
      console.error("Agent error:", error);

      // Improved error handling
      let errorMessage = t("schedule.quick-input.parse-failed") as string;
      if (error instanceof Error) {
        if (error.message.includes("timeout") || error.message.includes("TIMEOUT")) {
          errorMessage = t("schedule.quick-input.timeout-error") as string;
        } else if (error.message.includes("network") || error.message.includes("fetch")) {
          errorMessage = t("schedule.quick-input.network-error") as string;
        } else if (error.message.includes("401") || error.message.includes("Unauthorized")) {
          errorMessage = t("schedule.quick-input.unauthorized-error") as string;
        }
      }

      const retrySuffix = t("schedule.quick-input.retry-manual-mode") as string;
      toast.error(errorMessage + retrySuffix);
    } finally {
      setIsProcessingAgent(false);
    }
  };

  const executeCreate = async () => {
    if (!parsedSchedule) return;

    try {
      if (isEditMode && parsedSchedule.name) {
        // Update existing schedule
        const updatedSchedule = await updateSchedule.mutateAsync({
          schedule: parsedSchedule,
          updateMask: ["title", "description", "location", "start_ts", "end_ts", "reminders"],
        });

        if (updatedSchedule) {
          toast.success((t("schedule.schedule-updated") as string) || "Schedule updated");
          onSuccess?.(updatedSchedule);
          handleClose();
        }
      } else {
        // Create new schedule
        const validName =
          parsedSchedule.name && parsedSchedule.name.startsWith("schedules/") && parsedSchedule.name.length > 10
            ? parsedSchedule.name
            : `schedules/${self.crypto.randomUUID()}`;

        const scheduleToCreate = { ...parsedSchedule, name: validName };

        const createdSchedule = await createSchedule.mutateAsync(scheduleToCreate);

        if (createdSchedule) {
          toast.success(t("schedule.schedule-created"));
          onSuccess?.(createdSchedule);
          handleClose();
        }
      }
    } catch (error) {
      // Check if error is due to schedule conflicts
      const isConflictError =
        error && typeof error === "object" && "message" in error ? (error.message as string).includes("conflicts detected") : false;

      if (isConflictError) {
        // Extract conflict details from error message if available
        const errorMessage = (error as { message?: string }).message || "";

        // Try to fetch conflicts again to show them in the alert dialog
        try {
          const conflictResult = await checkConflict.mutateAsync({
            startTs: parsedSchedule.startTs,
            endTs: parsedSchedule.endTs,
            excludeNames: isEditMode && parsedSchedule.name ? [parsedSchedule.name] : [],
          });

          if (conflictResult.conflicts.length > 0) {
            // Show conflicts in the alert dialog instead of toast
            setConflicts(conflictResult.conflicts);
            setShowConflictAlert(true);
          } else {
            // Fallback to toast if we can't fetch conflicts
            toast.error(errorMessage, {
              duration: 6000,
              id: "schedule-conflict-error",
            });
          }
        } catch (_conflictCheckError) {
          // If conflict check fails, show the original error message
          toast.error(errorMessage, {
            duration: 6000,
            id: "schedule-conflict-error",
          });
        }
      } else {
        toast.error(
          isEditMode ? (t("schedule.quick-input.failed-to-update") as string) : (t("schedule.quick-input.failed-to-create") as string),
        );
      }
      console.error(isEditMode ? "Update error:" : "Create error:", error);
    }
  };

  const handleCreate = async () => {
    if (!parsedSchedule) return;

    try {
      // Ensure parsedSchedule has a valid endTs (default 1 hour if 0)
      if (parsedSchedule.endTs === BigInt(0)) {
        parsedSchedule.endTs = parsedSchedule.startTs + BigInt(3600);
        setParsedSchedule({ ...parsedSchedule });
      }

      // Check conflict - exclude self if editing
      const conflict = await checkConflict.mutateAsync({
        startTs: parsedSchedule.startTs,
        endTs: parsedSchedule.endTs,
        excludeNames: isEditMode && parsedSchedule.name ? [parsedSchedule.name] : [],
      });

      if (conflict.conflicts.length > 0) {
        setConflicts(conflict.conflicts);
        setShowConflictAlert(true);
        return;
      }

      await executeCreate();
    } catch (error) {
      console.error("Conflict check error:", error);
      // If conflict check fails, try to create anyway? Or just show error?
      // For now, proceed to create which might fail if backend enforces strictly, but usually it doesn't.
      await executeCreate();
    }
  };

  const handleAdjust = () => {
    setShowConflictAlert(false);
    // Keep parsedSchedule to allow editing
  };

  const handleDiscard = () => {
    handleClose();
  };

  const handleClose = () => {
    setInput("");
    setParsedSchedule(null);
    setConflicts([]);
    setShowConflictAlert(false);
    setAgentResponse(null);
    setConversationHistory([]); // Clear conversation history
    onOpenChange(false);
  };

  // Handle delete schedule
  const handleDeleteClick = () => {
    setShowDeleteConfirm(true);
  };

  const confirmDelete = async () => {
    if (!parsedSchedule?.name) return;

    try {
      await deleteSchedule.mutateAsync(parsedSchedule.name);
      toast.success(t("schedule.schedule-deleted"));
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      handleClose();
    } catch (error) {
      console.error("Delete error:", error);
      toast.error(t("schedule.parse-error") || "Failed to delete schedule");
    } finally {
      setShowDeleteConfirm(false);
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <ScheduleErrorBoundary>
          <DialogContent className="max-w-md">
            <DialogTitle>{isEditMode ? t("schedule.edit-schedule") : t("schedule.create-schedule")}</DialogTitle>
            <DialogDescription>{isEditMode ? "" : t("schedule.natural-language-hint")}</DialogDescription>

            <div className="space-y-4 mt-4">
              {/* Natural Language Input - Only for create mode */}
              {!isEditMode && !parsedSchedule && (
                <div className="space-y-2">
                  <Label htmlFor="schedule-input">
                    {t("schedule.description") || "Description"}
                    {agentResponse && <span className="text-primary ml-2">{t("schedule.quick-input.reply-to-assistant-hint")}</span>}
                  </Label>
                  <Textarea
                    id="schedule-input"
                    placeholder={
                      agentResponse
                        ? (t("schedule.quick-input.input-placeholder-agent-active") as string)
                        : (t("schedule.quick-input.input-placeholder-default") as string)
                    }
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        handleAgentParse();
                      }
                    }}
                    className="min-h-24 resize-none"
                  />
                </div>
              )}

              {/* Parse Button - Only for create mode */}
              {!isEditMode && !parsedSchedule && (
                <Button onClick={handleAgentParse} disabled={!input.trim() || isProcessingAgent} className="w-full cursor-pointer">
                  {isProcessingAgent && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  <Bot className="mr-2 h-4 w-4" />
                  {agentResponse ? (t("schedule.quick-input.continue-chat") as string) : (t("schedule.quick-input.smart-parse") as string)}
                </Button>
              )}

              {/* Agent Response Display */}
              {agentResponse && !parsedSchedule && (
                <div className="rounded-lg border bg-primary/5 p-4">
                  <div className="flex items-start gap-2 mb-2">
                    <Bot className="h-4 w-4 text-primary mt-0.5" />
                    <h4 className="text-sm font-medium">{t("schedule.quick-input.assistant-reply") as string}</h4>
                  </div>
                  <div className="prose dark:prose-invert prose-sm max-w-none break-words text-sm text-muted-foreground">
                    <ReactMarkdown
                      remarkPlugins={[remarkGfm, remarkBreaks]}
                      components={{
                        a: ({ node, ...props }) => (
                          <a {...props} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline" />
                        ),
                        p: ({ node, ...props }) => <p {...props} className="mb-2 last:mb-0" />,
                        ul: ({ node, ...props }) => <ul {...props} className="list-disc list-inside mb-2 space-y-1" />,
                        ol: ({ node, ...props }) => <ol {...props} className="list-decimal list-inside mb-2 space-y-1" />,
                      }}
                    >
                      {agentResponse}
                    </ReactMarkdown>
                  </div>
                  <div className="mt-3 flex justify-end gap-2 flex-wrap">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setAgentResponse(null);
                        setInput("");
                        setConversationHistory([]);
                      }}
                      className="cursor-pointer"
                    >
                      {t("common.clear")}
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => {
                        setAgentResponse(null);
                        queryClient.invalidateQueries({ queryKey: ["schedules"] });
                      }}
                      className="cursor-pointer"
                    >
                      {t("schedule.quick-input.refresh-schedules") as string}
                    </Button>
                  </div>
                </div>
              )}

              {/* Schedule Details Form */}
              {parsedSchedule && (
                <div className="space-y-3 rounded-lg border bg-muted/50 p-4">
                  <div className="flex items-center justify-between">
                    <h4 className="font-medium text-sm">
                      {isEditMode ? (t("schedule.edit-schedule") as string) : (t("schedule.quick-input.parse-result") as string)}
                    </h4>
                    {/* Only show reset button in create mode */}
                    {!isEditMode && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setParsedSchedule(null);
                          setConflicts([]);
                        }}
                        className="cursor-pointer"
                      >
                        <X className="h-4 w-4" />
                      </Button>
                    )}
                  </div>

                  <div className="space-y-3 text-sm">
                    {/* Title */}
                    <div className="flex items-center gap-2">
                      <Calendar className="h-4 w-4 text-muted-foreground shrink-0" />
                      <Input
                        value={parsedSchedule.title}
                        onChange={(e) => setParsedSchedule({ ...parsedSchedule, title: e.target.value })}
                        className="h-8 font-medium"
                        placeholder={t("common.title")}
                      />
                    </div>

                    {/* Time */}
                    <div className="flex items-center gap-2">
                      <Clock className="h-4 w-4 text-muted-foreground shrink-0" />
                      <div className="flex items-center gap-2 w-full">
                        <Input
                          type="datetime-local"
                          value={dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.startTs, nanos: 0 }))).format(
                            "YYYY-MM-DDTHH:mm",
                          )}
                          onChange={(e) => {
                            const ts = BigInt(dayjs(e.target.value).unix());
                            setParsedSchedule({ ...parsedSchedule, startTs: ts });
                          }}
                          className="h-8 w-full"
                        />
                        <span className="text-muted-foreground">-</span>
                        <Input
                          type="datetime-local"
                          value={
                            parsedSchedule.endTs > 0
                              ? dayjs(timestampDate(create(TimestampSchema, { seconds: parsedSchedule.endTs, nanos: 0 }))).format(
                                  "YYYY-MM-DDTHH:mm",
                                )
                              : ""
                          }
                          onChange={(e) => {
                            const ts = BigInt(dayjs(e.target.value).unix());
                            setParsedSchedule({ ...parsedSchedule, endTs: ts });
                          }}
                          className="h-8 w-full"
                        />
                      </div>
                    </div>

                    {/* Location */}
                    <div className="flex items-center gap-2">
                      <MapPin className="h-4 w-4 text-muted-foreground shrink-0" />
                      <Input
                        value={parsedSchedule.location || ""}
                        onChange={(e) => setParsedSchedule({ ...parsedSchedule, location: e.target.value })}
                        className="h-8"
                        placeholder={t("common.location") || "Location"}
                      />
                    </div>

                    {/* Description - Always show in edit mode or when present */}
                    {(isEditMode || parsedSchedule.description) && (
                      <div className="pl-6">
                        <Textarea
                          value={parsedSchedule.description || ""}
                          onChange={(e) => setParsedSchedule({ ...parsedSchedule, description: e.target.value })}
                          className="min-h-[60px] text-xs resize-none"
                          placeholder={t("schedule.description")}
                        />
                      </div>
                    )}

                    {parsedSchedule.reminders.length > 0 && (
                      <div className="flex flex-wrap gap-1 pt-2 pl-6">
                        {parsedSchedule.reminders.map((reminder, idx) => (
                          <span key={idx} className="rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                            {reminder.type === "before" && (t("schedule.reminders") as string)}: {reminder.value} {reminder.unit}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className={cn("flex justify-between gap-2 pt-2", isEditMode ? "border-t border-border/50" : "justify-end")}>
                    {isEditMode && (
                      <Button
                        variant="ghost"
                        onClick={handleDeleteClick}
                        className="text-destructive hover:text-destructive hover:bg-destructive/10 cursor-pointer"
                      >
                        <Trash2 className="h-4 w-4 mr-1" />
                        {t("common.delete")}
                      </Button>
                    )}
                    <div className="flex gap-2 ml-auto">
                      <Button variant="outline" onClick={handleClose} className="cursor-pointer">
                        {t("common.cancel")}
                      </Button>
                      <Button onClick={handleCreate} className="cursor-pointer">
                        {isEditMode ? t("common.save") : t("schedule.create-schedule")}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </DialogContent>
        </ScheduleErrorBoundary>
      </Dialog>

      {/* Conflict Alert */}
      <ScheduleConflictAlert
        open={showConflictAlert}
        onOpenChange={setShowConflictAlert}
        conflicts={conflicts}
        onAdjust={handleAdjust}
        onDiscard={handleDiscard}
      />

      <ScheduleConflictAlert
        open={showConflictAlert}
        onOpenChange={setShowConflictAlert}
        conflicts={conflicts}
        onAdjust={handleAdjust}
        onDiscard={handleDiscard}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              {t("schedule.delete-schedule")}
            </DialogTitle>
            <DialogDescription>{t("schedule.delete-confirm")}</DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 mt-4">
            <Button variant="outline" onClick={() => setShowDeleteConfirm(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={confirmDelete}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
};

import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { aiServiceClient } from "@/connect";
import {
  AI_STORAGE_KEYS,
  AIChatContextValue,
  AIChatState,
  ChatItem,
  ContextSeparator,
  Conversation,
  ConversationMessage,
  ConversationViewMode,
  ReferencedMemo,
  SidebarTab,
} from "@/types/aichat";
import { ParrotAgentType } from "@/types/parrot";
import { CapabilityType, CapabilityStatus } from "@/types/capability";
import { AgentType, AIConversation, AIMessage } from "@/types/proto/api/v1/ai_service_pb";

const MESSAGE_CACHE_LIMIT = 100; // Maximum MSG messages to cache per conversation

const generateId = () => `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

// Helper function to get default conversation title based on parrot type.
// Note: This returns a fallback English title. The actual display titles are
// localized by the backend using title keys (e.g., "chat.default.title").
function getDefaultTitle(parrotId: ParrotAgentType): string {
  const titles: Record<string, string> = {
    [ParrotAgentType.DEFAULT]: "Chat with Default Assistant",
    [ParrotAgentType.MEMO]: "Chat with Memo",
    [ParrotAgentType.SCHEDULE]: "Chat with Schedule",
    [ParrotAgentType.AMAZING]: "Chat with Amazing",
    [ParrotAgentType.CREATIVE]: "Chat with Creative",
  };
  return titles[parrotId] || "AI Chat";
}

const DEFAULT_STATE: AIChatState = {
  conversations: [],
  currentConversationId: null,
  viewMode: "hub",
  sidebarTab: "history",
  sidebarOpen: true,
  currentCapability: CapabilityType.AUTO,
  capabilityStatus: "idle",
};

const AIChatContext = createContext<AIChatContextValue | null>(null);

export function useAIChat(): AIChatContextValue {
  const context = useContext(AIChatContext);
  if (!context) {
    throw new Error("useAIChat must be used within AIChatProvider");
  }
  return context;
}

interface AIChatProviderProps {
  children: ReactNode;
  initialState?: Partial<AIChatState>;
}

// Helper function to check if item is ContextSeparator
function isContextSeparator(item: ChatItem): item is ContextSeparator {
  return "type" in item && item.type === "context-separator";
}

// FIFO: Enforce message cache limit (only counts MSG, keeps SEP between MSG)
function enforceFIFOMessages(messages: ChatItem[]): ChatItem[] {
  const reversed = [...messages].reverse();
  const result: ChatItem[] = [];
  let msgCount = 0;

  for (const item of reversed) {
    if (isContextSeparator(item)) {
      // Keep SEPARATOR only if it's between MSG (i.e., we have MSG after it)
      if (msgCount > 0 && result.length > 0 && !isContextSeparator(result[0])) {
        result.unshift(item);
      }
    } else {
      // Regular MSG message
      if (msgCount < MESSAGE_CACHE_LIMIT) {
        result.unshift(item);
        msgCount++;
      }
      // Drop excess MSG messages
    }
  }

  return result;
}

export function AIChatProvider({ children, initialState }: AIChatProviderProps) {
  const { t } = useTranslation();
  const [state, setState] = useState<AIChatState>(() => ({
    ...DEFAULT_STATE,
    ...initialState,
  }));

  // Helper to localize conversation title from backend key to display title
  const localizeTitle = useCallback(
    (titleKey: string): string => {
      // Handle non-key strings (e.g., user custom titles)
      if (!titleKey || !titleKey.startsWith("chat.")) {
        return titleKey;
      }

      try {
        // Handle "chat.new" - backend now returns just "chat.new"
        // Numbering is handled by frontend based on conversation position
        if (titleKey === "chat.new") {
          return t("chat.new");
        }

        // Handle legacy "chat.new.N" format for backward compatibility
        const newChatMatch = titleKey.match(/^chat\.new\.(\d+)$/);
        if (newChatMatch) {
          return t("chat.new");
        }

        // Handle other "chat.*.title" format (e.g., "chat.default.title")
        if (titleKey.endsWith(".title")) {
          return t(titleKey, titleKey); // Fallback to original key if translation missing
        }
      } catch (err) {
        // Fallback to original key if parsing or translation fails
        console.warn("Failed to localize title key:", titleKey, err);
      }

      return titleKey;
    },
    [t],
  );

  // Helper to get message count
  const getMessageCount = useCallback((conversation: Conversation): number => {
    return conversation.messages.filter((item) => !isContextSeparator(item)).length;
  }, []);

  // Computed values
  const currentConversation = useMemo(() => {
    return state.conversations.find((c) => c.id === state.currentConversationId) || null;
  }, [state.conversations, state.currentConversationId]);

  const conversationSummaries = useMemo(() => {
    return state.conversations
      .map((c) => ({
        id: c.id,
        title: c.title,
        parrotId: c.parrotId,
        updatedAt: c.updatedAt,
        // Prefer backend messageCount, fallback to local calculation
        messageCount: c.messageCount ?? getMessageCount(c),
        pinned: c.pinned || false,
      }))
      .sort((a, b) => {
        // Pinned first
        if (a.pinned && !b.pinned) return -1;
        if (!a.pinned && b.pinned) return 1;
        // Then by updated time
        return b.updatedAt - a.updatedAt;
      });
  }, [state.conversations, getMessageCount]);

  // Helpers to convert from Protobuf to local types
  // Note: convertMessageFromPb must be defined before convertConversationFromPb
  const convertMessageFromPb = useCallback((m: AIMessage): ChatItem => {
    if (m.type === "SEPARATOR") {
      return {
        type: "context-separator",
        id: String(m.id),
        uid: m.uid, // Include UID for incremental sync
        timestamp: Number(m.createdTs) * 1000,
        synced: true,
      };
    }
    // Safe JSON parse with fallback
    let metadata = {};
    try {
      metadata = JSON.parse(m.metadata || "{}");
    } catch {
      console.warn("Failed to parse message metadata", m.metadata);
    }
    return {
      id: String(m.id),
      uid: m.uid, // Include UID for incremental sync
      role: m.role.toLowerCase() as any,
      content: m.content,
      timestamp: Number(m.createdTs) * 1000,
      metadata,
    };
  }, []);

  // Helper: Convert protobuf AgentType enum to ParrotAgentType string
  const convertAgentTypeToParrotId = useCallback((agentType: AgentType): ParrotAgentType => {
    switch (agentType) {
      case AgentType.MEMO:
        return ParrotAgentType.MEMO;
      case AgentType.SCHEDULE:
        return ParrotAgentType.SCHEDULE;
      case AgentType.AMAZING:
        return ParrotAgentType.AMAZING;
      case AgentType.CREATIVE:
        return ParrotAgentType.CREATIVE;
      case AgentType.DEFAULT:
      default:
        return ParrotAgentType.DEFAULT;
    }
  }, []);

  // Helper: Convert ParrotAgentType string to protobuf AgentType enum
  const convertParrotIdToAgentType = useCallback((parrotId: ParrotAgentType): AgentType => {
    switch (parrotId) {
      case ParrotAgentType.MEMO:
        return AgentType.MEMO;
      case ParrotAgentType.SCHEDULE:
        return AgentType.SCHEDULE;
      case ParrotAgentType.AMAZING:
        return AgentType.AMAZING;
      case ParrotAgentType.CREATIVE:
        return AgentType.CREATIVE;
      case ParrotAgentType.DEFAULT:
      default:
        return AgentType.DEFAULT;
    }
  }, []);

  const convertConversationFromPb = useCallback(
    (pb: AIConversation): Conversation => {
      return {
        id: String(pb.id),
        title: localizeTitle(pb.title),
        parrotId: convertAgentTypeToParrotId(pb.parrotId),
        createdAt: Number(pb.createdTs) * 1000,
        updatedAt: Number(pb.updatedTs) * 1000,
        messages: pb.messages.map((m) => convertMessageFromPb(m)),
        referencedMemos: [], // Backend managed for RAG, but state can store it if needed
        pinned: pb.pinned,
        messageCount: pb.messageCount, // Use backend-provided message count
      };
    },
    [convertMessageFromPb, localizeTitle, convertAgentTypeToParrotId],
  );

  // Sync state with backend
  const refreshConversations = useCallback(async () => {
    try {
      const response = await aiServiceClient.listAIConversations({});
      const conversations = response.conversations.map((c) => convertConversationFromPb(c));
      setState((prev) => ({ ...prev, conversations }));
    } catch (e) {
      console.error("Failed to fetch conversations:", e);
    }
  }, [convertConversationFromPb]);

  // Handle migration from localStorage
  const migrateFromStorage = useCallback(
    async (localConversations: Conversation[]) => {
      console.log("Migrating AI conversations to cloud storage...");
      for (const local of localConversations) {
        try {
          // Use the shared conversion helper
          const parrotId = convertParrotIdToAgentType(local.parrotId);

          const pb = await aiServiceClient.createAIConversation({
            title: local.title,
            parrotId,
          });

          if (local.pinned) {
            await aiServiceClient.updateAIConversation({ id: pb.id, pinned: true });
          }

          // We don't bulk migrate history to avoid database bloat,
          // but the user's list is now persisted.
          // If critical, we could iterate messages here too.
          console.log(`Migrated conversation: ${local.title}`);
        } catch (e) {
          console.error(`Failed to migrate conversation: ${local.title}`, e);
        }
      }
      // Clear localStorage once migrated
      localStorage.removeItem(AI_STORAGE_KEYS.CONVERSATIONS);
    },
    [convertParrotIdToAgentType],
  );

  // Conversation actions
  const createConversation = useCallback(
    (parrotId: ParrotAgentType, title?: string): string => {
      const tempId = generateId(); // Temporary ID for UI

      // Asynchronously create on backend
      const agentType =
        parrotId === ParrotAgentType.MEMO
          ? AgentType.MEMO
          : parrotId === ParrotAgentType.SCHEDULE
            ? AgentType.SCHEDULE
            : parrotId === ParrotAgentType.AMAZING
              ? AgentType.AMAZING
              : parrotId === ParrotAgentType.CREATIVE
                ? AgentType.CREATIVE
                : AgentType.DEFAULT;

      aiServiceClient
        .createAIConversation({
          title: title || getDefaultTitle(parrotId),
          parrotId: agentType,
        })
        .then((pb) => {
          // Optimistically update state with the new conversation
          const newConv = convertConversationFromPb(pb);

          setState((prev) => {
            // Prevent duplicates
            const exists = prev.conversations.some((c) => c.id === newConv.id);
            const newConversations = exists ? prev.conversations : [newConv, ...prev.conversations];

            return {
              ...prev,
              conversations: newConversations,
              currentConversationId: newConv.id,
              viewMode: "chat",
            };
          });

          // Refresh in background to ensure sync
          refreshConversations();
        })
        .catch((err) => {
          console.error("Failed to create conversation:", err);
          // Rollback to hub view on error
          setState((prev) => ({ ...prev, viewMode: "hub" }));
        });

      return tempId;
    },
    [refreshConversations],
  );

  const deleteConversation = useCallback(
    (id: string) => {
      const numericId = parseInt(id);
      if (!isNaN(numericId)) {
        aiServiceClient
          .deleteAIConversation({ id: numericId })
          .then(() => {
            refreshConversations();
          })
          .catch((err) => {
            console.error("Failed to delete conversation:", err);
          });
      }

      setState((prev) => {
        const filtered = prev.conversations.filter((c) => c.id !== id);
        const newCurrentId = prev.currentConversationId === id ? (filtered.length > 0 ? filtered[0].id : null) : prev.currentConversationId;

        return {
          ...prev,
          conversations: filtered,
          currentConversationId: newCurrentId,
          viewMode: filtered.length === 0 && prev.currentConversationId === id ? "hub" : prev.viewMode,
        };
      });
    },
    [refreshConversations],
  );

  const selectConversation = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      currentConversationId: id,
      viewMode: "chat",
    }));
  }, []);

  const updateConversationTitle = useCallback((id: string, title: string) => {
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.updateAIConversation({ id: numericId, title });
    }
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, title, updatedAt: Date.now() } : c)),
    }));
  }, []);

  const pinConversation = useCallback((id: string) => {
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.updateAIConversation({ id: numericId, pinned: true });
    }
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, pinned: true } : c)),
    }));
  }, []);

  const unpinConversation = useCallback((id: string) => {
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.updateAIConversation({ id: numericId, pinned: false });
    }
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, pinned: false } : c)),
    }));
  }, []);

  // Message actions
  const addMessage = useCallback((conversationId: string, message: Omit<ConversationMessage, "id" | "timestamp">): string => {
    // For cloud persistence, message IDs and timestamps are generated by the backend
    // during the chat stream. Here we just update local state for optimism.
    // Note: SEPARATOR messages are added via addContextSeparator, not here.
    const newMessageId = generateId();
    const now = Date.now();

    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        // Increment messageCount for real messages (SEPARATOR uses addContextSeparator)
        const newMessageCount = (c.messageCount ?? 0) + 1;

        return {
          ...c,
          messages: [...c.messages, { ...message, id: newMessageId, timestamp: now }],
          messageCount: newMessageCount, // Update message count for conversation list
          updatedAt: now,
        };
      }),
    }));
    return newMessageId;
  }, []);

  const updateMessage = useCallback((conversationId: string, messageId: string, updates: Partial<ConversationMessage>) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        return {
          ...c,
          messages: c.messages.map((m) => {
            if (isContextSeparator(m)) return m;
            if (m.id !== messageId) return m;
            return { ...m, ...updates };
          }),
          updatedAt: Date.now(),
        };
      }),
    }));
  }, []);

  const deleteMessage = useCallback((conversationId: string, messageId: string) => {
    // Current backend doesn't support individual message deletion via API yet
    // but we update state for immediate UI feedback
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        return {
          ...c,
          messages: c.messages.filter((m) => !isContextSeparator(m) || ("id" in m && m.id !== messageId)),
          updatedAt: Date.now(),
        };
      }),
    }));
  }, []);

  const clearMessages = useCallback((conversationId: string) => {
    // For cloud persistence, clearing messages is actually adding a separator
    // or deleting the conversation. Here we treat it as an optimistic clear
    // but the backend will handle the clear context on the next message
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) =>
        c.id === conversationId ? { ...c, messages: [], messageCount: 0, updatedAt: Date.now() } : c,
      ),
    }));
  }, []);

  const addContextSeparator = useCallback(
    (conversationId: string, _trigger: "manual" | "auto" | "shortcut" = "manual") => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return "";

      // Call backend API to persist the SEPARATOR message
      // Backend is idempotent: won't create duplicate if last message is already SEPARATOR
      aiServiceClient
        .addContextSeparator({ conversationId: numericId })
        .then(() => {
          // After successful creation, refresh the conversation to show the new separator
          aiServiceClient.getAIConversation({ id: numericId }).then((pb) => {
            const fullConversation = convertConversationFromPb(pb);
            setState((prev) => ({
              ...prev,
              conversations: prev.conversations.map((c) => (c.id === conversationId ? fullConversation : c)),
            }));
          });
        })
        .catch((err) => {
          console.error("Failed to add context separator:", err);
        });

      // Note: Removed optimistic update to prevent duplicate SEPARATOR accumulation
      // Backend refresh is fast enough and the idempotent check prevents duplicates
      return "";
    },
    [convertConversationFromPb],
  );

  // Helper function to merge messages into state (pure function, safe for async callbacks)
  function mergeMessagesIntoState(
    prevState: AIChatState,
    conversationId: string,
    response: {
      messages: AIMessage[];
      hasMore: boolean;
      totalCount: number;
      latestMessageUid: string;
    },
    convertMessageFromPb: (m: AIMessage) => ChatItem,
  ): AIChatState {
    const newMessages = response.messages.map((m) => convertMessageFromPb(m));

    return {
      ...prevState,
      conversations: prevState.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        // For first load (no existing cache), use new messages directly
        if (!c.messageCache) {
          return {
            ...c,
            messages: enforceFIFOMessages(newMessages),
            messageCache: {
              lastMessageUid: response.latestMessageUid,
              totalCount: response.totalCount,
              hasMore: response.hasMore,
            },
          };
        }

        // For incremental sync, merge with existing messages
        const existingMessages = c.messages || [];
        const mergedMessages = [...existingMessages];

        // Append new messages (avoid duplicates by UID)
        const existingUids = new Set(existingMessages.filter((m) => !isContextSeparator(m)).map((m) => ("uid" in m ? m.uid : m.id)));

        for (const msg of newMessages) {
          if (!isContextSeparator(msg)) {
            const uid = (msg as any).uid || msg.id;
            if (!existingUids.has(uid)) {
              mergedMessages.push(msg);
            }
          } else {
            mergedMessages.push(msg);
          }
        }

        // Enforce FIFO limit
        return {
          ...c,
          messages: enforceFIFOMessages(mergedMessages),
          messageCache: {
            lastMessageUid: response.latestMessageUid,
            totalCount: response.totalCount,
            hasMore: response.hasMore,
          },
        };
      }),
    };
  }

  // Message sync actions with incremental sync and FIFO cache
  const syncMessages = useCallback(
    async (conversationId: string) => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return;

      // Use functional state update to get current state without creating dependency
      setState((prev) => {
        const conversation = prev.conversations.find((c) => c.id === conversationId);
        const lastMessageUid = conversation?.messageCache?.lastMessageUid || "";
        const limit = 100; // Always request 100 messages

        // Fire-and-forget async operation (we can't await in setState updater)
        aiServiceClient
          .listMessages({
            conversationId: numericId,
            lastMessageUid,
            limit,
          })
          .then((response) => {
            if (response.syncRequired) {
              // Backend requires full refresh, clear cache and retry
              setState((innerPrev) => ({
                ...innerPrev,
                conversations: innerPrev.conversations.map((c) => {
                  if (c.id !== conversationId) return c;
                  return { ...c, messages: [], messageCache: undefined };
                }),
              }));
              // Retry with empty lastMessageUid
              aiServiceClient
                .listMessages({
                  conversationId: numericId,
                  lastMessageUid: "",
                  limit,
                })
                .then((retryResponse) => {
                  setState((innerPrev) => mergeMessagesIntoState(innerPrev, conversationId, retryResponse, convertMessageFromPb));
                });
            } else {
              setState((innerPrev) => mergeMessagesIntoState(innerPrev, conversationId, response, convertMessageFromPb));
            }
          })
          .catch((e: unknown) => {
            console.error("Failed to sync messages:", e);
          });

        return prev; // Return previous state unchanged (async update will follow)
      });
    },
    [convertMessageFromPb],
  ); // Only depends on convertMessageFromPb

  const loadMoreMessages = useCallback(
    async (conversationId: string) => {
      const numericId = parseInt(conversationId);
      if (isNaN(numericId)) return;

      // Use functional state update to avoid stale closure
      setState((prev) => {
        const conversation = prev.conversations.find((c) => c.id === conversationId);
        if (!conversation?.messageCache?.hasMore) return prev; // No more messages to load

        // Find the oldest message UID for pagination
        const oldestMessage = conversation.messages.find((m) => !isContextSeparator(m));
        if (!oldestMessage) return prev;

        const oldestUid = (oldestMessage as any).uid || oldestMessage.id;

        // Fire-and-forget async operation
        aiServiceClient
          .listMessages({
            conversationId: numericId,
            lastMessageUid: oldestUid as string, // Request messages before this UID
            limit: 100,
          })
          .then((response) => {
            // Prepend messages (older messages come first)
            const olderMessages = response.messages.map((m) => convertMessageFromPb(m));
            setState((innerPrev) => ({
              ...innerPrev,
              conversations: innerPrev.conversations.map((c) => {
                if (c.id !== conversationId) return c;

                return {
                  ...c,
                  messages: [...olderMessages, ...c.messages],
                  messageCache: c.messageCache
                    ? {
                        ...c.messageCache,
                        hasMore: response.hasMore,
                      }
                    : undefined,
                };
              }),
            }));
          })
          .catch((e: unknown) => {
            console.error("Failed to load more messages:", e);
          });

        return prev;
      });
    },
    [convertMessageFromPb],
  ); // Only depends on convertMessageFromPb

  // Referenced content actions
  const addReferencedMemos = useCallback((conversationId: string, memos: ReferencedMemo[]) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        const existingUids = new Set(c.referencedMemos.map((m) => m.uid));
        const newMemos = memos.filter((m) => !existingUids.has(m.uid));

        return {
          ...c,
          referencedMemos: [...c.referencedMemos, ...newMemos],
        };
      }),
    }));
  }, []);

  // UI actions
  const setViewMode = useCallback((mode: ConversationViewMode) => {
    setState((prev) => ({ ...prev, viewMode: mode }));
  }, []);

  const setSidebarTab = useCallback((tab: SidebarTab) => {
    setState((prev) => ({ ...prev, sidebarTab: tab }));
  }, []);

  const setSidebarOpen = useCallback((open: boolean) => {
    setState((prev) => ({ ...prev, sidebarOpen: open }));
  }, []);

  const toggleSidebar = useCallback(() => {
    setState((prev) => ({ ...prev, sidebarOpen: !prev.sidebarOpen }));
  }, []);

  // ============================================================
  // CAPABILITY ACTIONS (新增 - 能力管理)
  // ============================================================
  const setCurrentCapability = useCallback((capability: CapabilityType) => {
    setState((prev) => ({ ...prev, currentCapability: capability }));
  }, []);

  const setCapabilityStatus = useCallback((status: CapabilityStatus) => {
    setState((prev) => ({ ...prev, capabilityStatus: status }));
  }, []);

  // Persistence
  const saveToStorage = useCallback(() => {
    try {
      // We still use localStorage for UI preferences, but not conversations
      localStorage.setItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION, state.currentConversationId || "");
      localStorage.setItem(AI_STORAGE_KEYS.SIDEBAR_TAB, state.sidebarTab);
    } catch (e) {
      console.error("Failed to save AI chat state:", e);
    }
  }, [state.currentConversationId, state.sidebarTab]);

  const loadFromStorage = useCallback(async () => {
    try {
      const conversationsData = localStorage.getItem(AI_STORAGE_KEYS.CONVERSATIONS);
      const currentConversationData = localStorage.getItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION);
      const sidebarTabData = localStorage.getItem(AI_STORAGE_KEYS.SIDEBAR_TAB);

      if (conversationsData) {
        const localConversations = JSON.parse(conversationsData);
        if (localConversations.length > 0) {
          await migrateFromStorage(localConversations);
        }
      }

      await refreshConversations();

      setState((prev) => ({
        ...prev,
        currentConversationId: currentConversationData || null,
        sidebarTab: sidebarTabData === "history" || sidebarTabData === "memos" ? (sidebarTabData as SidebarTab) : "history",
      }));
    } catch (e) {
      console.error("Failed to load AI chat state:", e);
      refreshConversations();
    }
  }, [migrateFromStorage, refreshConversations]);

  const clearStorage = useCallback(() => {
    try {
      localStorage.removeItem(AI_STORAGE_KEYS.CONVERSATIONS);
      localStorage.removeItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION);
      localStorage.removeItem(AI_STORAGE_KEYS.SIDEBAR_TAB);
      setState({ ...DEFAULT_STATE });
    } catch (e) {
      console.error("Failed to clear AI chat state:", e);
    }
  }, []);

  // Auto-save to localStorage when state changes (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      saveToStorage();
    }, 500); // 500ms debounce
    return () => clearTimeout(timer);
  }, [state, saveToStorage]);

  // Load from storage on mount (only once)
  useEffect(() => {
    loadFromStorage();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync messages when conversation changes (only for valid numeric IDs)
  useEffect(() => {
    if (state.currentConversationId) {
      const numericId = parseInt(state.currentConversationId);
      if (!isNaN(numericId)) {
        syncMessages(state.currentConversationId);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.currentConversationId]);

  const contextValue: AIChatContextValue = {
    state,
    currentConversation,
    conversations: state.conversations,
    conversationSummaries,
    createConversation,
    deleteConversation,
    selectConversation,
    updateConversationTitle,
    pinConversation,
    unpinConversation,
    addMessage,
    updateMessage,
    deleteMessage,
    clearMessages,
    addContextSeparator,
    addReferencedMemos,
    setViewMode,
    setSidebarTab,
    setSidebarOpen,
    toggleSidebar,
    setCurrentCapability,
    setCapabilityStatus,
    saveToStorage,
    loadFromStorage,
    clearStorage,
    syncMessages,
    loadMoreMessages,
  };

  return <AIChatContext.Provider value={contextValue}>{children}</AIChatContext.Provider>;
}

import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
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
import { AgentType, AIConversation, AIMessage } from "@/types/proto/api/v1/ai_service_pb";

const generateId = () => `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

const DEFAULT_STATE: AIChatState = {
  conversations: [],
  currentConversationId: null,
  viewMode: "hub",
  sidebarTab: "history",
  sidebarOpen: true,
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

export function AIChatProvider({ children, initialState }: AIChatProviderProps) {
  const [state, setState] = useState<AIChatState>(() => ({
    ...DEFAULT_STATE,
    ...initialState,
  }));

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
        messageCount: getMessageCount(c),
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
  const convertConversationFromPb = useCallback((pb: AIConversation): Conversation => {
    return {
      id: String(pb.id),
      title: pb.title,
      parrotId: pb.parrotId === AgentType.AGENT_TYPE_MEMO ? ParrotAgentType.MEMO :
        pb.parrotId === AgentType.AGENT_TYPE_SCHEDULE ? ParrotAgentType.SCHEDULE :
          pb.parrotId === AgentType.AGENT_TYPE_AMAZING ? ParrotAgentType.AMAZING :
            pb.parrotId === AgentType.AGENT_TYPE_CREATIVE ? ParrotAgentType.CREATIVE :
              ParrotAgentType.DEFAULT,
      createdAt: Number(pb.createdTs) * 1000,
      updatedAt: Number(pb.updatedTs) * 1000,
      messages: pb.messages.map(m => convertMessageFromPb(m)),
      referencedMemos: [], // Backend managed for RAG, but state can store it if needed
      pinned: pb.pinned,
    };
  }, []);

  const convertMessageFromPb = useCallback((m: AIMessage): ChatItem => {
    if (m.type === "SEPARATOR") {
      return {
        type: "context-separator",
        id: String(m.id),
        timestamp: Number(m.createdTs) * 1000,
        synced: true,
      };
    }
    return {
      id: String(m.id),
      role: m.role.toLowerCase() as any,
      content: m.content,
      timestamp: Number(m.createdTs) * 1000,
      metadata: JSON.parse(m.metadata || "{}"),
    };
  }, []);

  // Sync state with backend
  const refreshConversations = useCallback(async () => {
    try {
      const response = await aiServiceClient.listAIConversations({});
      const conversations = response.conversations.map(c => convertConversationFromPb(c));
      setState(prev => ({ ...prev, conversations }));
    } catch (e) {
      console.error("Failed to fetch conversations:", e);
    }
  }, [convertConversationFromPb]);

  // Handle migration from localStorage
  const migrateFromStorage = useCallback(async (localConversations: Conversation[]) => {
    console.log("Migrating AI conversations to cloud storage...");
    for (const local of localConversations) {
      try {
        // Create conversation
        const parrotId = local.parrotId === ParrotAgentType.MEMO ? AgentType.AGENT_TYPE_MEMO :
          local.parrotId === ParrotAgentType.SCHEDULE ? AgentType.AGENT_TYPE_SCHEDULE :
            local.parrotId === ParrotAgentType.AMAZING ? AgentType.AGENT_TYPE_AMAZING :
              local.parrotId === ParrotAgentType.CREATIVE ? AgentType.AGENT_TYPE_CREATIVE :
                AgentType.AGENT_TYPE_DEFAULT;

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
  }, []);

  // Conversation actions
  const createConversation = useCallback((parrotId: ParrotAgentType, title?: string): string => {
    const tempId = generateId(); // Temporary ID for UI

    // Asynchronously create on backend
    const agentType = parrotId === ParrotAgentType.MEMO ? AgentType.AGENT_TYPE_MEMO :
      parrotId === ParrotAgentType.SCHEDULE ? AgentType.AGENT_TYPE_SCHEDULE :
        parrotId === ParrotAgentType.AMAZING ? AgentType.AGENT_TYPE_AMAZING :
          parrotId === ParrotAgentType.CREATIVE ? AgentType.AGENT_TYPE_CREATIVE :
            AgentType.AGENT_TYPE_DEFAULT;

    aiServiceClient.createAIConversation({
      title: title || getDefaultTitle(parrotId),
      parrotId: agentType,
    }).then(pb => {
      refreshConversations().then(() => {
        setState(prev => ({ ...prev, currentConversationId: String(pb.id) }));
      });
    });

    return tempId;
  }, [refreshConversations]);

  const deleteConversation = useCallback((id: string) => {
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.deleteAIConversation({ id: numericId }).then(() => {
        refreshConversations();
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
  }, [refreshConversations]);

  const selectConversation = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      currentConversationId: id,
      viewMode: "chat",
    }));

    // Fetch full conversation with messages if needed
    const numericId = parseInt(id);
    if (!isNaN(numericId)) {
      aiServiceClient.getAIConversation({ id: numericId }).then(pb => {
        const fullConversation = convertConversationFromPb(pb);
        setState(prev => ({
          ...prev,
          conversations: prev.conversations.map(c => c.id === id ? fullConversation : c)
        }));
      });
    }
  }, [convertConversationFromPb]);

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
    const newMessageId = generateId();
    const now = Date.now();

    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        return {
          ...c,
          messages: [...c.messages, { ...message, id: newMessageId, timestamp: now }],
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
      conversations: prev.conversations.map((c) => (c.id === conversationId ? { ...c, messages: [], updatedAt: Date.now() } : c)),
    }));
  }, []);

  const addContextSeparator = useCallback((conversationId: string, trigger: "manual" | "auto" | "shortcut" = "manual") => {
    const separatorId = generateId();
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => {
        if (c.id !== conversationId) return c;

        const separator: ContextSeparator = {
          type: "context-separator",
          id: separatorId,
          timestamp: Date.now(),
          synced: false, // Will be synced when the user sends the next message ('---' command)
          trigger,
        };

        return {
          ...c,
          messages: [...c.messages, separator],
          updatedAt: Date.now(),
        };
      }),
    }));
    return separatorId;
  }, []);

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

  // Load from storage on mount
  useEffect(() => {
    loadFromStorage();
  }, [loadFromStorage]);

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
    saveToStorage,
    loadFromStorage,
    clearStorage,
  };

  return <AIChatContext.Provider value={contextValue}>{children}</AIChatContext.Provider>;
}

// Helper function to get default conversation title based on parrot type
// Returns English default; localization should be handled by caller via useTranslation hook
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

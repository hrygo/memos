import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
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

  // Conversation actions
  const createConversation = useCallback((parrotId: ParrotAgentType, title?: string): string => {
    const now = Date.now();
    const newConversation: Conversation = {
      id: generateId(),
      title: title || getDefaultTitle(parrotId),
      parrotId,
      createdAt: now,
      updatedAt: now,
      messages: [],
      referencedMemos: [],
      pinned: false,
    };

    setState((prev) => ({
      ...prev,
      conversations: [newConversation, ...prev.conversations],
      currentConversationId: newConversation.id,
      viewMode: "chat",
    }));

    return newConversation.id;
  }, []);

  const deleteConversation = useCallback((id: string) => {
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
  }, []);

  const selectConversation = useCallback((id: string) => {
    setState((prev) => {
      if (!prev.conversations.find((c) => c.id === id)) return prev;
      return {
        ...prev,
        currentConversationId: id,
        viewMode: "chat",
      };
    });
  }, []);

  const updateConversationTitle = useCallback((id: string, title: string) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, title, updatedAt: Date.now() } : c)),
    }));
  }, []);

  const pinConversation = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, pinned: true } : c)),
    }));
  }, []);

  const unpinConversation = useCallback((id: string) => {
    setState((prev) => ({
      ...prev,
      conversations: prev.conversations.map((c) => (c.id === id ? { ...c, pinned: false } : c)),
    }));
  }, []);

  // Message actions
  const addMessage = useCallback((conversationId: string, message: Omit<ConversationMessage, "id" | "timestamp">): string => {
    // Generate ID and timestamp BEFORE setState (setState is async!)
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
          id: separatorId, // Future: sync ID for conversation storage
          timestamp: Date.now(),
          synced: false, // Future: sync status
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
      localStorage.setItem(AI_STORAGE_KEYS.CONVERSATIONS, JSON.stringify(state.conversations));
      localStorage.setItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION, state.currentConversationId || "");
      localStorage.setItem(AI_STORAGE_KEYS.SIDEBAR_TAB, state.sidebarTab);
    } catch (e) {
      console.error("Failed to save AI chat state:", e);
    }
  }, [state]);

  const loadFromStorage = useCallback(() => {
    try {
      const conversationsData = localStorage.getItem(AI_STORAGE_KEYS.CONVERSATIONS);
      const currentConversationData = localStorage.getItem(AI_STORAGE_KEYS.CURRENT_CONVERSATION);
      const sidebarTabData = localStorage.getItem(AI_STORAGE_KEYS.SIDEBAR_TAB);

      setState((prev) => ({
        ...prev,
        conversations: conversationsData ? JSON.parse(conversationsData) : [],
        currentConversationId: currentConversationData || null,
        sidebarTab: sidebarTabData === "history" || sidebarTabData === "memos" ? (sidebarTabData as SidebarTab) : "history",
      }));
    } catch (e) {
      console.error("Failed to load AI chat state:", e);
    }
  }, []);

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

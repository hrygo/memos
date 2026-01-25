
import { MessageSquarePlus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAIChat } from "@/contexts/AIChatContext";
import { ParrotAgentI18n, useAvailableParrots } from "@/hooks/useParrots";
import { cn } from "@/lib/utils";
import { PARROT_ICONS, PARROT_THEMES } from "@/types/parrot";

export function ParrotHub() {
    const { t } = useTranslation();
    const { createConversation, conversations, selectConversation } = useAIChat();
    const availableParrots = useAvailableParrots();

    const handleAgentSelect = (agent: ParrotAgentI18n) => {
        // Check if there's an existing conversation for this agent
        const existingConversation = conversations.find((c) => c.parrotId === agent.id);

        if (existingConversation) {
            selectConversation(existingConversation.id);
        } else {
            createConversation(agent.id, agent.displayName);
        }
    };

    return (
        <div className="w-full h-full overflow-y-auto bg-zinc-50 dark:bg-zinc-900 p-4 md:p-8">
            <div className="max-w-5xl mx-auto">

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 md:gap-6">
                    {availableParrots.map((agent) => {
                        const theme = PARROT_THEMES[agent.id] || PARROT_THEMES.DEFAULT;
                        const icon = PARROT_ICONS[agent.id] || agent.icon;

                        return (
                            <button
                                key={agent.id}
                                onClick={() => handleAgentSelect(agent)}
                                className={cn(
                                    "flex flex-col text-left h-full p-6 rounded-2xl border transition-all duration-300 group hover:shadow-lg relative overflow-hidden",
                                    "bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700/50",
                                    "hover:border-zinc-300 dark:hover:border-zinc-700",
                                    "focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-900 dark:focus:ring-zinc-100"
                                )}
                            >
                                {/* Background Decor */}
                                <div className={cn(
                                    "absolute top-0 right-0 w-32 h-32 rounded-full blur-3xl opacity-0 group-hover:opacity-20 transition-opacity duration-500",
                                    theme.accent
                                )} />

                                <div className="relative z-10 flex flex-col h-full">
                                    <div className="flex items-start justify-between mb-4">
                                        <div className={cn(
                                            "w-12 h-12 rounded-xl flex items-center justify-center text-2xl shadow-sm border border-zinc-200 dark:border-zinc-700 transition-transform group-hover:scale-110 duration-300",
                                            theme.iconBg
                                        )}>
                                            {icon.startsWith("/") ? (
                                                <img src={icon} alt={agent.displayName} className="w-8 h-8 object-contain" />
                                            ) : (
                                                <span>{icon}</span>
                                            )}
                                        </div>
                                    </div>

                                    <h3 className={cn("text-lg font-bold mb-1 group-hover:text-zinc-900 dark:group-hover:text-zinc-50 transition-colors", "text-zinc-900 dark:text-zinc-100")}>
                                        {agent.displayName}
                                    </h3>

                                    <p className="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-3 uppercase tracking-wider">
                                        {agent.displayNameAlt}
                                    </p>

                                    <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed mb-6 flex-grow">
                                        {agent.description}
                                    </p>

                                    <div className={cn(
                                        "mt-auto flex items-center text-sm font-semibold transition-colors duration-200",
                                        theme.iconText
                                    )}>
                                        <span>{t("ai.start-chat") || "Start Chat"}</span>
                                        <MessageSquarePlus className="w-4 h-4 ml-2 transition-transform group-hover:translate-x-1" />
                                    </div>
                                </div>
                            </button>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}

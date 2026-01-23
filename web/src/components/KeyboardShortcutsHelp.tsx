import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Kbd } from "@/components/ui/kbd";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

interface Shortcut {
  key: string;
  description: string;
}

interface ShortcutCategory {
  title: string;
  shortcuts: Shortcut[];
}

const KeyboardShortcutsHelp = () => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Open with "?" key, but only when not typing in an input
      if (e.key === "?" && !(e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement)) {
        e.preventDefault();
        setOpen(true);
      }
      // Close with Escape
      if (e.key === "Escape" && open) {
        setOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  const categories: ShortcutCategory[] = [
    {
      title: t("shortcuts.editor") || "Editor",
      shortcuts: [
        { key: "Cmd/Ctrl + Enter", description: t("shortcuts.save-memo") || "Save memo" },
        { key: "Esc", description: t("shortcuts.cancel-edit") || "Cancel edit" },
      ],
    },
    {
      title: t("shortcuts.formatting") || "Formatting",
      shortcuts: [
        { key: "Cmd/Ctrl + B", description: t("shortcuts.bold") || "Bold text" },
        { key: "Cmd/Ctrl + I", description: t("shortcuts.italic") || "Italic text" },
        { key: "Cmd/Ctrl + K", description: t("shortcuts.link") || "Insert link" },
      ],
    },
    {
      title: t("shortcuts.navigation") || "Navigation",
      shortcuts: [
        { key: "?", description: t("shortcuts.show-help") || "Show keyboard shortcuts" },
        { key: "Esc", description: t("shortcuts.close-dialog") || "Close dialog/drawer" },
      ],
    },
  ];

  const formatKey = (key: string) => {
    return key.split("/").map((part, index) => (
      <span key={index} className={cn("flex items-center gap-1", index > 0 && "ml-1")}>
        {part.split(" + ").map((k, i) => (
          <span key={i} className="flex items-center gap-1">
            <Kbd>{k}</Kbd>
            {i < part.split(" + ").length - 1 && <span className="text-muted-foreground text-xs">+</span>}
          </span>
        ))}
        {index < (key.split("/").length - 1) && <span className="text-muted-foreground text-xs mx-1">or</span>}
      </span>
    ));
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-md" aria-describedby="keyboard-shortcuts-description">
        <DialogHeader>
          <DialogTitle>{t("shortcuts.title") || "Keyboard Shortcuts"}</DialogTitle>
          <DialogDescription id="keyboard-shortcuts-description">
            {t("shortcuts.description") || "Press ? to open this help dialog anytime"}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          {categories.map((category) => (
            <div key={category.title}>
              <h3 className="text-sm font-semibold mb-2">{category.title}</h3>
              <div className="space-y-2">
                {category.shortcuts.map((shortcut) => (
                  <div
                    key={shortcut.key}
                    className="flex items-center justify-between text-sm"
                  >
                    <span className="text-muted-foreground">{shortcut.description}</span>
                    <div className="flex items-center">{formatKey(shortcut.key)}</div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default KeyboardShortcutsHelp;

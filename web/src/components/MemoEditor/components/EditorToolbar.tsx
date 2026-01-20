import { LoaderIcon, SparklesIcon } from "lucide-react";
import type { FC } from "react";
import { toast } from "react-hot-toast";
import { Button } from "@/components/ui/button";
import { useSuggestTags } from "@/hooks/useAIQueries";
import { useTranslate } from "@/utils/i18n";
import { validationService } from "../services";
import { useEditorContext } from "../state";
import InsertMenu from "../Toolbar/InsertMenu";
import VisibilitySelector from "../Toolbar/VisibilitySelector";
import type { EditorToolbarProps } from "../types";

export const EditorToolbar: FC<EditorToolbarProps> = ({ onSave, onCancel, memoName }) => {
  const t = useTranslate();
  const { state, actions, dispatch } = useEditorContext();
  const { valid } = validationService.canSave(state);
  const { mutate: suggestTags, isPending: isSuggesting } = useSuggestTags();

  const isSaving = state.ui.isLoading.saving;

  const handleLocationChange = (location: typeof state.metadata.location) => {
    dispatch(actions.setMetadata({ location }));
  };

  const handleToggleFocusMode = () => {
    dispatch(actions.toggleFocusMode());
  };

  const handleVisibilityChange = (visibility: typeof state.metadata.visibility) => {
    dispatch(actions.setMetadata({ visibility }));
  };

  const handleSuggestTags = () => {
    if (!state.content || state.content.length < 5) {
      toast.error("Content too short for AI analysis");
      return;
    }

    suggestTags(
      { content: state.content },
      {
        onSuccess: (tags) => {
          if (tags.length > 0) {
            // Filter out existing tags to avoid duplicates? Simple approach for now.
            const newTags = tags.map((tag) => `#${tag}`).join(" ");
            const newContent = state.content + (state.content.endsWith("\n") ? "" : "\n") + newTags;
            dispatch(actions.updateContent(newContent));
            toast.success("Tags added!");
          } else {
            toast("No tags suggested");
          }
        },
        onError: (err) => {
          toast.error("Failed: " + err.message);
        },
      },
    );
  };

  return (
    <div className="w-full flex flex-row justify-between items-center mb-2">
      <div className="flex flex-row justify-start items-center">
        <InsertMenu
          isUploading={state.ui.isLoading.uploading}
          location={state.metadata.location}
          onLocationChange={handleLocationChange}
          onToggleFocusMode={handleToggleFocusMode}
          memoName={memoName}
        />
      </div>

      <div className="flex flex-row justify-end items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          className="w-8 h-8 text-muted-foreground hover:text-blue-500"
          onClick={handleSuggestTags}
          disabled={isSuggesting || !state.content}
          title={t("common.ai-assistant")}
        >
          {isSuggesting ? <LoaderIcon className="w-4 h-4 animate-spin" /> : <SparklesIcon className="w-4 h-4" />}
        </Button>

        <VisibilitySelector value={state.metadata.visibility} onChange={handleVisibilityChange} />

        {onCancel && (
          <Button variant="ghost" onClick={onCancel} disabled={isSaving}>
            Cancel
          </Button>
        )}

        <Button onClick={onSave} disabled={!valid || isSaving}>
          {isSaving ? "Saving..." : "Save"}
        </Button>
      </div>
    </div>
  );
};

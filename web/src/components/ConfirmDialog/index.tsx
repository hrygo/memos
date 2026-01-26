import * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  confirmLabel: string;
  cancelLabel: string;
  onConfirm: () => void | Promise<void>;
  confirmVariant?: "default" | "destructive";
}

export default function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  onConfirm,
  confirmVariant = "default",
}: ConfirmDialogProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = React.useState(false);

  const handleConfirm = async () => {
    try {
      setLoading(true);
      await onConfirm();
      onOpenChange(false);
    } catch (e) {
      // Intentionally swallow errors so user can retry; surface via caller's toast/logging
      console.error("ConfirmDialog error for action:", title, e);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o: boolean) => !loading && onOpenChange(o)}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className={description ? "" : "sr-only"}>{description || t("common.confirm")}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="ghost" disabled={loading} onClick={() => onOpenChange(false)}>
            {cancelLabel}
          </Button>
          <Button variant={confirmVariant} disabled={loading} onClick={handleConfirm} data-loading={loading}>
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

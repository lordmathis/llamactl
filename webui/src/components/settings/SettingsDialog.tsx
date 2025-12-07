import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import ApiKeysSection from "./ApiKeysSection";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Settings</DialogTitle>
          <DialogDescription>
            Manage your application settings and API keys.
          </DialogDescription>
        </DialogHeader>
        <ApiKeysSection />
      </DialogContent>
    </Dialog>
  );
}

export default SettingsDialog;

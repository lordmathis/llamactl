import { useState } from "react";
import Header from "@/components/Header";
import InstanceList from "@/components/InstanceList";
import ModelsList from "@/components/ModelsList";
import InstanceDialog from "@/components/InstanceDialog";
import LoginDialog from "@/components/LoginDialog";
import SystemInfoDialog from "./components/SystemInfoDialog";
import SettingsDialog from "./components/settings/SettingsDialog";
import { type CreateInstanceOptions, type Instance } from "@/types/instance";
import { useInstances } from "@/contexts/InstancesContext";
import { useAuth } from "@/contexts/AuthContext";
import { ThemeProvider } from "@/contexts/ThemeContext";
import { Toaster } from "sonner";
import { cn } from "@/lib/utils";

function App() {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [activeTab, setActiveTab] = useState<'instances' | 'models'>('instances');
  const [isInstanceModalOpen, setIsInstanceModalOpen] = useState(false);
  const [isSystemInfoModalOpen, setIsSystemInfoModalOpen] = useState(false);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);
  const [editingInstance, setEditingInstance] = useState<Instance | undefined>(
    undefined
  );
  const { createInstance, updateInstance } = useInstances();

  const handleCreateInstance = () => {
    setEditingInstance(undefined);
    setIsInstanceModalOpen(true);
  };

  const handleEditInstance = (instance: Instance) => {
    setEditingInstance(instance);
    setIsInstanceModalOpen(true);
  };

  const handleSaveInstance = (name: string, options: CreateInstanceOptions) => {
    if (editingInstance) {
      void updateInstance(editingInstance.name, options);
    } else {
      void createInstance(name, options);
    }
  };

  const handleShowSystemInfo = () => {
    setIsSystemInfoModalOpen(true);
  };

  const handleShowSettings = () => {
    setIsSettingsModalOpen(true);
  };

  // Show loading spinner while checking auth
  if (authLoading) {
    return (
      <ThemeProvider>
        <div className="min-h-screen bg-background flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
            <p className="text-muted-foreground">Loading...</p>
          </div>
        </div>
      </ThemeProvider>
    );
  }

  // Show login dialog if not authenticated
  if (!isAuthenticated) {
    return (
      <ThemeProvider>
        <div className="min-h-screen bg-background">
          <LoginDialog open={true} />
        </div>
      </ThemeProvider>
    );
  }

  // Show main app if authenticated
  return (
    <ThemeProvider>
      <div className="min-h-screen bg-background">
        <Header
          onCreateInstance={handleCreateInstance}
          onShowSystemInfo={handleShowSystemInfo}
          onShowSettings={handleShowSettings}
        />
        <main className="container mx-auto max-w-4xl px-4 py-8">
          {/* Tab Navigation */}
          <div className="border-b border-border mb-6">
            <div className="flex gap-4">
              <button
                onClick={() => setActiveTab('instances')}
                className={cn(
                  "px-4 py-2 border-b-2 transition-colors",
                  activeTab === 'instances'
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                )}
              >
                Instances
              </button>
              <button
                onClick={() => setActiveTab('models')}
                className={cn(
                  "px-4 py-2 border-b-2 transition-colors",
                  activeTab === 'models'
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground"
                )}
              >
                Models
              </button>
            </div>
          </div>

          {/* Tab Content */}
          {activeTab === 'instances' && <InstanceList editInstance={handleEditInstance} />}
          {activeTab === 'models' && <ModelsList />}
        </main>

        <InstanceDialog
          open={isInstanceModalOpen}
          onOpenChange={setIsInstanceModalOpen}
          onSave={handleSaveInstance}
          instance={editingInstance}
        />

        <SystemInfoDialog
          open={isSystemInfoModalOpen}
          onOpenChange={setIsSystemInfoModalOpen}
        />

        <SettingsDialog
          open={isSettingsModalOpen}
          onOpenChange={setIsSettingsModalOpen}
        />

        <Toaster />
      </div>
    </ThemeProvider>
  );
}

export default App;
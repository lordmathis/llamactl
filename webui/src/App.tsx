import { useState } from "react";
import Header from "@/components/Header";
import InstanceList from "@/components/InstanceList";
import InstanceDialog from "@/components/InstanceDialog";
import LoginDialog from "@/components/LoginDialog";
import SystemInfoDialog from "./components/SystemInfoDialog";
import { type CreateInstanceOptions, type Instance } from "@/types/instance";
import { useInstances } from "@/contexts/InstancesContext";
import { useAuth } from "@/contexts/AuthContext";

function App() {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const [isInstanceModalOpen, setIsInstanceModalOpen] = useState(false);
  const [isSystemInfoModalOpen, setIsSystemInfoModalOpen] = useState(false);
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
      updateInstance(editingInstance.name, options);
    } else {
      createInstance(name, options);
    }
  };

  const handleShowSystemInfo = () => {
    setIsSystemInfoModalOpen(true);
  };

  // Show loading spinner while checking auth
  if (authLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  // Show login dialog if not authenticated
  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-gray-50">
        <LoginDialog open={true} />
      </div>
    );
  }

  // Show main app if authenticated
  return (
    <div className="min-h-screen bg-gray-50">
      <Header onCreateInstance={handleCreateInstance} onShowSystemInfo={handleShowSystemInfo} />
      <main className="container mx-auto max-w-4xl px-4 py-8">
        <InstanceList editInstance={handleEditInstance} />
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
    </div>
  );
}

export default App;
import { useState } from "react";
import Header from "@/components/Header";
import InstanceList from "@/components/InstanceList";
import InstanceModal from "@/components/InstanceModal";
import { CreateInstanceOptions, Instance } from "@/types/instance";
import { useInstances } from "@/contexts/InstancesContext";
import SystemInfoModal from "./components/SystemInfoModal";

function App() {
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

  return (
    <div className="min-h-screen bg-gray-50">
      <Header onCreateInstance={handleCreateInstance} onShowSystemInfo={handleShowSystemInfo} />
      <main className="container mx-auto max-w-4xl px-4 py-8">
        <InstanceList editInstance={handleEditInstance} />
      </main>

      <InstanceModal
        open={isInstanceModalOpen}
        onOpenChange={setIsInstanceModalOpen}
        onSave={handleSaveInstance}
        instance={editingInstance}
      />

      <SystemInfoModal
        open={isSystemInfoModalOpen}
        onOpenChange={setIsSystemInfoModalOpen}
      />
    </div>
  );
}

export default App;

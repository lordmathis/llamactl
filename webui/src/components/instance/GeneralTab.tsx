import type React from "react";
import { useEffect, useState } from "react";
import type { CreateInstanceOptions } from "@/types/instance";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import AutoRestartConfiguration from "@/components/instance/AutoRestartConfiguration";
import NumberInput from "@/components/form/NumberInput";
import CheckboxInput from "@/components/form/CheckboxInput";
import SelectInput from "@/components/form/SelectInput";
import { nodesApi, type NodesMap } from "@/lib/api";

interface GeneralTabProps {
  instanceName: string;
  nameError: string;
  isEditing: boolean;
  formData: CreateInstanceOptions;
  onNameChange: (name: string) => void;
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void;
}

const GeneralTab: React.FC<GeneralTabProps> = ({
  instanceName,
  nameError,
  isEditing,
  formData,
  onNameChange,
  onChange,
}) => {
  const [nodes, setNodes] = useState<NodesMap>({});
  const [loadingNodes, setLoadingNodes] = useState(true);

  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const fetchedNodes = await nodesApi.list();
        setNodes(fetchedNodes);

        const nodeNames = Object.keys(fetchedNodes);
        if (
          nodeNames.length > 0 &&
          (!formData.nodes || formData.nodes.length === 0)
        ) {
          onChange("nodes", [nodeNames[0]]);
        }
      } catch (error) {
        console.error("Failed to fetch nodes:", error);
      } finally {
        setLoadingNodes(false);
      }
    };

    void fetchNodes();
  }, [formData.nodes, onChange]);

  const nodeOptions = Object.keys(nodes).map((nodeName) => ({
    value: nodeName,
    label: nodeName,
  }));

  const handleNodeChange = (value: string | undefined) => {
    if (value) {
      onChange("nodes", [value]);
    } else {
      onChange("nodes", undefined);
    }
  };

  const selectedNode =
    formData.nodes && formData.nodes.length > 0 ? formData.nodes[0] : "";

  return (
    <div className="space-y-6 py-4">
      <div className="grid gap-2">
        <Label htmlFor="name">
          Instance Name <span className="text-red-500">*</span>
        </Label>
        <Input
          id="name"
          value={instanceName}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="my-instance"
          disabled={isEditing}
          className={nameError ? "border-red-500" : ""}
        />
        {nameError && <p className="text-sm text-red-500">{nameError}</p>}
        <p className="text-sm text-muted-foreground">
          A short, unique name for this instance (letters, numbers, hyphens). Cannot be changed after creation.
        </p>
      </div>

      {!loadingNodes && Object.keys(nodes).length > 0 && (
        <SelectInput
          id="node"
          label="Node"
          value={selectedNode}
          onChange={handleNodeChange}
          options={nodeOptions}
          description={
            isEditing
              ? "Node cannot be changed after instance creation"
              : "Select the node where the instance will run"
          }
          disabled={isEditing}
        />
      )}

      <div className="grid gap-2">
        <Label htmlFor="group">Group</Label>
        <Input
          id="group"
          value={formData.group || ""}
          onChange={(e) => onChange("group", e.target.value || undefined)}
          placeholder="e.g., large-models"
        />
        <p className="text-sm text-muted-foreground">
          Optional label for grouping instances (e.g. <code>large-models</code>). Groups can have
          shared running-instance limits configured in the server config under <code>instances.group_limits</code>.
        </p>
      </div>

      <div className="space-y-4">
        <h3 className="text-lg font-medium">Basic Instance Options</h3>

        <NumberInput
          id="idle_timeout"
          label="Idle Timeout (minutes)"
          value={formData.idle_timeout}
          onChange={(value) => onChange("idle_timeout", value)}
          placeholder="30"
          description="Minutes of inactivity before the instance is automatically stopped to free resources (0 = never stop automatically)."
        />

        <CheckboxInput
          id="on_demand_start"
          label="On Demand Start"
          value={formData.on_demand_start}
          onChange={(value) => onChange("on_demand_start", value)}
          description="When enabled, llamactl starts this instance automatically the first time a request arrives for it, instead of requiring a manual start."
        />
      </div>

      <AutoRestartConfiguration formData={formData} onChange={onChange} />
    </div>
  );
};

export default GeneralTab;

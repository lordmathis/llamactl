import React, { useState, useEffect } from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import AutoRestartConfiguration from '@/components/instance/AutoRestartConfiguration'
import NumberInput from '@/components/form/NumberInput'
import CheckboxInput from '@/components/form/CheckboxInput'
import EnvironmentVariablesInput from '@/components/form/EnvironmentVariablesInput'
import SelectInput from '@/components/form/SelectInput'
import { nodesApi, type NodesMap } from '@/lib/api'

interface InstanceSettingsCardProps {
  instanceName: string
  nameError: string
  isEditing: boolean
  formData: CreateInstanceOptions
  onNameChange: (name: string) => void
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}

const InstanceSettingsCard: React.FC<InstanceSettingsCardProps> = ({
  instanceName,
  nameError,
  isEditing,
  formData,
  onNameChange,
  onChange
}) => {
  const [nodes, setNodes] = useState<NodesMap>({})
  const [loadingNodes, setLoadingNodes] = useState(true)

  useEffect(() => {
    const fetchNodes = async () => {
      try {
        const fetchedNodes = await nodesApi.list()
        setNodes(fetchedNodes)

        // Auto-select first node if none selected
        const nodeNames = Object.keys(fetchedNodes)
        if (nodeNames.length > 0 && (!formData.nodes || formData.nodes.length === 0)) {
          onChange('nodes', [nodeNames[0]])
        }
      } catch (error) {
        console.error('Failed to fetch nodes:', error)
      } finally {
        setLoadingNodes(false)
      }
    }

    void fetchNodes()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const nodeOptions = Object.keys(nodes).map(nodeName => ({
    value: nodeName,
    label: nodeName
  }))

  const handleNodeChange = (value: string | undefined) => {
    if (value) {
      onChange('nodes', [value])
    } else {
      onChange('nodes', undefined)
    }
  }

  const selectedNode = formData.nodes && formData.nodes.length > 0 ? formData.nodes[0] : ''

  return (
    <Card>
      <CardHeader>
        <CardTitle>Instance Settings</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Instance Name */}
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
            Unique identifier for the instance
          </p>
        </div>

        {/* Node Selection */}
        {!loadingNodes && Object.keys(nodes).length > 0 && (
          <SelectInput
            id="node"
            label="Node"
            value={selectedNode}
            onChange={handleNodeChange}
            options={nodeOptions}
            description="Select the node where the instance will run (default: main node)"
          />
        )}

        {/* Auto Restart Configuration */}
        <AutoRestartConfiguration
          formData={formData}
          onChange={onChange}
        />

        {/* Basic Instance Options */}
        <div className="space-y-4">
          <h3 className="text-lg font-medium">Basic Instance Options</h3>

          <NumberInput
            id="idle_timeout"
            label="Idle Timeout (minutes)"
            value={formData.idle_timeout}
            onChange={(value) => onChange('idle_timeout', value)}
            placeholder="30"
            description="Minutes before stopping an idle instance"
          />

          <CheckboxInput
            id="on_demand_start"
            label="On Demand Start"
            value={formData.on_demand_start}
            onChange={(value) => onChange('on_demand_start', value)}
            description="Start instance only when needed"
          />

          <EnvironmentVariablesInput
            id="environment"
            label="Environment Variables"
            value={formData.environment}
            onChange={(value) => onChange('environment', value)}
            description="Custom environment variables for the instance"
          />
        </div>
      </CardContent>
    </Card>
  )
}

export default InstanceSettingsCard
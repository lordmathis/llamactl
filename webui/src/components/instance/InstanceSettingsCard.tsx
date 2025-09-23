import React from 'react'
import type { CreateInstanceOptions } from '@/types/instance'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import AutoRestartConfiguration from '@/components/instance/AutoRestartConfiguration'
import NumberInput from '@/components/form/NumberInput'
import CheckboxInput from '@/components/form/CheckboxInput'

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
        </div>
      </CardContent>
    </Card>
  )
}

export default InstanceSettingsCard
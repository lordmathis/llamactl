import React, { useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Upload, Download } from 'lucide-react'
import IniEditor from './IniEditor'
import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

interface PresetTabProps {
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}

const PresetTab: React.FC<PresetTabProps> = ({ formData, onChange }) => {
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleImport = () => {
    fileInputRef.current?.click()
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const reader = new FileReader()
      reader.onload = (event) => {
        const content = event.target?.result as string
        onChange('preset_ini', content)
      }
      reader.readAsText(file)
    }
  }

  const handleExport = () => {
    const content = formData.preset_ini || ''
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'preset.ini'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <div className="h-full flex flex-col space-y-4">
      <div>
        <h3 className="text-lg font-semibold">Model Presets (preset.ini)</h3>
        <p className="text-sm text-muted-foreground mt-1">
          Optional: Configure multiple models for router mode. Leave empty to run without router mode.
          This will auto-set models-preset option unless you specify a custom path.
        </p>
      </div>

      <div className="flex gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={handleImport}
          className="flex items-center gap-2"
        >
          <Upload className="h-4 w-4" />
          Import
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={handleExport}
          className="flex items-center gap-2"
        >
          <Download className="h-4 w-4" />
          Export
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          accept=".ini"
          onChange={handleFileChange}
          className="hidden"
        />
      </div>

      <div className="flex-1 min-h-0">
        <IniEditor
          value={formData.preset_ini || ''}
          onChange={(value) => onChange('preset_ini', value)}
          className="h-full"
        />
      </div>
    </div>
  )
}

export default PresetTab

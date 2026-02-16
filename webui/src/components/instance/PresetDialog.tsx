import React, { useRef } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Upload, Download } from 'lucide-react'
import IniEditor from './IniEditor'
import type { CreateInstanceOptions } from '@/schemas/instanceOptions'

interface PresetDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  formData: CreateInstanceOptions
  onChange: (key: keyof CreateInstanceOptions, value: unknown) => void
}

const PresetDialog: React.FC<PresetDialogProps> = ({
  open,
  onOpenChange,
  formData,
  onChange
}) => {
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
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Model Presets (preset.ini)</DialogTitle>
          <DialogDescription>
            Optional: Configure multiple models for router mode. Leave empty to run without router mode.
            This will auto-set models-preset option unless you specify a custom path.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 min-h-0 overflow-y-auto">
          <IniEditor
            value={formData.preset_ini || ''}
            onChange={(value) => onChange('preset_ini', value)}
            className="h-full"
          />
        </div>

        <DialogFooter>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={handleImport}
              className="flex items-center gap-2"
            >
              <Upload className="h-4 w-4" />
              Import
            </Button>
            <Button
              variant="outline"
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
          <Button onClick={() => onOpenChange(false)}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default PresetDialog

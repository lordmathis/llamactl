import React, { useState, useRef, useEffect, useCallback } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Alert } from '@/components/ui/alert'
import { Send, Square, Trash2, Loader2, AlertCircle, ChevronDown } from 'lucide-react'
import { streamChat, type ChatMessage, type Model } from '@/lib/api'

interface ChatDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  instanceName: string
  /** All currently loaded models for this instance. */
  loadedModels: Model[]
}

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
  streaming?: boolean
  error?: boolean
}

const ChatDialog: React.FC<ChatDialogProps> = ({
  open,
  onOpenChange,
  instanceName,
  loadedModels,
}) => {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [isStreaming, setIsStreaming] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Pick the first loaded model as default when models change
  useEffect(() => {
    if (loadedModels.length > 0 && !selectedModel) {
      setSelectedModel(loadedModels[0].id)
    }
  }, [loadedModels, selectedModel])

  // Auto-scroll to bottom on new content
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // Focus textarea when dialog opens
  useEffect(() => {
    if (open) {
      setTimeout(() => textareaRef.current?.focus(), 100)
    }
  }, [open])

  const activeModel = selectedModel || loadedModels[0]?.id || ''

  const sendMessage = useCallback(() => {
    const trimmed = input.trim()
    if (!trimmed || isStreaming || !activeModel) return

    setError(null)
    setInput('')

    const userMsg: Message = {
      id: crypto.randomUUID(),
      role: 'user',
      content: trimmed,
    }

    const assistantId = crypto.randomUUID()
    const assistantMsg: Message = {
      id: assistantId,
      role: 'assistant',
      content: '',
      streaming: true,
    }

    setMessages(prev => [...prev, userMsg, assistantMsg])
    setIsStreaming(true)

    // Build the full conversation history for context
    const history: ChatMessage[] = [
      ...messages.map(m => ({ role: m.role, content: m.content })),
      { role: 'user', content: trimmed },
    ]

    abortRef.current = streamChat(
      instanceName,
      history,
      activeModel,
      (chunk) => {
        setMessages(prev =>
          prev.map(m =>
            m.id === assistantId
              ? { ...m, content: m.content + chunk }
              : m
          )
        )
      },
      () => {
        setIsStreaming(false)
        setMessages(prev =>
          prev.map(m =>
            m.id === assistantId ? { ...m, streaming: false } : m
          )
        )
      },
      (err) => {
        setIsStreaming(false)
        setMessages(prev =>
          prev.map(m =>
            m.id === assistantId
              ? { ...m, streaming: false, content: m.content || err, error: true }
              : m
          )
        )
        setError(err)
      }
    )
  }, [input, isStreaming, activeModel, messages, instanceName])

  const handleStop = () => {
    abortRef.current?.abort()
    setIsStreaming(false)
    setMessages(prev =>
      prev.map(m => m.streaming ? { ...m, streaming: false } : m)
    )
  }

  const handleClear = () => {
    handleStop()
    setMessages([])
    setError(null)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl max-w-[calc(100%-2rem)] max-h-[85vh] flex flex-col gap-0 p-0">
        {/* Header */}
        <DialogHeader className="px-4 pt-4 pb-3 border-b">
          <div className="flex items-start justify-between gap-2">
            <div className="flex-1 min-w-0">
              <DialogTitle className="text-base">
                Chat — {instanceName}
              </DialogTitle>
              <DialogDescription className="sr-only">
                Chat with the {instanceName} instance
              </DialogDescription>
            </div>

            {/* Model selector (only shown for multi-model instances) */}
            {loadedModels.length > 1 && (
              <div className="relative shrink-0">
                <select
                  value={selectedModel}
                  onChange={e => setSelectedModel(e.target.value)}
                  className="appearance-none text-xs rounded-md border border-input bg-background px-2 py-1 pr-6 text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  title="Select model"
                >
                  {loadedModels.map(m => (
                    <option key={m.id} value={m.id}>{m.id}</option>
                  ))}
                </select>
                <ChevronDown className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground" />
              </div>
            )}

            {/* Single model badge */}
            {loadedModels.length === 1 && (
              <Badge variant="secondary" className="text-xs shrink-0 max-w-[14rem] truncate" title={activeModel}>
                {activeModel}
              </Badge>
            )}
          </div>
        </DialogHeader>

        {/* Message thread */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4 min-h-0">
          {messages.length === 0 && (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground py-12">
              Start a conversation — press Enter to send, Shift+Enter for a new line.
            </div>
          )}

          {messages.map(msg => (
            <div
              key={msg.id}
              className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div
                className={`max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap break-words ${
                  msg.role === 'user'
                    ? 'bg-primary text-primary-foreground'
                    : msg.error
                    ? 'bg-destructive/10 border border-destructive/20 text-destructive'
                    : 'bg-muted text-foreground'
                }`}
              >
                {msg.content}
                {msg.streaming && (
                  <span className="inline-block w-2 h-4 ml-0.5 bg-current opacity-70 animate-pulse rounded-sm" />
                )}
              </div>
            </div>
          ))}

          {error && !isStreaming && (
            <Alert variant="destructive" className="py-2">
              <AlertCircle className="h-4 w-4" />
              <span className="ml-2 text-sm">{error}</span>
            </Alert>
          )}

          <div ref={bottomRef} />
        </div>

        {/* Input area */}
        <div className="px-4 pb-4 pt-3 border-t space-y-2">
          <div className="flex gap-2">
            <Textarea
              ref={textareaRef}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={activeModel ? 'Type a message…' : 'No model loaded'}
              disabled={isStreaming || !activeModel}
              rows={2}
              className="resize-none text-sm flex-1"
            />

            <div className="flex flex-col gap-1.5">
              {isStreaming ? (
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={handleStop}
                  title="Stop generation"
                  className="h-full"
                >
                  <Square className="h-4 w-4" />
                </Button>
              ) : (
                <Button
                  size="sm"
                  onClick={sendMessage}
                  disabled={!input.trim() || !activeModel}
                  title="Send message"
                  className="h-full"
                >
                  {isStreaming ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                </Button>
              )}
            </div>
          </div>

          <div className="flex items-center justify-between">
            <p className="text-xs text-muted-foreground">
              Enter to send · Shift+Enter for new line
            </p>
            <Button
              size="sm"
              variant="ghost"
              onClick={handleClear}
              disabled={messages.length === 0}
              className="h-6 text-xs text-muted-foreground"
            >
              <Trash2 className="h-3 w-3 mr-1" />
              Clear
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export default ChatDialog

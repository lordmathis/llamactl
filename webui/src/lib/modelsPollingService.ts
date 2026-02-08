import type { DownloadJob } from '@/types/model'
import { llamaCppModelsApi } from '@/lib/api'

type JobsCallback = (jobs: DownloadJob[]) => void

class ModelsPollingService {
  private interval: NodeJS.Timeout | null = null
  private callbacks: Set<JobsCallback> = new Set()
  private readonly POLL_INTERVAL = 2000 // 2 seconds
  private expectingJobs = false // Flag to keep polling when a download just started
  private emptyPollCount = 0 // Count consecutive empty polls

  /**
   * Subscribe to job updates
   */
  subscribe(callback: JobsCallback): () => void {
    this.callbacks.add(callback)

    // Start polling if this is the first subscriber
    if (this.callbacks.size === 1) {
      this.startPolling()
    }

    // Return unsubscribe function
    return () => {
      this.callbacks.delete(callback)

      // Stop polling if no more subscribers
      if (this.callbacks.size === 0) {
        this.stopPolling()
      }
    }
  }

  /**
   * Poll for active jobs
   */
  private async pollJobs(): Promise<void> {
    try {
      const { jobs } = await llamaCppModelsApi.listJobs()

      // Filter to active and recently completed/failed jobs (exclude only completed)
      const activeJobs = jobs.filter(j =>
        j.status === 'downloading' || j.status === 'queued' || j.status === 'failed' || j.status === 'cancelled'
      )

      // Notify all callbacks
      this.callbacks.forEach(cb => cb(activeJobs))

      // Handle stopping logic
      if (activeJobs.length === 0) {
        this.emptyPollCount++

        // Only stop if:
        // 1. We're not expecting jobs (no recent download started), AND
        // 2. We've had at least 2 consecutive empty polls (to handle timing issues)
        if (!this.expectingJobs && this.emptyPollCount >= 2) {
          this.stopPolling()
        }
      } else {
        // Reset counters when we have active jobs
        this.emptyPollCount = 0
        this.expectingJobs = false
      }
    } catch (error) {
      console.error('Failed to poll jobs:', error)
      // Continue polling even on error - don't break the polling loop
    }
  }

  /**
   * Start polling for jobs
   */
  private startPolling(): void {
    // Don't start if already polling
    if (this.interval) {
      return
    }

    // Reset counters
    this.emptyPollCount = 0

    // Perform initial poll immediately
    void this.pollJobs()

    // Set up interval for subsequent polls
    this.interval = setInterval(() => {
      void this.pollJobs()
    }, this.POLL_INTERVAL)
  }

  /**
   * Stop polling for jobs
   */
  private stopPolling(): void {
    if (this.interval) {
      clearInterval(this.interval)
      this.interval = null
    }
  }

  /**
   * Notify that a download was just started (so we should keep polling)
   */
  downloadStarted(): void {
    this.expectingJobs = true
    this.emptyPollCount = 0

    // Ensure polling is active
    if (!this.interval && this.callbacks.size > 0) {
      this.startPolling()
    } else {
      // Trigger immediate poll
      void this.pollJobs()
    }
  }

  /**
   * Clean up service
   */
  destroy(): void {
    this.stopPolling()
    this.callbacks.clear()
  }
}

export const modelsPollingService = new ModelsPollingService()

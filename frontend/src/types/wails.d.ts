type WailsCallback = (...data: any[]) => void

declare global {
  interface WailsRuntime {
    EventsOn(eventName: string, callback: WailsCallback): () => void
    EventsOnMultiple(eventName: string, callback: WailsCallback, maxCallbacks: number): () => void
    EventsOnce(eventName: string, callback: WailsCallback): () => void
    EventsOff(eventName: string, ...additionalEventNames: string[]): void
    EventsOffAll(): void
    EventsEmit(eventName: string, ...data: any[]): void
  }

  interface Window {
    go?: Record<string, Record<string, Record<string, (...args: any[]) => Promise<any>>>>
    runtime?: WailsRuntime
  }
}

export {}

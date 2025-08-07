/// <reference types="vite/client" />

declare global {
  interface ImportMetaEnv {
    readonly VITE_APP_VERSION?: string
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv
  }
}

export {}

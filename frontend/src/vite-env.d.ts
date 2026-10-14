/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Empty or unset: same-origin requests. */
  readonly VITE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

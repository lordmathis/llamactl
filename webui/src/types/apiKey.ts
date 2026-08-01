export enum PermissionMode {
  AllowAll = "allow_all",
  PerInstance = "per_instance"
}

export interface ApiKey {
  id: number
  name: string
  user_id: string
  permission_mode: PermissionMode
  expires_at: number | null
  created_at: number
  updated_at: number
  last_used_at: number | null
}

export interface CreateKeyRequest {
  name: string
  permission_mode: PermissionMode
  expires_at?: number
  permissions?: InstancePermissionSpec[]
}

export interface InstancePermissionSpec {
  instance_id: number
  can_start?: boolean   // defaults to true
  can_evict?: boolean   // defaults to true
}

export interface CreateKeyResponse extends ApiKey {
  key: string
}

export interface KeyPermissionResponse {
  instance_id: number
  instance_name: string
  can_start: boolean
  can_evict: boolean
}

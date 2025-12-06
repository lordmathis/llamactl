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
  Name: string
  PermissionMode: PermissionMode
  ExpiresAt?: number
  InstancePermissions: InstancePermission[]
}

export interface InstancePermission {
  InstanceID: number
}

export interface CreateKeyResponse extends ApiKey {
  key: string
}

export interface KeyPermissionResponse {
  instance_id: number
  instance_name: string
}

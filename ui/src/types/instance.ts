import { CreateInstanceOptions } from '@/schemas/instanceOptions'

export { type CreateInstanceOptions } from '@/schemas/instanceOptions'

export interface Instance {
  name: string;
  running: boolean;
  options?: CreateInstanceOptions;
}
import type { Flag, ValueStruct } from './device'

export interface TemplateParameter {
  uuid: string
  template_id: number
  name: string
  valuestruct: ValueStruct
  flag: Flag
}

export interface Template {
  id: number
  name: string
  parameters: TemplateParameter[] | null
}

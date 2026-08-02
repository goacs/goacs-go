export interface ProvisionRule {
  id?: number
  provision_id?: number
  parameter: string
  operator: string
  value: string
}

export interface Provision {
  id: number
  name: string
  events: string
  requests: string
  script: string[]
  rules: ProvisionRule[] | null
  created_at: string
  updated_at: string
}

export interface ProvisionStoreRequest {
  name: string
  events: string
  requests: string
  script: string[]
  rules: ProvisionRule[]
}

// CWMP event codes selectable when building a provisioning rule's trigger.
export const CWMP_EVENTS = [
  { label: 'BOOTSTRAP', value: '0 BOOTSTRAP' },
  { label: 'BOOT', value: '1 BOOT' },
  { label: 'PERIODIC', value: '2 PERIODIC' },
  { label: 'VALUE CHANGE', value: '4 VALUE CHANGE' },
  { label: 'CONNECTION REQUEST', value: '6 CONNECTION REQUEST' },
  { label: 'TRANSFER COMPLETE', value: '7 TRANSFER COMPLETE' },
]

export const CWMP_REQUESTS = [
  { label: 'Empty', value: 'Empty' },
  { label: 'Inform', value: 'Inform' },
  { label: 'GetParameterValuesResponse', value: 'GetParameterValuesResponse' },
  { label: 'SetParameterValuesProcessor', value: 'SetParameterValuesProcessor' },
  { label: 'TransferComplete', value: 'TransferComplete' },
]

export const RULE_OPERATORS = ['>', '>=', '<', '<=', '==', '!=', 'in', 'not in']

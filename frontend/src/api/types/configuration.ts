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
  priority: number
  enabled: boolean
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

export interface ProvisionSimulateParam {
  key: string
  value: string
}

export interface ProvisionSimulateRequest {
  event: string
  request: string
  root: string
  params: ProvisionSimulateParam[]
}

export interface ProvisionSimulateConditionResult {
  parameter: string
  operator: string
  value: string
  actual: string
  passed: boolean
}

// Matches http/controllers/provision.go's ProvisionSimulateResult. Computed
// server-side by acs/logic.EvaluateProvisionMatch - the same function the real
// ProvisionMatcher uses - so the simulator can't drift from production matching.
export interface ProvisionSimulateResult {
  provision_id: number
  name: string
  priority: number
  enabled: boolean
  script_count: number
  event_match: boolean
  request_match: boolean
  condition_results: ProvisionSimulateConditionResult[]
  conditions_match: boolean
  overall_match: boolean
}

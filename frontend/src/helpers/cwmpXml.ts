export interface ParsedDeviceId {
  manufacturer?: string
  oui?: string
  productClass?: string
  serialNumber?: string
}

export interface ParsedParameter {
  name: string
  value: string
  type: string
}

export interface ParsedInform {
  method: string
  deviceId?: ParsedDeviceId
  events: string[]
  parameters: ParsedParameter[]
}

const XSI_NS = 'http://www.w3.org/2001/XMLSchema-instance'

function textOf(el: Element, tag: string): string | undefined {
  return el.getElementsByTagNameNS('*', tag)[0]?.textContent?.trim() || undefined
}

/**
 * Best-effort parse of a CWMP SOAP envelope (as stored verbatim in LogEntry.full_xml).
 * Returns null when the envelope isn't a recognized shape (Faults, RPC types we don't
 * model yet, or malformed XML) - callers should fall back to showing the raw XML.
 */
export function parseCwmpXml(xml: string): ParsedInform | null {
  try {
    const doc = new DOMParser().parseFromString(xml, 'application/xml')
    if (doc.getElementsByTagName('parsererror').length > 0) return null

    const body = doc.getElementsByTagNameNS('*', 'Body')[0]
    const rpcEl = body ? Array.from(body.children).find((el) => el.localName !== 'Fault') : undefined
    if (!rpcEl) return null

    const deviceIdEl = rpcEl.getElementsByTagNameNS('*', 'DeviceId')[0]
    const deviceId: ParsedDeviceId | undefined = deviceIdEl
      ? {
          manufacturer: textOf(deviceIdEl, 'Manufacturer'),
          oui: textOf(deviceIdEl, 'OUI'),
          productClass: textOf(deviceIdEl, 'ProductClass'),
          serialNumber: textOf(deviceIdEl, 'SerialNumber'),
        }
      : undefined

    const events = Array.from(rpcEl.getElementsByTagNameNS('*', 'EventStruct'))
      .map((el) => textOf(el, 'EventCode') ?? '')
      .filter(Boolean)

    const parameters: ParsedParameter[] = Array.from(rpcEl.getElementsByTagNameNS('*', 'ParameterValueStruct')).map(
      (el) => {
        const valueEl = el.getElementsByTagNameNS('*', 'Value')[0]
        return {
          name: textOf(el, 'Name') ?? '',
          value: valueEl?.textContent?.trim() ?? '',
          type: valueEl?.getAttributeNS(XSI_NS, 'type')?.replace(/^xsd:/, '') ?? '',
        }
      },
    )

    if (!deviceId && events.length === 0 && parameters.length === 0) return null

    return { method: rpcEl.localName, deviceId, events, parameters }
  } catch {
    return null
  }
}

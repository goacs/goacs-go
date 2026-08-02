export interface Flag {
  read: boolean
  write: boolean
  add_object: boolean
  system: boolean
  periodic_read: boolean
  important: boolean
  send: boolean
}

export function emptyFlag(): Flag {
  return {
    read: false,
    write: false,
    add_object: false,
    system: false,
    periodic_read: false,
    important: false,
    send: false,
  }
}

export interface ValueStruct {
  value: string
  type: string
}

export interface Parameter {
  name: string
  valuestruct: ValueStruct
  flag: Flag
}

// Mirrors Go's acs/types.IPAddress (a net.IPAddr wrapper) - marshals as
// {"IP": "1.2.3.4", "Zone": ""}, not a plain string.
export interface IPAddress {
  IP: string
  Zone: string
}

export interface CPE {
  uuid: string
  serial_number: string
  oui: string
  manufacturer: string
  software_version: string
  hardware_version: string
  ip_address: IPAddress
  connection_request_user: string
  connection_request_password: string
  connection_request_url: string
  debug: boolean
  updated_at: string
}

export interface CPETemplate {
  id: number
  name: string
  priority: number
}

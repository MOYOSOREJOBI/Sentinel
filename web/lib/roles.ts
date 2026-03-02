export type Role = 'admin' | 'analyst' | 'viewer'
export type Tab = 'overview' | 'alerts' | 'scores' | 'governance'
export type Capability =
  | 'alerts:ack'
  | 'alerts:resolve'
  | 'alerts:suppress'
  | 'replay:start'
  | 'audit:verify'
  | 'models:deploy'

const roleCapabilities: Record<Role, Set<Capability>> = {
  viewer: new Set(),
  analyst: new Set(['alerts:ack', 'replay:start']),
  admin: new Set(['alerts:ack', 'alerts:resolve', 'alerts:suppress', 'replay:start', 'audit:verify', 'models:deploy']),
}

export function navItemsForRole(role: Role): Tab[] {
  if (role === 'admin') return ['overview', 'alerts', 'scores', 'governance']
  if (role === 'analyst') return ['overview', 'alerts', 'scores']
  return ['overview', 'scores']
}

export function hasCapability(role: Role, capability: Capability): boolean {
  return roleCapabilities[role].has(capability)
}

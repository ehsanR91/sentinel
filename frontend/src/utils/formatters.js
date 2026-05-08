const TIME_FORMAT_KEY = 'sc_pref_time_format'
const TIMEZONE_KEY = 'sc_pref_timezone'
const DENSITY_KEY = 'sc_pref_density'
const APPS_VIEW_KEY = 'sc_apps_view'

const reasonMap = {
  '': {
    label: 'Authenticated',
    category: 'auth',
    icon: 'mdi mdi-check-circle-outline',
    description: 'Authentication completed successfully.'
  },
  bad_credentials: {
    label: 'Bad Credentials',
    category: 'auth',
    icon: 'mdi mdi-account-alert-outline',
    description: 'The submitted username or password was rejected.'
  },
  '2fa_required': {
    label: 'Two-Factor Required',
    category: '2fa',
    icon: 'mdi mdi-shield-key-outline',
    description: 'Primary credentials were valid, but a second factor is required.'
  },
  bad_totp: {
    label: 'Bad TOTP Code',
    category: '2fa',
    icon: 'mdi mdi-shield-alert-outline',
    description: 'A TOTP code was supplied but did not validate.'
  },
  rate_limited: {
    label: 'Rate Limited',
    category: 'security',
    icon: 'mdi mdi-timer-sand',
    description: 'The request was rejected because the IP crossed the brute-force threshold.'
  },
  terminal_connect: {
    label: 'Terminal Connected',
    category: 'session',
    icon: 'mdi mdi-console',
    description: 'The audited terminal session was opened.'
  },
  terminal_elevate_ok: {
    label: 'Privilege Elevation',
    category: 'session',
    icon: 'mdi mdi-shield-check-outline',
    description: 'The user elevated the terminal session successfully.'
  },
  terminal_elevate_fail: {
    label: 'Elevation Failed',
    category: 'session',
    icon: 'mdi mdi-shield-remove-outline',
    description: 'The terminal privilege elevation attempt was denied.'
  }
}

function safeGet (key, fallback) {
  try {
    return localStorage.getItem(key) || fallback
  } catch {
    return fallback
  }
}

function safeSet (key, value) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // noop
  }
}

function toDate (value) {
  if (value instanceof Date) return value
  if (typeof value === 'number') {
    return new Date(value < 1e12 ? value * 1000 : value)
  }
  return new Date(value)
}

export function getTimeFormatPreference () {
  return safeGet(TIME_FORMAT_KEY, 'both')
}

export function setTimeFormatPreference (value) {
  safeSet(TIME_FORMAT_KEY, value)
}

export function getTimezonePreference () {
  return safeGet(TIMEZONE_KEY, 'local')
}

export function setTimezonePreference (value) {
  safeSet(TIMEZONE_KEY, value)
}

export function getDensityPreference () {
  return safeGet(DENSITY_KEY, 'comfortable')
}

export function setDensityPreference (value) {
  safeSet(DENSITY_KEY, value)
}

export function getAppsViewPreference () {
  return safeGet(APPS_VIEW_KEY, 'grid')
}

export function setAppsViewPreference (value) {
  safeSet(APPS_VIEW_KEY, value)
}

export function formatRelativeTime (value) {
  if (!value) return 'Unknown'
  const date = toDate(value)
  if (Number.isNaN(date.getTime())) return 'Unknown'
  const diff = Math.round((Date.now() - date.getTime()) / 1000)
  const abs = Math.abs(diff)
  if (abs < 60) return `${abs}s ago`
  if (abs < 3600) return `${Math.floor(abs / 60)}m ago`
  if (abs < 86400) return `${Math.floor(abs / 3600)}h ago`

  const yesterday = new Date()
  yesterday.setDate(yesterday.getDate() - 1)
  if (date.toDateString() === yesterday.toDateString()) {
    return `Yesterday ${new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(date)}`
  }
  if (abs < 604800) {
    return `${Math.floor(abs / 86400)}d ago`
  }
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(date)
}

export function formatAbsoluteTime (value, options = {}) {
  if (!value) return 'Unknown'
  const date = toDate(value)
  if (Number.isNaN(date.getTime())) return 'Unknown'
  const zoneMode = options.timezoneMode || getTimezonePreference()
  const formatter = new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    timeZone: zoneMode === 'utc' ? 'UTC' : undefined,
    timeZoneName: 'short'
  })
  return formatter.format(date)
}

export function formatTimestamp (value, options = {}) {
  const mode = options.mode || getTimeFormatPreference()
  const absolute = formatAbsoluteTime(value, options)
  const relative = formatRelativeTime(value)
  if (mode === 'relative') {
    return { primary: relative, secondary: null, title: absolute }
  }
  if (mode === 'absolute') {
    return { primary: absolute, secondary: null, title: absolute }
  }
  return { primary: relative, secondary: absolute, title: absolute }
}

export function parseUserAgent (ua = '') {
  const source = ua.toLowerCase()
  const browser = /edg\//.test(source) ? 'Edge'
    : /chrome\//.test(source) && !/edg\//.test(source) ? 'Chrome'
      : /firefox\//.test(source) ? 'Firefox'
        : /safari\//.test(source) && !/chrome\//.test(source) ? 'Safari'
          : /postmanruntime/.test(source) ? 'Postman'
            : /curl\//.test(source) ? 'curl'
              : source === 'ws' ? 'WebSocket'
                : 'Unknown'
  const versionMatch = ua.match(/(?:edg|chrome|firefox|version|postmanruntime|curl)\/([\d.]+)/i)
  const version = versionMatch ? versionMatch[1].split('.').slice(0, 2).join('.') : ''
  const os = /windows nt/i.test(source) ? 'Windows'
    : /mac os x/i.test(source) ? 'macOS'
      : /android/i.test(source) ? 'Android'
        : /iphone|ipad|ios/i.test(source) ? 'iOS'
          : /linux/i.test(source) ? 'Linux'
            : source === 'ws' ? 'Server'
              : 'Unknown OS'
  const device = /mobile/i.test(source) ? 'Mobile'
    : /ipad|tablet/i.test(source) ? 'Tablet'
      : source === 'ws' ? 'Terminal'
        : 'Desktop'
  return {
    browser,
    version,
    os,
    device,
    label: `${browser}${version ? ` ${version}` : ''} · ${os} · ${device}`
  }
}

export function getReasonMeta (reason = '') {
  const direct = reasonMap[reason]
  if (direct) return { code: reason, ...direct }

  if (reason.startsWith('terminal_exec:')) {
    const command = reason.replace('terminal_exec:', '').trim()
    return {
      code: reason,
      label: 'Terminal Command',
      category: 'command',
      icon: 'mdi mdi-console-network-outline',
      description: command ? `Executed command: ${command}` : 'Executed terminal command.'
    }
  }
  if (reason.startsWith('terminal_blocked:')) {
    const command = reason.replace('terminal_blocked:', '').trim()
    return {
      code: reason,
      label: 'Blocked Command',
      category: 'command',
      icon: 'mdi mdi-block-helper',
      description: command ? `Blocked command: ${command}` : 'A command was blocked by policy.'
    }
  }
  if (reason.startsWith('terminal_highrisk:')) {
    const command = reason.replace('terminal_highrisk:', '').trim()
    return {
      code: reason,
      label: 'High-Risk Command',
      category: 'command',
      icon: 'mdi mdi-alert-octagon-outline',
      description: command ? `High-risk command executed: ${command}` : 'A high-risk command was executed.'
    }
  }

  return {
    code: reason,
    label: reason ? reason.replace(/[_:]/g, ' ').replace(/\b\w/g, char => char.toUpperCase()) : 'Authenticated',
    category: 'other',
    icon: 'mdi mdi-information-outline',
    description: reason || 'Authentication completed successfully.'
  }
}

export function extractIPs (text = '') {
  return Array.from(new Set((text.match(/\b(?:\d{1,3}\.){3}\d{1,3}\b/g) || [])))
}

export function extractPorts (text = '') {
  return Array.from(new Set((text.match(/:(\d{1,5})\b/g) || []).map(match => match.slice(1))))
}

export function normalizeAlertMessage (message = '') {
  return message
    .replace(/\d{4}-\d{2}-\d{2}t\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:z|[+-]\d{2}:?\d{2})/ig, '')
    .replace(/\s+/g, ' ')
    .trim()
}

export function summarizeAlert (alert) {
  const normalized = normalizeAlertMessage(alert.message || '')
  if (!normalized && alert.ip) {
    return `${alert.type || 'Alert'} involving ${alert.ip}`
  }
  return normalized || 'Alert event'
}

export function formatAlertMeta (alert) {
  const ips = extractIPs(alert.message || '')
  const ports = extractPorts(alert.message || '')
  const meta = []
  if (alert.source) meta.push(alert.source)
  if (ips.length) meta.push(`${ips.length} IP${ips.length > 1 ? 's' : ''}`)
  if (ports.length) meta.push(`${ports.length} port${ports.length > 1 ? 's' : ''}`)
  return meta
}

export function groupAlerts (alerts = []) {
  const sorted = [...alerts].sort((left, right) => (right.ts || 0) - (left.ts || 0))
  const groups = new Map()
  sorted.forEach(alert => {
    const bucket = Math.floor((alert.ts || 0) / 300)
    const key = [alert.severity, alert.type || 'unknown', alert.source || 'unknown', normalizeAlertMessage(alert.message || ''), bucket].join('::')
    const current = groups.get(key)
    if (current) {
      current.count += 1
      current.items.push(alert)
      current.read = current.read && !!alert.read
      current.latestTs = Math.max(current.latestTs, alert.ts || 0)
      return
    }
    groups.set(key, {
      id: `group-${alert.id}`,
      count: 1,
      items: [alert],
      read: !!alert.read,
      latestTs: alert.ts || 0,
      base: alert
    })
  })
  return Array.from(groups.values()).sort((left, right) => right.latestTs - left.latestTs)
}

export function matchesQuery (record, query, config = {}) {
  if (!query) return true
  const fields = config.fields || []
  const operators = config.operators || {}
  const tokens = query.toLowerCase().split(/\s+/).filter(Boolean)
  return tokens.every(token => {
    if (token.includes(':')) {
      const [rawKey, ...rawValueParts] = token.split(':')
      const rawValue = rawValueParts.join(':')
      const resolver = operators[rawKey]
      if (!resolver) return true
      return String(resolver(record) || '').toLowerCase().includes(rawValue)
    }
    return fields.some(field => String(record[field] || '').toLowerCase().includes(token))
  })
}

export function highlightMatch (text = '', query = '') {
  if (!query) return text
  const terms = query.split(/\s+/).filter(token => token && !token.includes(':'))
  if (!terms.length) return text
  const pattern = new RegExp(`(${terms.map(term => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`, 'ig')
  return text.replace(pattern, '<mark>$1</mark>')
}

export function loadSavedFilters (key) {
  try {
    return JSON.parse(localStorage.getItem(`sc_saved_filters_${key}`) || '[]')
  } catch {
    return []
  }
}

export function saveSavedFilters (key, filters) {
  safeSet(`sc_saved_filters_${key}`, JSON.stringify(filters))
}
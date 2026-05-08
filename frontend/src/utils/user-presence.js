const PRESENCE_KEY = 'sc_user_presence'
export const USER_PRESENCE_EVENT = 'sentinel:user-presence-change'

function safeParse(value, fallback) {
  try {
    return JSON.parse(value ?? '')
  } catch {
    return fallback
  }
}

function normalizeStatus(status) {
  return ['online', 'away', 'dnd', 'offline'].includes(status) ? status : 'online'
}

export function getPresenceStore() {
  return safeParse(localStorage.getItem(PRESENCE_KEY), {})
}

export function getUserPresence(username) {
  const store = getPresenceStore()
  const entry = store[username] || {}
  return {
    status: normalizeStatus(entry.status || 'online'),
    autoAwayMinutes: Number(entry.autoAwayMinutes || 15),
    avatarUrl: entry.avatarUrl || '',
    lastUpdatedAt: Number(entry.lastUpdatedAt || 0)
  }
}

export function setUserPresence(username, nextValue = {}) {
  if (!username) return getUserPresence('')
  const store = getPresenceStore()
  const nextEntry = {
    ...getUserPresence(username),
    ...nextValue,
    status: normalizeStatus(nextValue.status || getUserPresence(username).status),
    lastUpdatedAt: Date.now()
  }
  store[username] = nextEntry
  localStorage.setItem(PRESENCE_KEY, JSON.stringify(store))
  window.dispatchEvent(new CustomEvent(USER_PRESENCE_EVENT, {
    detail: { username, value: nextEntry }
  }))
  return nextEntry
}
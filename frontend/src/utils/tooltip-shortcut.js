const APPLE_PLATFORM_RE = /(mac|iphone|ipad|ipod)/i

function platformName() {
  if (typeof navigator === 'undefined') return ''
  return navigator.userAgentData?.platform || navigator.platform || navigator.userAgent || ''
}

export function isApplePlatform() {
  return APPLE_PLATFORM_RE.test(platformName())
}

function normalizeToken(token, apple) {
  const raw = String(token || '').trim()
  const key = raw.toLowerCase()

  if (!raw) return ''

  const appleMap = {
    cmd: '⌘',
    command: '⌘',
    meta: '⌘',
    '⌘': '⌘',
    option: '⌥',
    alt: '⌥',
    '⌥': '⌥',
    ctrl: '⌃',
    control: '⌃',
    '⌃': '⌃',
    shift: '⇧',
    '⇧': '⇧',
    enter: '⏎',
    return: '⏎',
    '⏎': '⏎',
    backspace: '⌫',
    delete: '⌫',
    '⌫': '⌫',
    esc: '⎋',
    escape: '⎋',
    '⎋': '⎋',
    space: '␣',
    spacebar: '␣',
    '␣': '␣',
    tab: '⇥',
    '⇥': '⇥',
    up: '↑',
    down: '↓',
    left: '←',
    right: '→'
  }

  const standardMap = {
    cmd: 'Cmd',
    command: 'Cmd',
    meta: 'Meta',
    '⌘': 'Cmd',
    option: 'Alt',
    alt: 'Alt',
    '⌥': 'Alt',
    ctrl: 'Ctrl',
    control: 'Ctrl',
    '⌃': 'Ctrl',
    shift: 'Shift',
    '⇧': 'Shift',
    enter: 'Enter',
    return: 'Enter',
    '⏎': 'Enter',
    backspace: 'Backspace',
    delete: 'Delete',
    '⌫': 'Backspace',
    esc: 'Esc',
    escape: 'Esc',
    '⎋': 'Esc',
    space: 'Space',
    spacebar: 'Space',
    '␣': 'Space',
    tab: 'Tab',
    '⇥': 'Tab',
    up: 'Up',
    down: 'Down',
    left: 'Left',
    right: 'Right'
  }

  const map = apple ? appleMap : standardMap
  return map[key] || raw
}

export function normalizeShortcut(shortcut, options = {}) {
  const apple = options.apple ?? isApplePlatform()
  if (!shortcut) return []

  const tokens = Array.isArray(shortcut)
    ? shortcut.flatMap(value => normalizeShortcut(value, { apple }))
    : String(shortcut)
        .split(/\s*\+\s*/)
        .map(part => part.trim())
        .filter(Boolean)

  return tokens
    .map(token => normalizeToken(token, apple))
    .filter(Boolean)
}
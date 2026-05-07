/**
 * Console logging guard for production
 */
const PROD = import.meta.env.MODE === 'production'

const original = {
  log: console.log,
  warn: console.warn,
  error: console.error,
  info: console.info,
  debug: console.debug
}

function noop() {}

if (PROD) {
  console.log = noop
  console.info = noop
  console.debug = noop
  console.warn = noop
  console.error = noop
}

export function restoreConsole() {
  Object.assign(console, original)
}

export default { restoreConsole }

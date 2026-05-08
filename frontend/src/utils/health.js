export function getHealthTone(score) {
  const val = Number(score) || 0
  if (val < 50) return 'error'
  if (val < 80) return 'warn'
  return 'ok'
}

export function getHealthColor(score) {
  return `var(--state-${getHealthTone(score)})`
}

export function getHealthStatusWord(score) {
  const val = Number(score) || 0
  if (val < 35) return 'Critical'
  if (val < 50) return 'Degraded'
  if (val < 80) return 'Healthy'
  return 'Optimal'
}

<template>
  <div
    class="container-logs-modal"
    :class="{ minimized: modal.minimized, maximized: modal.maximized, inactive: !focused }"
    :style="modalStyle"
    @mousedown.stop="bringToFront"
    role="dialog"
    aria-modal="false"
    :aria-labelledby="`container-logs-title-${modal.modalId}`"
  >
    <div class="clm-frame">
      <div class="clm-header" @pointerdown="startDrag" :class="{ dragging: dragging }">
        <div class="clm-header-left">
          <span class="status-dot" :class="statusClass"></span>
          <div>
            <div :id="`container-logs-title-${modal.modalId}`" class="clm-title">{{ container.name }} <span class="clm-subtitle">({{ container.id }})</span></div>
            <div class="clm-meta">{{ statusText }}</div>
          </div>
        </div>
        <div class="clm-controls">
          <button type="button" class="icon-btn" @click.stop="toggleMinimize" :title="modal.minimized ? 'Restore' : 'Minimize'">
            <i :class="`mdi ${modal.minimized ? 'mdi-arrow-up-bold' : 'mdi-window-minimize'}`"></i>
          </button>
          <button type="button" class="icon-btn" @click.stop="toggleMaximize" :title="modal.maximized ? 'Restore' : 'Maximize'">
            <i :class="`mdi ${modal.maximized ? 'mdi-window-restore' : 'mdi-window-maximize'}`"></i>
          </button>
          <button type="button" class="icon-btn text-danger" @click.stop="closeWindow" title="Close">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
      </div>

      <div v-if="!modal.minimized" class="clm-body">
        <div class="clm-toolbar">
          <div class="toolbar-group">
            <label>Tail</label>
            <select v-model.number="tailSize" @change="refreshLogs" class="form-select form-select-sm">
              <option :value="100">100</option>
              <option :value="500">500</option>
              <option :value="1000">1000</option>
              <option :value="5000">5000</option>
              <option :value="10000">10000</option>
            </select>
            <button class="btn btn-sm btn-outline-secondary" @click="setTailAll">All</button>
          </div>

          <div class="toolbar-group">
            <button class="btn btn-sm" :class="follow ? 'btn-sc-primary' : 'btn-outline-secondary'" @click="toggleFollow">
              <i :class="`mdi ${follow ? 'mdi-play-circle' : 'mdi-pause-circle'}`"></i>
              {{ follow ? 'LIVE' : 'Paused' }}
            </button>
            <span class="clm-status-badge" :class="connectionClass">{{ connectionLabel }}</span>
          </div>

          <div class="toolbar-group">
            <label class="form-label">Since</label>
            <input v-model="since" type="datetime-local" class="form-control form-control-sm" @change="refreshLogs" />
            <label class="form-label">Until</label>
            <input v-model="until" type="datetime-local" class="form-control form-control-sm" @change="refreshLogs" />
          </div>

          <div class="toolbar-group">
            <button class="btn btn-sm btn-outline-secondary" @click="jumpToBottom" :disabled="atBottom">Jump to bottom</button>
            <button class="btn btn-sm btn-outline-secondary" @click="clearView">Clear</button>
            <button class="btn btn-sm btn-outline-secondary" @click="downloadView">Download</button>
          </div>
        </div>

        <div class="clm-actions-row">
          <div class="clm-actions-left">
            <label class="form-label">Search</label>
            <input
              v-model="searchText"
              @input="debounceSearch"
              class="form-control form-control-sm"
              placeholder="Search / regex…"
              @keydown.ctrl.f.prevent="focusSearch"
              @keydown.meta.f.prevent="focusSearch"
            />
            <button class="btn btn-sm btn-outline-secondary" @click="nextMatch" :disabled="searchMatches===0">Next</button>
            <button class="btn btn-sm btn-outline-secondary" @click="prevMatch" :disabled="searchMatches===0">Prev</button>
            <span class="search-count">{{ searchMatchesLabel }}</span>
          </div>

          <div class="clm-actions-right">
            <div class="form-check form-switch">
              <input class="form-check-input" type="checkbox" v-model="searchRegex" :id="`clm-search-regex-${modal.modalId}`" />
              <label class="form-check-label" :for="`clm-search-regex-${modal.modalId}`">Regex</label>
            </div>
            <div class="form-check form-switch">
              <input class="form-check-input" type="checkbox" v-model="caseSensitive" :id="`clm-search-case-${modal.modalId}`" />
              <label class="form-check-label" :for="`clm-search-case-${modal.modalId}`">Case</label>
            </div>
            <div class="form-check form-switch">
              <input class="form-check-input" type="checkbox" v-model="wholeWord" :id="`clm-search-whole-${modal.modalId}`" />
              <label class="form-check-label" :for="`clm-search-whole-${modal.modalId}`">Whole</label>
            </div>
          </div>
        </div>

        <div class="clm-options-row">
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="prettyPrint" :id="`clm-pretty-${modal.modalId}`" />
            <label class="form-check-label" :for="`clm-pretty-${modal.modalId}`">Pretty print</label>
          </div>
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="wrapLines" :id="`clm-wrap-${modal.modalId}`" />
            <label class="form-check-label" :for="`clm-wrap-${modal.modalId}`">Word wrap</label>
          </div>
          <div class="form-check form-switch">
            <input class="form-check-input" type="checkbox" v-model="showTimestamps" :id="`clm-ts-${modal.modalId}`" />
            <label class="form-check-label" :for="`clm-ts-${modal.modalId}`">Timestamps</label>
          </div>
        </div>

        <div class="clm-banner" v-if="banner">{{ banner }}</div>

        <div class="clm-viewer" ref="viewer" @scroll="onScroll">
          <div :style="{ height: topSpacer + 'px' }"></div>
          <div v-for="line in visibleLines" :key="line.key" class="clm-line" :class="{ stderr: line.stream === 'stderr', highlight: line.isMatch }">
            <div class="clm-line-number">{{ line.lineNumber }}</div>
            <div class="clm-line-content" :class="{ wrap: wrapLines }" v-html="renderLine(line)"></div>
          </div>
          <div :style="{ height: bottomSpacer + 'px' }"></div>
        </div>
      </div>

      <div v-if="modal.minimized" class="clm-minimized-bar" @click.stop="toggleMinimize">
        <i class="mdi mdi-text-box-outline"></i> {{ container.name }} logs (minimized)
      </div>

      <div class="resize-handle bottom-right" @pointerdown.stop.prevent="startResize($event, 'bottom-right')"></div>
      <div class="resize-handle right" @pointerdown.stop.prevent="startResize($event, 'right')"></div>
      <div class="resize-handle bottom" @pointerdown.stop.prevent="startResize($event, 'bottom')"></div>
    </div>
  </div>
</template>

<script>
import api from '@/services/api'

const LINE_HEIGHT = 20
const MAX_BUFFER_LINES = 50000
const SEARCH_DEBOUNCE_MS = 150

export default {
  name: 'ContainerLogsModal',
  props: {
    modal: { type: Object, required: true },
    container: { type: Object, required: true }
  },
  data() {
    return {
      lines: [],
      socket: null,
      connectionStatus: 'connecting',
      reconnectAttempts: 0,
      tailSize: 500,
      follow: true,
      since: '',
      until: '',
      showTimestamps: true,
      prettyPrint: true,
      wrapLines: true,
      searchText: '',
      searchRegex: false,
      caseSensitive: false,
      wholeWord: false,
      searchMatches: 0,
      currentMatch: 0,
      searchError: '',
      lineKey: 0,
      atBottom: true,
      pendingLines: 0,
      banner: '',
      dragging: false,
      resizing: false,
      resizeDir: '',
      dragStartX: 0,
      dragStartY: 0,
      startLeft: 0,
      startTop: 0,
      startWidth: 0,
      startHeight: 0,
      focused: true,
      lastSearchQuery: '',
      searchTimer: null
    }
  },
  computed: {
    modalStyle() {
      return {
        left: this.modal.left + 'px',
        top: this.modal.top + 'px',
        width: this.modal.width + 'px',
        height: this.modal.height + 'px',
        zIndex: this.modal.zIndex
      }
    },
    statusClass() {
      if (this.connectionStatus === 'connected') return 'online'
      if (this.connectionStatus === 'reconnecting') return 'warn'
      return 'offline'
    },
    statusText() {
      if (this.connectionStatus === 'connected') return 'Streaming'
      if (this.connectionStatus === 'reconnecting') return 'Reconnecting'
      return 'Disconnected'
    },
    connectionLabel() {
      if (this.connectionStatus === 'connected') return 'LIVE'
      if (this.connectionStatus === 'reconnecting') return 'Reconnecting'
      return 'Disconnected'
    },
    visibleCount() {
      const height = this.$refs.viewer ? this.$refs.viewer.clientHeight : 320
      return Math.ceil(height / LINE_HEIGHT) + 12
    },
    scrollTop() {
      return this.$refs.viewer ? this.$refs.viewer.scrollTop : 0
    },
    visibleStart() {
      return Math.max(0, Math.floor(this.scrollTop / LINE_HEIGHT) - 6)
    },
    visibleLines() {
      const slice = this.lines.slice(this.visibleStart, this.visibleStart + this.visibleCount)
      return slice
    },
    topSpacer() {
      return this.visibleStart * LINE_HEIGHT
    },
    bottomSpacer() {
      const remaining = this.lines.length - (this.visibleStart + this.visibleCount)
      return remaining > 0 ? remaining * LINE_HEIGHT : 0
    },
    searchMatchesLabel() {
      return this.searchError ? `Invalid regex` : `${this.currentMatch}/${this.searchMatches}`
    }
  },
  mounted() {
    this.loadSnapshot()
    this.openStream()
    window.addEventListener('resize', this.clampPosition)
    window.addEventListener('keydown', this.handleKeydown)
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.clampPosition)
    window.removeEventListener('keydown', this.handleKeydown)
    this.closeStream()
  },
  methods: {
    bringToFront() {
      this.$emit('focus', this.modal.modalId)
      this.focused = true
    },
    closeWindow() {
      if (this.follow && this.lines.length > 0) {
        if (!window.confirm('Stop streaming and close this log window?')) {
          return
        }
      }
      this.closeStream()
      this.$emit('close', this.modal.modalId)
    },
    toggleMinimize() {
      this.$emit('update', { ...this.modal, minimized: !this.modal.minimized })
    },
    toggleMaximize() {
      this.$emit('update', { ...this.modal, maximized: !this.modal.maximized })
    },
    async loadSnapshot() {
      try {
        const params = { tail: this.tailSize, stdout: 1, stderr: 1, timestamps: 1 }
        if (this.since) params.since = this.toIso(this.since)
        if (this.until) params.until = this.toIso(this.until)
        const { data } = await api.getContainerLogs(this.container.id, params)
        const rawLines = Array.isArray(data.lines) ? data.lines : []
        this.lines = rawLines.map((item, idx) => this.normalizeLine(item, idx + 1))
        this.searchMatches = 0
        this.currentMatch = 0
        this.pendingLines = 0
        this.banner = ''
        this.scrollToBottom()
      } catch (err) {
        this.banner = `Failed to load logs: ${err.response?.data?.error || err.message || 'unknown'}`
      }
    },
    normalizeLine(raw, index) {
      this.lineKey += 1
      return {
        key: `${Date.now()}-${this.lineKey}`,
        stream: raw.stream || 'stdout',
        timestamp: raw.timestamp || '',
        text: raw.text || raw.line || '',
        lineNumber: index,
        isMatch: false
      }
    },
    async openStream() {
      if (!this.follow) return
      this.closeStream()
      this.connectionStatus = 'connecting'
      const wsUrl = api.getContainerLogsWsUrl(this.container.id, {
        stdout: 1,
        stderr: 1,
        timestamps: 1,
        tail: this.tailSize
      })
      this.socket = new WebSocket(wsUrl)
      this.socket.onopen = () => {
        this.connectionStatus = 'connected'
        this.reconnectAttempts = 0
      }
      this.socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'line') {
            this.appendLine({ stream: msg.stream, timestamp: msg.timestamp, text: msg.text })
          } else if (msg.type === 'status') {
            this.connectionStatus = msg.status || 'connected'
          } else if (msg.type === 'error') {
            this.banner = msg.error || 'Log stream error'
          }
        } catch (e) {
          console.warn('Malformed log stream payload', e)
        }
      }
      this.socket.onclose = () => {
        this.connectionStatus = 'disconnected'
        if (this.follow) {
          this.reconnectAttempts += 1
          const delay = Math.min(30000, 1000 * Math.pow(2, Math.min(this.reconnectAttempts, 5)))
          setTimeout(() => this.openStream(), delay)
        }
      }
      this.socket.onerror = () => {
        this.connectionStatus = 'disconnected'
      }
    },
    closeStream() {
      if (this.socket) {
        this.socket.close()
        this.socket = null
      }
    },
    appendLine(raw) {
      const newLine = this.normalizeLine(raw, this.lines.length + 1)
      this.lines.push(newLine)
      if (this.lines.length > MAX_BUFFER_LINES) {
        this.lines.splice(0, this.lines.length-MAX_BUFFER_LINES)
        this.banner = 'Older logs trimmed; increase tail size or download to preserve more lines.'
      }
      this.applySearchToLine(newLine)
      if (this.atBottom) {
        this.$nextTick(() => this.scrollToBottom())
      } else {
        this.pendingLines += 1
      }
    },
    applySearchToLine(line) {
      if (!this.searchText) {
        line.isMatch = false
        return
      }
      const query = this.searchText
      const flags = this.caseSensitive ? '' : 'i'
      let pattern = query
      if (!this.searchRegex) {
        pattern = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      }
      if (this.wholeWord) {
        pattern = `\\b${pattern}\\b`
      }
      try {
        const re = new RegExp(pattern, flags)
        line.isMatch = re.test(line.text)
      } catch (err) {
        line.isMatch = false
      }
    },
    refreshLogs() {
      this.loadSnapshot()
      if (this.follow) {
        this.openStream()
      }
    },
    setTailAll() {
      this.tailSize = 0
      this.refreshLogs()
    },
    scrollToBottom() {
      const el = this.$refs.viewer
      if (el) {
        el.scrollTop = el.scrollHeight
        this.atBottom = true
        this.pendingLines = 0
      }
    },
    onScroll() {
      const el = this.$refs.viewer
      if (!el) return
      const distance = el.scrollHeight - (el.scrollTop + el.clientHeight)
      this.atBottom = distance < 10
      if (this.atBottom) {
        this.pendingLines = 0
      }
    },
    renderLine(line) {
      const raw = this.showTimestamps || !line.timestamp ? line.text : this.stripTimestamp(line.text)
      const escaped = this.escapeHtml(raw)
      if (!this.searchText) {
        return this.prettyPrint ? this.applyPrettyPrint(escaped) : escaped
      }
      const highlight = this.buildHighlight(escaped)
      return this.prettyPrint ? this.applyPrettyPrint(highlight) : highlight
    },
    buildHighlight(text) {
      const query = this.searchText
      const flags = this.caseSensitive ? 'g' : 'gi'
      let pattern = this.searchRegex ? query : query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      if (this.wholeWord) {
        pattern = `\\b${pattern}\\b`
      }
      try {
        const re = new RegExp(pattern, flags)
        return text.replace(re, (match) => `<span class="clm-match">${match}</span>`)
      } catch (err) {
        this.searchError = err.message
        return text
      }
    },
    escapeHtml(value) {
      return value.replace(/[&<>"']/g, (char) => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
      })[char])
    },
    stripTimestamp(text) {
      const parts = text.split(' ')
      if (parts.length > 1 && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(parts[0])) {
        return text.slice(parts[0].length + 1)
      }
      return text
    },
    toggleFollow() {
      this.follow = !this.follow
      if (this.follow) {
        this.openStream()
      } else {
        this.closeStream()
      }
    },
    applyPrettyPrint(text) {
      if (!this.prettyPrint) return text
      try {
        const json = JSON.parse(this.stripAnsi(text))
        return `<pre>${this.escapeHtml(JSON.stringify(json, null, 2))}</pre>`
      } catch {
        return text
      }
    },
    stripAnsi(input) {
      return input.replace(/\u001b\[[0-9;]*m/g, '')
    },
    focusSearch() {
      const input = this.$el.querySelector('input[placeholder="Search / regex…"]')
      if (input) {
        input.focus()
      }
    },
    debounceSearch() {
      clearTimeout(this.searchTimer)
      this.searchTimer = setTimeout(() => {
        this.applySearch()
      }, SEARCH_DEBOUNCE_MS)
    },
    applySearch() {
      this.searchError = ''
      this.searchMatches = 0
      this.currentMatch = 0
      const query = this.searchText
      if (!query) {
        return
      }
      const flags = this.caseSensitive ? 'g' : 'gi'
      let pattern = this.searchRegex ? query : query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      if (this.wholeWord) {
        pattern = `\\b${pattern}\\b`
      }
      try {
        const re = new RegExp(pattern, flags)
        this.lines.forEach((line) => {
          line.isMatch = re.test(line.text)
          if (line.isMatch) {
            this.searchMatches += 1
          }
        })
        this.currentMatch = this.searchMatches > 0 ? 1 : 0
      } catch (err) {
        this.searchError = err.message
      }
    },
    nextMatch() {
      if (this.searchMatches === 0) return
      this.currentMatch = this.currentMatch < this.searchMatches ? this.currentMatch + 1 : 1
    },
    prevMatch() {
      if (this.searchMatches === 0) return
      this.currentMatch = this.currentMatch > 1 ? this.currentMatch - 1 : this.searchMatches
    },
    downloadView() {
      const content = this.lines.map((line) => line.text).join('\n')
      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${this.container.name}-logs.txt`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    },
    clampPosition() {
      const margin = 16
      const width = window.innerWidth
      const height = window.innerHeight
      let left = Math.min(Math.max(this.modal.left, margin), width - this.modal.width - margin)
      let top = Math.min(Math.max(this.modal.top, margin), height - this.modal.height - margin)
      if (left !== this.modal.left || top !== this.modal.top) {
        this.$emit('update', { ...this.modal, left, top })
      }
    },
    startDrag(event) {
      if (this.modal.maximized || this.modal.minimized) return
      this.dragging = true
      this.dragStartX = event.clientX
      this.dragStartY = event.clientY
      this.startLeft = this.modal.left
      this.startTop = this.modal.top
      window.addEventListener('pointermove', this.doDrag)
      window.addEventListener('pointerup', this.stopDrag)
    },
    doDrag(event) {
      if (!this.dragging) return
      const left = this.startLeft + event.clientX - this.dragStartX
      const top = this.startTop + event.clientY - this.dragStartY
      this.$emit('update', { ...this.modal, left: Math.max(12, left), top: Math.max(12, top) })
    },
    stopDrag() {
      this.dragging = false
      window.removeEventListener('pointermove', this.doDrag)
      window.removeEventListener('pointerup', this.stopDrag)
    },
    startResize(event, direction) {
      this.resizing = true
      this.resizeDir = direction
      this.dragStartX = event.clientX
      this.dragStartY = event.clientY
      this.startWidth = this.modal.width
      this.startHeight = this.modal.height
      this.startLeft = this.modal.left
      this.startTop = this.modal.top
      window.addEventListener('pointermove', this.doResize)
      window.addEventListener('pointerup', this.stopResize)
    },
    doResize(event) {
      if (!this.resizing) return
      let width = this.startWidth
      let height = this.startHeight
      let left = this.startLeft
      let top = this.startTop
      const dx = event.clientX - this.dragStartX
      const dy = event.clientY - this.dragStartY
      if (this.resizeDir.includes('right')) {
        width = Math.max(420, this.startWidth + dx)
      }
      if (this.resizeDir.includes('bottom')) {
        height = Math.max(260, this.startHeight + dy)
      }
      this.$emit('update', { ...this.modal, width, height, left, top })
    },
    stopResize() {
      this.resizing = false
      window.removeEventListener('pointermove', this.doResize)
      window.removeEventListener('pointerup', this.stopResize)
    },
    focusSearch() {
      const input = this.$el.querySelector('input[placeholder="Search / regex…"]')
      if (input) input.focus()
    },
    handleKeydown(event) {
      if (event.key === 'Escape') {
        this.closeWindow()
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'f') {
        event.preventDefault()
        this.focusSearch()
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'l') {
        event.preventDefault()
        this.clearView()
      }
      if ((event.altKey && event.key === 'ArrowDown') || (event.altKey && event.key === 'ArrowUp') || (event.altKey && event.key === 'ArrowLeft') || (event.altKey && event.key === 'ArrowRight')) {
        event.preventDefault()
        const delta = 16
        let left = this.modal.left
        let top = this.modal.top
        if (event.key === 'ArrowDown') top += delta
        if (event.key === 'ArrowUp') top -= delta
        if (event.key === 'ArrowLeft') left -= delta
        if (event.key === 'ArrowRight') left += delta
        this.$emit('update', { ...this.modal, left: Math.max(12, left), top: Math.max(12, top) })
      }
    },
    clearView() {
      this.lines = []
      this.banner = 'Log view cleared. Streaming continues in the background.'
    },
    jumpToBottom() {
      this.scrollToBottom()
    },
    toIso(localDatetime) {
      if (!localDatetime) return ''
      const d = new Date(localDatetime)
      return d.toISOString()
    }
  }
}
</script>

<style scoped>
.container-logs-modal {
  position: fixed;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  pointer-events: auto;
}
.container-logs-modal.inactive {
  opacity: 0.95;
}
.clm-frame {
  position: absolute;
  background: var(--sc-bg-card, #0c1724);
  border: 1px solid var(--sc-border, #1f2e48);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 420px;
  min-height: 260px;
  box-shadow: 0 16px 48px rgba(0,0,0,0.35);
}
.clm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1rem;
  cursor: grab;
  background: linear-gradient(180deg, rgba(15,23,42,0.98), rgba(8,13,24,0.95));
  border-bottom: 1px solid rgba(255,255,255,0.05);
}
.clm-header.dragging {
  cursor: grabbing;
}
.clm-header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.clm-title {
  font-size: 0.92rem;
  font-weight: 700;
}
.clm-subtitle {
  font-size: 0.78rem;
  color: var(--sc-text-muted, #8aa4c8);
}
.clm-meta {
  font-size: 0.72rem;
  color: var(--sc-text-muted, #8aa4c8);
}
.clm-controls {
  display: flex;
  gap: 0.35rem;
}
.icon-btn {
  border: none;
  background: transparent;
  color: var(--sc-text, #e2ecff);
  padding: 0.45rem;
  cursor: pointer;
}
.clm-body {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
  height: calc(100% - 56px);
}
.clm-toolbar,
.clm-actions-row,
.clm-options-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
}
.toolbar-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.clm-status-badge {
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  font-size: 0.72rem;
  background: rgba(15,23,42,0.85);
}
.clm-status-badge.online { background: rgba(34,214,124,0.15); color: #22d67c; }
.clm-status-badge.warn { background: rgba(245,166,35,0.15); color: #f5a623; }
.clm-status-badge.offline { background: rgba(240,64,64,0.12); color: #f04040; }
.clm-banner {
  font-size: 0.78rem;
  color: #f5a623;
  background: rgba(245,166,35,0.08);
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
}
.clm-viewer {
  flex: 1;
  overflow: auto;
  background: var(--sc-bg-secondary, #06101d);
  border: 1px solid rgba(255,255,255,0.04);
  border-radius: 10px;
  padding: 0.45rem;
  font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
}
.clm-line {
  display: grid;
  grid-template-columns: 48px 1fr;
  gap: 0.75rem;
  padding: 0.15rem 0.5rem;
  min-height: 20px;
  align-items: flex-start;
  font-size: 0.74rem;
  line-height: 1.4;
  white-space: pre;
}
.clm-line.stderr {
  background: rgba(255, 80, 80, 0.06);
}
.clm-line-number {
  color: var(--sc-text-muted, #5a7499);
  user-select: none;
  text-align: right;
  font-size: 0.72rem;
}
.clm-line-content {
  width: 100%;
  white-space: pre;
  word-break: break-word;
}
.clm-line-content.wrap {
  white-space: pre-wrap;
}
.clm-match {
  background: rgba(92, 140, 255, 0.25);
  color: #dbeafe;
  border-radius: 3px;
}
.clm-minimized-bar {
  padding: 0.75rem 1rem;
  background: rgba(15,23,42,0.95);
  border-top: 1px solid rgba(255,255,255,0.08);
  color: var(--sc-text, #e2ecff);
  cursor: pointer;
}
.resize-handle {
  position: absolute;
  background: transparent;
}
.bottom-right {
  width: 18px;
  height: 18px;
  right: 0;
  bottom: 0;
  cursor: se-resize;
}
.right {
  width: 10px;
  top: 0;
  right: 0;
  bottom: 0;
  cursor: e-resize;
}
.bottom {
  height: 10px;
  left: 0;
  right: 0;
  bottom: 0;
  cursor: s-resize;
}
@media (max-width: 900px) {
  .container-logs-modal {
    left: 0 !important;
    top: 0 !important;
  }
  .clm-frame {
    width: 100vw !important;
    height: 100vh !important;
    border-radius: 0 !important;
    min-width: 100%;
    min-height: 100%;
  }
  .resize-handle { display: none; }
}
</style>

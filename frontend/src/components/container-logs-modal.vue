<template>
  <div
    class="container-logs-modal"
    v-show="!modal.minimized"
    :class="{ minimized: modal.minimized, maximized: modal.maximized, inactive: !focused }"
    :style="modalStyle"
    @pointerdown.stop.prevent="bringToFront"
    role="dialog"
    aria-modal="false"
    :aria-labelledby="`container-logs-title-${modal.modalId}`"
  >
    <div class="clm-frame" @pointerdown.stop="bringToFront">
      <div class="clm-header" @pointerdown.stop.prevent="startDrag" :class="{ dragging: dragging || resizing }">
        <div class="clm-header-left">
          <span class="status-dot" :class="statusClass"></span>
          <div class="clm-title-group">
            <div :id="`container-logs-title-${modal.modalId}`" class="clm-title">{{ container.name }}</div>
            <div class="clm-status-text">{{ statusText }}</div>
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

      <div class="clm-body">
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

        <div class="clm-toolbar" :class="{ expanded: toolbarExpanded, collapsed: !toolbarExpanded }">
          <div class="toolbar-row">
            <div class="toolbar-group toolbar-main">
              <label class="toolbar-label" for="clm-tail-select">Tail</label>
              <select id="clm-tail-select" v-model.number="tailSize" @change="refreshLogs" class="toolbar-select">
                <option :value="100">100</option>
                <option :value="500">500</option>
                <option :value="1000">1000</option>
                <option :value="5000">5000</option>
                <option :value="10000">10000</option>
              </select>
              <button type="button" class="toolbar-button" @click="setTailAll" title="Tail all logs">All</button>
            </div>

            <div class="toolbar-group toolbar-main">
              <button
                type="button"
                class="toolbar-icon-btn"
                :class="{ active: follow }"
                @click="toggleFollow"
                :title="follow ? 'Pause live streaming' : 'Resume live streaming'"
              >
                <i :class="`mdi ${follow ? 'mdi-play-circle' : 'mdi-pause-circle'}`"></i>
              </button>
              <span class="clm-status-badge" :class="connectionClass">{{ connectionLabel }}</span>
            </div>

            <div class="toolbar-group toolbar-search-group">
              <button type="button" class="toolbar-icon-btn" @click.stop.prevent="toggleSearch" title="Search logs">
                <i class="mdi mdi-magnify"></i>
              </button>
              <div v-if="searchExpanded" class="search-popover">
                <input
                  ref="searchInput"
                  v-model="searchText"
                  @input="debounceSearch"
                  class="toolbar-search-input"
                  placeholder="Search / regex…"
                  @keydown.enter.prevent="applySearch"
                  @keydown.esc.prevent="closeSearch"
                />
                <div class="search-controls">
                  <button type="button" class="toolbar-icon-btn" @click="prevMatch" :disabled="searchMatches===0" title="Previous match">
                    <i class="mdi mdi-chevron-up"></i>
                  </button>
                  <button type="button" class="toolbar-icon-btn" @click="nextMatch" :disabled="searchMatches===0" title="Next match">
                    <i class="mdi mdi-chevron-down"></i>
                  </button>
                  <span class="search-count">{{ searchMatchesLabel }}</span>
                </div>
              </div>
            </div>

            <button type="button" class="toolbar-icon-btn toolbar-more-btn" @click="toggleToolbarExpanded" :title="toolbarExpanded ? 'Collapse toolbar' : 'Expand toolbar'">
              <i class="mdi" :class="toolbarExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"></i>
            </button>
          </div>

          <div v-if="toolbarExpanded" class="toolbar-row toolbar-expanded-row">
            <div class="toolbar-group">
              <label class="toolbar-label" for="clm-since-input">Since</label>
              <input id="clm-since-input" v-model="since" type="datetime-local" class="toolbar-input" @change="refreshLogs" />
            </div>
            <div class="toolbar-group">
              <label class="toolbar-label" for="clm-until-input">Until</label>
              <input id="clm-until-input" v-model="until" type="datetime-local" class="toolbar-input" @change="refreshLogs" />
            </div>
            <div class="toolbar-group toolbar-toggle-group">
              <button type="button" class="toolbar-icon-btn" :class="{ active: searchRegex }" @click="searchRegex = !searchRegex" title="Regex search">
                <i class="mdi mdi-code-tags"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" :class="{ active: caseSensitive }" @click="caseSensitive = !caseSensitive" title="Case sensitive">
                <i class="mdi mdi-format-letter-case"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" :class="{ active: wholeWord }" @click="wholeWord = !wholeWord" title="Whole word match">
                <i class="mdi mdi-target"></i>
              </button>
            </div>
            <div class="toolbar-group toolbar-toggle-group">
              <button type="button" class="toolbar-icon-btn" :class="{ active: showStdout }" @click="showStdout = !showStdout" title="Show stdout">
                <i class="mdi mdi-console"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" :class="{ active: showStderr }" @click="showStderr = !showStderr" title="Show stderr">
                <i class="mdi mdi-console-network"></i>
              </button>
            </div>
            <div class="toolbar-group toolbar-toggle-group">
              <button type="button" class="toolbar-icon-btn" :class="{ active: prettyPrint }" @click="prettyPrint = !prettyPrint" title="Pretty print">
                <i class="mdi mdi-format-align-left"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" :class="{ active: wrapLines }" @click="wrapLines = !wrapLines" title="Word wrap">
                <i class="mdi mdi-arrow-split-vertical"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" :class="{ active: showTimestamps }" @click="showTimestamps = !showTimestamps" title="Show timestamps">
                <i class="mdi mdi-timer-outline"></i>
              </button>
            </div>
            <div class="toolbar-group toolbar-actions-group">
              <button type="button" class="toolbar-icon-btn" @click="clearView" title="Clear view">
                <i class="mdi mdi-broom"></i>
              </button>
              <button type="button" class="toolbar-icon-btn" @click="downloadView" title="Download logs">
                <i class="mdi mdi-download"></i>
              </button>
            </div>
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

      <div class="resize-handle top-left" @pointerdown.stop.prevent="startResize($event, 'top-left')"></div>
      <div class="resize-handle top" @pointerdown.stop.prevent="startResize($event, 'top')"></div>
      <div class="resize-handle top-right" @pointerdown.stop.prevent="startResize($event, 'top-right')"></div>
      <div class="resize-handle right" @pointerdown.stop.prevent="startResize($event, 'right')"></div>
      <div class="resize-handle bottom-right" @pointerdown.stop.prevent="startResize($event, 'bottom-right')"></div>
      <div class="resize-handle bottom" @pointerdown.stop.prevent="startResize($event, 'bottom')"></div>
      <div class="resize-handle bottom-left" @pointerdown.stop.prevent="startResize($event, 'bottom-left')"></div>
      <div class="resize-handle left" @pointerdown.stop.prevent="startResize($event, 'left')"></div>
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
      resizeFrame: null,
      pendingResize: null,
      toolbarExpanded: false,
      searchExpanded: false,
      showStdout: true,
      showStderr: true,
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
    filteredLines() {
      return this.lines.filter((line) => {
        if (line.stream === 'stdout' && !this.showStdout) return false
        if (line.stream === 'stderr' && !this.showStderr) return false
        return true
      })
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
      return this.filteredLines.slice(this.visibleStart, this.visibleStart + this.visibleCount)
    },
    topSpacer() {
      return this.visibleStart * LINE_HEIGHT
    },
    bottomSpacer() {
      const remaining = this.filteredLines.length - (this.visibleStart + this.visibleCount)
      return remaining > 0 ? remaining * LINE_HEIGHT : 0
    },
    searchMatchesLabel() {
      return this.searchError ? `Invalid regex` : `${this.currentMatch}/${this.searchMatches}`
    }
  },
  mounted() {
    this.toolbarExpanded = localStorage.getItem('clmToolbarExpanded') === 'true'
    this.loadSnapshot()
    this.openStream()
    this.clampPosition()
    window.addEventListener('resize', this.clampPosition)
    window.addEventListener('keydown', this.handleKeydown)
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.clampPosition)
    window.removeEventListener('keydown', this.handleKeydown)
    this.closeStream()
    if (this.resizeFrame) {
      cancelAnimationFrame(this.resizeFrame)
      this.resizeFrame = null
    }
    this.resetBodyDragState()
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
      const margin = 16
      if (!this.modal.maximized) {
        const prevRect = {
          left: this.modal.left,
          top: this.modal.top,
          width: this.modal.width,
          height: this.modal.height
        }
        const width = Math.max(360, window.innerWidth - margin * 2)
        const height = Math.max(240, window.innerHeight - margin * 2)
        this.$emit('update', {
          ...this.modal,
          maximized: true,
          prevRect,
          left: margin,
          top: margin,
          width,
          height
        })
        return
      }
      const rect = this.modal.prevRect || {
        left: this.modal.left,
        top: this.modal.top,
        width: this.modal.width,
        height: this.modal.height
      }
      this.$emit('update', {
        ...this.modal,
        maximized: false,
        left: rect.left,
        top: rect.top,
        width: rect.width,
        height: rect.height
      })
    },
    toggleToolbarExpanded() {
      this.toolbarExpanded = !this.toolbarExpanded
      localStorage.setItem('clmToolbarExpanded', String(this.toolbarExpanded))
    },
    toggleSearch() {
      this.searchExpanded = !this.searchExpanded
      if (this.searchExpanded) {
        this.$nextTick(() => this.$refs.searchInput?.focus())
      }
    },
    closeSearch() {
      this.searchExpanded = false
    },
    clampPosition() {
      const margin = 16
      const maxWidth = Math.max(360, window.innerWidth - margin * 2)
      const maxHeight = Math.max(240, window.innerHeight - margin * 2)
      let width = Math.min(this.modal.width, maxWidth)
      let height = Math.min(this.modal.height, maxHeight)
      let left = Math.min(Math.max(this.modal.left, margin), window.innerWidth - width - margin)
      let top = Math.min(Math.max(this.modal.top, margin), window.innerHeight - height - margin)
      if (Math.abs(left - margin) < 8) left = margin
      if (Math.abs(top - margin) < 8) top = margin
      if (Math.abs((window.innerWidth - width - margin) - left) < 8) left = window.innerWidth - width - margin
      if (Math.abs((window.innerHeight - height - margin) - top) < 8) top = window.innerHeight - height - margin
      if (width !== this.modal.width || height !== this.modal.height || left !== this.modal.left || top !== this.modal.top) {
        this.$emit('update', { ...this.modal, width, height, left, top })
      }
    },
    resetBodyDragState() {
      document.body.style.userSelect = ''
      document.body.style.cursor = ''
    },
    startDrag(event) {
      if (this.modal.maximized || this.modal.minimized) return
      this.dragging = true
      this.dragStartX = event.clientX
      this.dragStartY = event.clientY
      this.startLeft = this.modal.left
      this.startTop = this.modal.top
      document.body.style.userSelect = 'none'
      document.body.style.cursor = 'grabbing'
      window.addEventListener('pointermove', this.doDrag)
      window.addEventListener('pointerup', this.stopDrag)
    },
    doDrag(event) {
      if (!this.dragging) return
      const left = this.startLeft + event.clientX - this.dragStartX
      const top = this.startTop + event.clientY - this.dragStartY
      this.$emit('update', {
        ...this.modal,
        left: Math.max(16, left),
        top: Math.max(16, top)
      })
    },
    stopDrag() {
      this.dragging = false
      window.removeEventListener('pointermove', this.doDrag)
      window.removeEventListener('pointerup', this.stopDrag)
      this.resetBodyDragState()
    },
    startResize(event, direction) {
      if (this.modal.maximized || this.modal.minimized) return
      this.resizing = true
      this.resizeDir = direction
      this.dragStartX = event.clientX
      this.dragStartY = event.clientY
      this.startWidth = this.modal.width
      this.startHeight = this.modal.height
      this.startLeft = this.modal.left
      this.startTop = this.modal.top
      this.pendingResize = { dx: 0, dy: 0 }
      event.target.setPointerCapture?.(event.pointerId)
      document.body.style.userSelect = 'none'
      document.body.style.cursor = this.resizeCursor(direction)
      window.addEventListener('pointermove', this.doResize)
      window.addEventListener('pointerup', this.stopResize)
    },
    doResize(event) {
      if (!this.resizing) return
      this.pendingResize = {
        dx: event.clientX - this.dragStartX,
        dy: event.clientY - this.dragStartY
      }
      if (!this.resizeFrame) {
        this.resizeFrame = requestAnimationFrame(() => {
          this.resizeFrame = null
          this.applyResize()
        })
      }
    },
    applyResize() {
      if (!this.pendingResize) return
      let width = this.startWidth
      let height = this.startHeight
      let left = this.startLeft
      let top = this.startTop
      const dx = this.pendingResize.dx
      const dy = this.pendingResize.dy
      const minWidth = 360
      const minHeight = 240
      const margin = 16
      if (this.resizeDir.includes('right')) {
        width = Math.max(minWidth, this.startWidth + dx)
      }
      if (this.resizeDir.includes('left')) {
        width = Math.max(minWidth, this.startWidth - dx)
        left = this.startLeft + dx
      }
      if (this.resizeDir.includes('bottom')) {
        height = Math.max(minHeight, this.startHeight + dy)
      }
      if (this.resizeDir.includes('top')) {
        height = Math.max(minHeight, this.startHeight - dy)
        top = this.startTop + dy
      }
      width = Math.min(width, window.innerWidth - margin * 2)
      height = Math.min(height, window.innerHeight - margin * 2)
      left = Math.min(Math.max(left, margin), window.innerWidth - width - margin)
      top = Math.min(Math.max(top, margin), window.innerHeight - height - margin)
      if (Math.abs(left - margin) < 8) left = margin
      if (Math.abs(top - margin) < 8) top = margin
      if (Math.abs((window.innerWidth - width - margin) - left) < 8) left = window.innerWidth - width - margin
      if (Math.abs((window.innerHeight - height - margin) - top) < 8) top = window.innerHeight - height - margin
      this.$emit('update', { ...this.modal, width, height, left, top })
    },
    stopResize(event) {
      this.resizing = false
      if (event?.currentTarget) {
        event.currentTarget.releasePointerCapture?.(event.pointerId)
      }
      window.removeEventListener('pointermove', this.doResize)
      window.removeEventListener('pointerup', this.stopResize)
      if (this.resizeFrame) {
        cancelAnimationFrame(this.resizeFrame)
        this.resizeFrame = null
      }
      this.pendingResize = null
      this.resetBodyDragState()
    },
    resizeCursor(direction) {
      if (direction.includes('left') && direction.includes('top')) return 'nwse-resize'
      if (direction.includes('right') && direction.includes('bottom')) return 'nwse-resize'
      if (direction.includes('left') && direction.includes('bottom')) return 'nesw-resize'
      if (direction.includes('right') && direction.includes('top')) return 'nesw-resize'
      if (direction === 'top' || direction === 'bottom') return 'ns-resize'
      return 'ew-resize'
    },
    focusSearch() {
      const input = this.$refs.searchInput || this.$el.querySelector('.toolbar-search-input')
      if (input) input.focus()
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
  pointer-events: auto;
  min-width: 360px;
  min-height: 240px;
  max-width: calc(100vw - 32px);
  max-height: calc(100vh - 32px);
}
.container-logs-modal.inactive .clm-frame {
  box-shadow: 0 8px 20px rgba(0,0,0,0.18);
}
.clm-frame {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--sc-bg-card, #0d1626);
  border: 1px solid var(--sc-border, #1f2e48);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  transition: transform 150ms ease-out, opacity 150ms ease-out, box-shadow 150ms ease-out;
}
.clm-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 34px;
  height: 34px;
  padding: 0 8px;
  gap: 10px;
  cursor: grab;
  background: linear-gradient(180deg, rgba(15,23,42,0.98), rgba(8,13,24,0.96));
  border-bottom: 1px solid rgba(255,255,255,0.08);
}
.clm-header.dragging {
  cursor: grabbing;
}
.clm-header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}
.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background: #64748b;
}
.status-dot.online { background: #22d67c; }
.status-dot.warn { background: #f5a623; }
.status-dot.offline { background: #f04040; }
.clm-title-group {
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.clm-title {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--sc-text, #e2ecff);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.clm-status-text {
  font-size: 0.72rem;
  color: var(--sc-text-muted, #8aa4c8);
  line-height: 1.1;
}
.clm-controls {
  display: flex;
  gap: 4px;
}
.icon-btn,
.toolbar-icon-btn {
  border: none;
  background: transparent;
  color: var(--sc-text, #e2ecff);
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.15s ease;
}
.icon-btn:hover,
.toolbar-icon-btn:hover {
  background: rgba(255,255,255,0.06);
}
.toolbar-icon-btn.active {
  background: rgba(74,158,255,0.16);
  color: #7aa9ff;
}
.icon-btn.text-danger {
  color: #f04040;
}
.clm-body {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
}
.clm-toolbar {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  border-bottom: 1px solid rgba(255,255,255,0.08);
  background: rgba(7,13,22,0.95);
}
.toolbar-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  flex-wrap: nowrap;
  overflow-x: auto;
}
.toolbar-row::-webkit-scrollbar {
  height: 6px;
}
.toolbar-row::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.1);
  border-radius: 3px;
}
.toolbar-group {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: nowrap;
  min-width: 0;
}
.toolbar-main {
  border-right: 1px solid rgba(255,255,255,0.08);
  padding-right: 10px;
  margin-right: 10px;
}
.toolbar-label {
  color: var(--sc-text-muted, #8aa4c8);
  font-size: 0.72rem;
  white-space: nowrap;
}
.toolbar-select,
.toolbar-input {
  height: 28px;
  min-width: 72px;
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 6px;
  background: var(--sc-bg-secondary, #0a1220);
  color: var(--sc-text, #e2ecff);
  padding: 0 8px;
  font-size: 0.78rem;
}
.toolbar-select {
  appearance: none;
}
.toolbar-button {
  height: 28px;
  padding: 0 8px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  background: transparent;
  color: var(--sc-text, #e2ecff);
  font-size: 0.78rem;
  cursor: pointer;
}
.toolbar-button:hover {
  background: rgba(255,255,255,0.05);
}
.toolbar-search-group {
  position: relative;
}
.search-popover {
  position: absolute;
  right: 0;
  top: 42px;
  width: min(320px, 320px);
  background: var(--sc-bg-card, #0d1626);
  border: 1px solid rgba(255,255,255,0.08);
  border-radius: 10px;
  box-shadow: 0 16px 40px rgba(0,0,0,0.45);
  padding: 8px;
  z-index: 10;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.toolbar-search-input {
  width: 100%;
  min-width: 180px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid rgba(255,255,255,0.08);
  background: var(--sc-bg-secondary, #07101f);
  color: var(--sc-text, #e2ecff);
  padding: 0 10px;
  font-size: 0.82rem;
}
.search-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 0.75rem;
  color: var(--sc-text-muted, #8aa4c8);
}
.search-count {
  min-width: 64px;
}
.toolbar-more-btn {
  margin-left: auto;
}
.toolbar-expanded-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 4px;
}
.toolbar-toggle-group {
  border-right: 1px solid rgba(255,255,255,0.08);
  padding-right: 10px;
  margin-right: 10px;
}
.toolbar-actions-group {
  margin-left: auto;
}
.clm-banner {
  font-size: 0.78rem;
  color: #f5a623;
  background: rgba(245,166,35,0.1);
  padding: 8px 10px;
  border-radius: 8px;
  margin: 8px 12px 0;
}
.clm-viewer {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  background: var(--sc-bg-secondary, #06101d);
  border: 1px solid rgba(255,255,255,0.06);
  border-radius: 10px;
  padding: 8px;
  margin: 0 12px 12px;
  font-family: 'JetBrains Mono','Fira Code',ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,'Liberation Mono','Courier New',monospace;
}
.clm-line {
  display: grid;
  grid-template-columns: 46px 1fr;
  gap: 0.65rem;
  padding: 0.2rem 0.4rem;
  min-height: 20px;
  align-items: flex-start;
  font-size: 0.78rem;
  line-height: 1.35;
  white-space: pre;
}
.clm-line.stderr {
  background: rgba(255, 80, 80, 0.05);
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
.resize-handle {
  position: absolute;
  background: transparent;
  z-index: 15;
}
.resize-handle.top-left,
.resize-handle.top-right,
.resize-handle.bottom-left,
.resize-handle.bottom-right {
  width: 12px;
  height: 12px;
}
.resize-handle.top,
.resize-handle.bottom {
  height: 8px;
  left: 12px;
  right: 12px;
}
.resize-handle.left,
.resize-handle.right {
  width: 8px;
  top: 12px;
  bottom: 12px;
}
.resize-handle.top-left { top: 0; left: 0; cursor: nwse-resize; }
.resize-handle.top { top: 0; cursor: ns-resize; }
.resize-handle.top-right { top: 0; right: 0; cursor: nesw-resize; }
.resize-handle.right { right: 0; cursor: ew-resize; }
.resize-handle.bottom-right { right: 0; bottom: 0; cursor: nwse-resize; }
.resize-handle.bottom { bottom: 0; cursor: ns-resize; }
.resize-handle.bottom-left { bottom: 0; left: 0; cursor: nesw-resize; }
.resize-handle.left { left: 0; cursor: ew-resize; }
@media (max-width: 900px) {
  .container-logs-modal {
    left: 0 !important;
    top: 0 !important;
    width: 100vw !important;
    height: 100vh !important;
    min-width: 100%;
    min-height: 100%;
  }
  .clm-frame {
    border-radius: 0 !important;
  }
  .resize-handle { display: none; }
}
</style>

<template>
  <div>
    <PageHeader title="Logs" icon="mdi mdi-file-document" :items="[{text:'Logs',active:true,icon:'mdi mdi-file-document'}]">
      <template #actions>
        <button class="btn btn-sm" :class="streaming ? 'btn-sc-danger' : 'btn-sc-primary'" @click="toggleStream">
          <i :class="`mdi ${streaming ? 'mdi-stop' : 'mdi-play'} me-1`"></i>
          {{ streaming ? 'Pause' : 'Stream' }}
        </button>
      </template>
    </PageHeader>

    <!-- Controls -->
    <div class="card mb-3">
      <div class="card-body py-2">
        <div class="row g-2 align-items-center">
          <div class="col-md-3">
            <ScSelect v-model="source" :options="[
              {value:'auth',label:'auth.log'},
              {value:'syslog',label:'syslog'},
              {value:'kern',label:'kern.log'},
              {value:'nginx',label:'nginx/access.log'},
              {value:'nginx-error',label:'nginx/error.log'},
              {value:'docker',label:'docker daemon'},
              {value:'crowdsec',label:'crowdsec'},
              {value:'fail2ban',label:'fail2ban'},
              {value:'journal',label:'journald (all)'}
            ]" size="sm" @change="loadLogs" />
          </div>
          <div class="col-md-2">
            <ScSelect v-model="severityFilter" :options="[
              {value:'',label:'All levels'},
              {value:'error',label:'ERROR'},
              {value:'warn',label:'WARN'},
              {value:'info',label:'INFO'}
            ]" size="sm" />
          </div>
          <div class="col-md-4">
            <input v-model="searchText" class="form-control form-control-sm" placeholder="Search / regex filter…" />
          </div>
          <div class="col-md-2">
            <ScSelect v-model="lines" :options="[
              {value:20,label:'20 lines'},
              {value:100,label:'100 lines'},
              {value:500,label:'500 lines'},
              {value:1000,label:'1000 lines'}
            ]" size="sm" @change="loadLogs" />
          </div>
          <div class="col-md-1">
            <ScSelect v-model="autoRefreshSec" :options="[
              {value:2,label:'2s'},
              {value:5,label:'5s'},
              {value:10,label:'10s'},
              {value:30,label:'30s'}
            ]" size="sm" :disabled="!streaming" @change="restartPolling" />
          </div>
          <div class="col-md-1">
            <Tooltip label="Clear logs" description="Remove the current in-memory log buffer from this view." variant="rich" as-child>
              <button class="btn btn-sm w-100 log-clear-btn" @click="clearLogs">
                <i class="mdi mdi-delete-outline"></i>
              </button>
            </Tooltip>
          </div>
        </div>
      </div>
    </div>

    <!-- Log table -->
    <div class="card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <div class="d-flex align-items-center gap-2">
          <span class="status-dot" :class="streaming ? 'online' : 'offline'"></span>
          <span class="log-source-label">{{ source }}</span>
          <span class="log-entries-label">{{ filteredLogs.length }} entries</span>
        </div>
        <button class="btn btn-sm log-download-btn" @click="downloadLogs">
          <i class="mdi mdi-download me-1"></i>Export
        </button>
      </div>

      <div class="card-body p-0">
        <div v-if="loading" class="text-center py-4 log-loading-text">
          <i class="mdi mdi-loading mdi-spin me-2"></i>Loading logs…
        </div>
        <div v-else-if="filteredLogs.length === 0" class="text-center py-4 log-empty-text">
          No log entries matching criteria
        </div>
        <div v-else style="overflow-x:auto">
          <table class="table mb-0" style="font-size:0.75rem">
            <thead>
              <tr>
                <th style="width:80px">Level</th>
                <th style="width:160px">Timestamp</th>
                <th style="width:120px">Source</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in paginatedLogs"
                :key="entry.key"
                :class="rowClass(entry.level)"
              >
                <td>
                  <span class="badge rounded-pill" :style="badgeStyle(entry.level)">
                    {{ entry.level.toUpperCase() }}
                  </span>
                </td>
                <td class="font-mono log-timestamp" :style="{ whiteSpace: 'nowrap' }">{{ entry.ts }}</td>
                <td class="font-mono log-source" :style="{ whiteSpace: 'nowrap' }">{{ entry.src }}</td>
                <td :class="{ 'log-line-flash': entry.flash }" class="log-message" :style="{ wordBreak: 'break-all', whiteSpace: 'pre-wrap' }">{{ entry.msg }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="d-flex align-items-center justify-content-between px-3 py-2 log-pagination">
          <span class="log-pagination-info">
            Page {{ currentPage }} of {{ totalPages }} ({{ filteredLogs.length }} entries)
          </span>
          <div class="d-flex gap-1">
            <button class="btn btn-sm log-pagination-btn" :disabled="currentPage === 1" @click="currentPage--">
              <i class="mdi mdi-chevron-left"></i>
            </button>
            <button class="btn btn-sm log-pagination-btn" :disabled="currentPage === totalPages" @click="currentPage++">
              <i class="mdi mdi-chevron-right"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { useDocumentVisibility } from '@vueuse/core'
import PageHeader from '@/components/page-header.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import api from '@/services/api'

const PAGE_SIZE = 100

export default {
  name: 'LogsPage',
  setup() {
    return {
      documentVisibility: useDocumentVisibility()
    }
  },
  components: { PageHeader, Tooltip },
  data() {
    return {
      source: 'auth',
      severityFilter: '',
      searchText: '',
      lines: 20,
      autoRefreshSec: 5,
      streaming: true,
      loading: false,
      logBuffer: [],
      logKeyCounter: 0,
      streamTimer: null,
      currentPage: 1
    }
  },

  computed: {
    parsedLogs() {
      return this.logBuffer.map((entry) => parseLogLine(entry))
    },

    filteredLogs() {
      let list = this.parsedLogs
      if (this.severityFilter) {
        list = list.filter(e => e.level === this.severityFilter || matchSeverity(e.level, this.severityFilter))
      }
      if (this.searchText) {
        try {
          const re = new RegExp(this.searchText, 'i')
          list = list.filter(e => re.test(e.raw))
        } catch {
          const s = this.searchText.toLowerCase()
          list = list.filter(e => e.raw.toLowerCase().includes(s))
        }
      }
      return list
    },

    totalPages() {
      return Math.max(1, Math.ceil(this.filteredLogs.length / PAGE_SIZE))
    },
    paginatedLogs() {
      const start = (this.currentPage - 1) * PAGE_SIZE
      return this.filteredLogs.slice(start, start + PAGE_SIZE)
    }
  },

  watch: {
    documentVisibility(value) {
      if (value === 'visible' && this.streaming) {
        this.loadLogs({ quiet: true })
      }
    },
    source() { this.currentPage = 1; this.loadLogs() },
    lines()  { this.currentPage = 1; this.loadLogs() },
    filteredLogs() {
      if (this.currentPage > this.totalPages) this.currentPage = this.totalPages
    }
  },

  mounted() {
    this.loadLogs()
    if (this.streaming) this.startPolling()
  },
  beforeUnmount() {
    this.stopPolling()
  },

  methods: {
    async loadLogs(opts = {}) {
      const quiet = Boolean(opts.quiet)
      if (!quiet) this.loading = true
      try {
        const { data } = await api.getLogs(this.source, this.lines)
        const lines = Array.isArray(data.lines) ? data.lines : []
        this.mergeIncoming(lines, quiet)
      } catch (e) {
        if (!quiet) {
          this.$swal({ icon: 'error', title: 'Failed to load logs', text: e.response?.data?.error || e.message })
        }
      } finally {
        if (!quiet) this.loading = false
      }
    },

    mergeIncoming(lines, quiet) {
      const incoming = [...lines].reverse() // newest first
      if (!quiet || this.logBuffer.length === 0) {
        this.logBuffer = incoming.map(raw => this.wrapRawLine(raw, false))
        return
      }

      const topRaw = this.logBuffer[0]?.raw
      if (!topRaw) {
        this.logBuffer = incoming.map(raw => this.wrapRawLine(raw, false))
        return
      }

      const pivot = incoming.findIndex((raw) => raw === topRaw)
      if (pivot <= 0) {
        return
      }

      const newEntries = incoming.slice(0, pivot).map(raw => this.wrapRawLine(raw, true))
      if (!newEntries.length) return
      this.logBuffer = [...newEntries, ...this.logBuffer].slice(0, this.lines)

      setTimeout(() => {
        newEntries.forEach((entry) => {
          const idx = this.logBuffer.findIndex((x) => x.key === entry.key)
          if (idx !== -1) this.logBuffer[idx].flash = false
        })
      }, 1200)
    },

    wrapRawLine(raw, flash) {
      this.logKeyCounter += 1
      return { raw, key: `${Date.now()}-${this.logKeyCounter}`, flash }
    },

    startPolling() {
      this.stopPolling()
      this.streamTimer = setInterval(() => {
        if (this.documentVisibility !== 'visible') return
        this.loadLogs({ quiet: true })
      }, this.autoRefreshSec * 1000)
    },
    restartPolling() {
      if (this.streaming) this.startPolling()
    },
    stopPolling() {
      if (this.streamTimer) { clearInterval(this.streamTimer); this.streamTimer = null }
    },
    toggleStream() {
      this.streaming = !this.streaming
      this.streaming ? this.startPolling() : this.stopPolling()
    },
    clearLogs() {
      this.logBuffer = []
      this.currentPage = 1
    },

    badgeStyle(level) {
      const colors = {
        error:   'background:rgba(240,64,64,0.15);color:#f04040',
        warn:    'background:rgba(245,166,35,0.15);color:#f5a623',
        info:    'background:rgba(74,158,255,0.15);color:#4a9eff',
        debug:   'background:rgba(138,164,200,0.12);color:#8aa4c8',
        success: 'background:rgba(34,214,124,0.15);color:#22d67c',
      }
      return (colors[level] || colors.debug) + ';font-size:0.6rem;padding:2px 6px'
    },

    rowClass(level) {
      if (level === 'error')   return 'log-row-error'
      if (level === 'warn')    return 'log-row-warn'
      if (level === 'success') return 'log-row-success'
      return ''
    },

    downloadLogs() {
      const blob = new Blob([this.filteredLogs.map(e => e.raw).join('\n')], { type: 'text/plain' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `${this.source}-${Date.now()}.log`
      a.click()
    }
  }
}

// ── helpers ───────────────────────────────────────────────────────────────────

function matchSeverity(level, filter) {
  if (filter === 'error') return level === 'error'
  if (filter === 'warn')  return level === 'warn'
  if (filter === 'info')  return level === 'info' || level === 'success'
  return true
}

const TS_RE   = /^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}|\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2})/
const SRC_RE  = /\b(sshd|kernel|cron|sudo|fail2ban|psad|ufw|docker|nginx|systemd|crowdsec|auditd)\b/i
const ERR_RE  = /error|fail|crit|emerg|fatal|denied/i
const WARN_RE = /warn|notice|ban|block|reject|invalid/i
const OK_RE   = /accepted|success|start(?:ed)?|enabled|reloaded/i
const INFO_RE = /info|debug|notice/i

function parseLogLine(raw) {
  if (raw && typeof raw === 'object') {
    const parsed = parseLogLine(raw.raw)
    parsed.key = raw.key
    parsed.flash = Boolean(raw.flash)
    return parsed
  }
  const ts  = (raw.match(TS_RE) || [])[1] || ''
  const src = (raw.match(SRC_RE) || [])[1] || ''
  const msg = raw.replace(TS_RE, '').replace(/^\s*\S+\s+/, '').trim()

  let level = 'debug'
  if (ERR_RE.test(raw))  level = 'error'
  else if (WARN_RE.test(raw)) level = 'warn'
  else if (OK_RE.test(raw))   level = 'success'
  else if (INFO_RE.test(raw)) level = 'info'

  return { raw, ts, src, msg: msg || raw, level, key: raw, flash: false }
}
</script>

<style scoped>
.log-row-error td { background: rgba(240,64,64,0.04) }
.log-row-warn  td { background: rgba(245,166,35,0.04) }
.log-row-success td { background: rgba(34,214,124,0.03) }

.log-line-flash {
  animation: logFlash 1.2s ease;
}

@keyframes logFlash {
  0% { background: rgba(74,158,255,0.22); }
  100% { background: transparent; }
}

/* Theme-aware styles for logs page */
.log-source-label {
  font-size: 0.78rem;
  font-family: monospace;
  color: var(--sc-text-secondary);
}

.log-entries-label {
  font-size: 0.72rem;
  color: var(--sc-text-muted);
}

.log-clear-btn {
  background: var(--sc-bg-primary-subtle);
  color: var(--sc-primary);
  font-size: 0.75rem;
}

.log-download-btn {
  background: var(--sc-bg-primary-subtle);
  color: var(--sc-primary);
  font-size: 0.72rem;
}

.log-loading-text {
  color: var(--sc-text-muted);
}

.log-empty-text {
  color: var(--sc-text-muted);
  font-size: 0.8rem;
}

.log-timestamp {
  color: var(--sc-text-muted);
}

.log-source {
  color: var(--sc-text-secondary);
}

.log-message {
  color: var(--sc-text);
}

.log-pagination {
  border-top: 1px solid var(--sc-border);
}

.log-pagination-info {
  font-size: 0.72rem;
  color: var(--sc-text-muted);
}

.log-pagination-btn {
  background: var(--sc-bg-primary-subtle);
  color: var(--sc-primary);
  padding: 2px 8px;
  font-size: 0.72rem;
}
</style>

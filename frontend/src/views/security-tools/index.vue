<template>
  <div>
    <PageHeader title="Security Tools" icon="mdi mdi-shield-search" :items="[{ text: 'Security Tools', active: true, icon: 'mdi mdi-shield-search' }]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" @click="loadTools">
          <i class="mdi mdi-refresh me-1"></i>Refresh
        </button>
      </template>
    </PageHeader>

    <div class="row g-3 mb-3">
      <div v-for="tool in tools" :key="tool.name" class="col-xl-3 col-md-6">
        <div class="card h-100">
          <div class="card-header d-flex align-items-center justify-content-between">
            <div class="d-flex align-items-center gap-2">
              <i :class="toolIcon(tool.name)" style="font-size:1.1rem;color:#4a9eff"></i>
              <h6 class="mb-0">{{ tool.label }}</h6>
            </div>
            <span class="badge" :class="tool.running ? 'badge-warning' : (tool.last_status === 'success' ? 'badge-online' : 'badge-offline')">
              {{ tool.running ? 'RUNNING' : (tool.last_status || 'never run') }}
            </span>
          </div>
          <div class="card-body d-flex flex-column">
            <div style="font-size:0.78rem;color:#8aa4c8">{{ tool.description }}</div>
            <div style="font-size:0.72rem;color:#5a7499" class="mt-2">Last run: {{ tool.last_run_at ? new Date(tool.last_run_at).toLocaleString() : 'Never' }}</div>
            <div class="d-flex gap-2 mt-auto pt-3">
              <button class="btn btn-sm btn-sc-primary" :disabled="tool.running" @click="runTool(tool)">
                <i class="mdi mdi-play me-1"></i>Run
              </button>
              <button class="btn btn-sm btn-outline-secondary" @click="openLogs(tool)">
                <i class="mdi mdi-text-box-search-outline me-1"></i>Logs
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Logs Modal -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="showLogsModal" class="sc-modal-backdrop" @click.self="closeLogsModal">
          <div class="sc-modal sc-modal-lg">
            <div class="sc-modal-header">
              <div class="d-flex align-items-center gap-2">
                <i :class="activeTool ? toolIcon(activeTool.name) : 'mdi mdi-console-line'" style="font-size:1.1rem;color:#4a9eff"></i>
                <h5 class="mb-0">{{ activeTool ? activeTool.label : '' }} — Logs</h5>
              </div>
              <div class="d-flex align-items-center gap-2">
                <span v-if="activeLogs.running" class="badge badge-warning">
                  <i class="mdi mdi-loading mdi-spin me-1"></i>Running
                </span>
                <span v-else-if="activeLogs.logs && activeLogs.logs.length" class="badge badge-online">Done</span>
                <button class="btn btn-sm btn-outline-secondary" @click="pollLogsOnce">
                  <i class="mdi mdi-refresh"></i>
                </button>
                <button class="sc-modal-close" @click="closeLogsModal">
                  <i class="mdi mdi-close"></i>
                </button>
              </div>
            </div>
            <div class="sc-modal-body p-0">
              <div v-if="!activeLogs.logs || activeLogs.logs.length === 0" class="p-4 text-center" style="color:#5a7499">
                <i class="mdi mdi-text-box-remove-outline" style="font-size:2rem;display:block;margin-bottom:0.5rem"></i>
                No log output yet. Run the tool to generate results.
              </div>
              <div v-else ref="logBox" class="tool-log-box">
                <div v-for="(line, idx) in activeLogs.logs" :key="idx" class="tool-log-line font-mono">{{ line }}</div>
                <div v-if="activeLogs.running" class="tool-log-line font-mono" style="color:#f5a623">
                  <i class="mdi mdi-loading mdi-spin me-1"></i>running...
                </div>
              </div>
            </div>
            <div class="sc-modal-footer">
              <div style="font-size:0.72rem;color:#5a7499">
                {{ activeLogs.logs ? activeLogs.logs.length : 0 }} lines
                <span v-if="activeTool && activeTool.last_run_at"> · Last run: {{ new Date(activeTool.last_run_at).toLocaleString() }}</span>
              </div>
              <button class="btn btn-sm btn-outline-secondary" @click="closeLogsModal">Close</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import api from '@/services/api'

const TOOL_ICONS = {
  rkhunter:    'mdi mdi-bug-check-outline',
  clamav:      'mdi mdi-virus-off-outline',
  lynis:       'mdi mdi-shield-check-outline',
  chkrootkit:  'mdi mdi-magnify-scan'
}

export default {
  name: 'SecurityToolsPage',
  components: { PageHeader },
  data() {
    return {
      tools: [],
      activeTool: null,
      activeLogs: { logs: [], running: false },
      showLogsModal: false,
      pollTimer: null
    }
  },
  async mounted() {
    await this.loadTools()
  },
  beforeUnmount() {
    this.stopPolling()
  },
  methods: {
    toolIcon(name) {
      return TOOL_ICONS[name] || 'mdi mdi-shield-search'
    },
    stopPolling() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer)
        this.pollTimer = null
      }
    },
    async loadTools() {
      const { data } = await api.getSecurityTools()
      this.tools = Array.isArray(data) ? data : []
    },
    async runTool(tool, installAttempted = false) {
      try {
        await api.runSecurityTool(tool.name)
      } catch (err) {
        const msg = err.detailedMessage || err.message || 'Failed to start tool'
        if (!installAttempted && msg.toLowerCase().includes('not installed')) {
          const confirmed = await this.$swal({
            icon: 'warning',
            title: `${tool.label} is not installed`,
            text: `${tool.label} must be installed before it can run. Install it now?`,
            showCancelButton: true,
            confirmButtonText: 'Install',
            cancelButtonText: 'Cancel'
          })
          if (confirmed.isConfirmed) {
            try {
              await api.installSecurityTool(tool.name)
              this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${tool.label} install started`, showConfirmButton: false, timer: 2500 })
              await this.loadTools()
              return this.runTool(tool, true)
            } catch (installErr) {
              const installMsg = installErr.detailedMessage || installErr.message || 'Install failed'
              this.$swal({ icon: 'error', title: `Install failed`, text: installMsg })
            }
          }
        } else {
          this.$swal({ icon: 'error', title: `Failed to start ${tool.label}`, text: msg })
        }
        return
      }

      this.activeTool = tool
      this.activeLogs = { logs: [], running: true }
      this.showLogsModal = true
      await this.pollLogsOnce()
      await this.loadTools()
      this.stopPolling()
      this.pollTimer = setInterval(() => this.pollLogsOnce(), 2000)
    },
    async openLogs(tool) {
      this.activeTool = tool
      this.activeLogs = { logs: [], running: false }
      this.showLogsModal = true
      await this.pollLogsOnce()
      this.stopPolling()
      if (this.activeLogs.running) {
        this.pollTimer = setInterval(() => this.pollLogsOnce(), 2000)
      }
    },
    closeLogsModal() {
      this.showLogsModal = false
      this.stopPolling()
    },
    async pollLogsOnce() {
      if (!this.activeTool) return
      try {
        const { data } = await api.getSecurityToolLogs(this.activeTool.name)
        this.activeLogs = data || { logs: [], running: false }
        await this.loadTools()
        // Keep activeTool in sync with loaded data
        const updated = this.tools.find(t => t.name === this.activeTool.name)
        if (updated) this.activeTool = updated
        this.$nextTick(() => {
          const el = this.$refs.logBox
          if (el) el.scrollTop = el.scrollHeight
        })
        if (!this.activeLogs.running) {
          this.stopPolling()
        }
      } catch (e) {
        console.error('Failed to fetch tool logs:', e)
      }
    }
  }
}
</script>

<style scoped>
.tool-log-box {
  height: 420px;
  overflow-y: auto;
  background: #07111f;
  padding: 0.8rem;
}
.tool-log-line {
  color: #93b7d9;
  font-size: 0.75rem;
  line-height: 1.35;
  white-space: pre-wrap;
  word-break: break-word;
}
/* Modal transitions */
.modal-fade-enter-active,
.modal-fade-leave-active { transition: opacity 0.2s ease, transform 0.2s ease; }
.modal-fade-enter-from,
.modal-fade-leave-to { opacity: 0; }
.modal-fade-enter-from .sc-modal,
.modal-fade-leave-to .sc-modal { transform: translateY(-16px) scale(0.98); }
</style>

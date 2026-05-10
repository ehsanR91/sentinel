<template>
  <div class="sc-view sc-view-tasks">
    <PageHeader title="Tasks" icon="mdi mdi-clock-outline" :items="[{ text: 'Tasks', active: true, icon: 'mdi mdi-clock-outline' }]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" :disabled="loading" @click="loadTasks">
          <i :class="`mdi ${loading ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>Refresh
        </button>
      </template>
    </PageHeader>

    <div class="row g-3 mb-4">
      <div class="col-xl-3 col-md-6"><StatCard label="Defined" :value="stats.total || 0" sub="task definitions" icon="mdi mdi-playlist-check" icon-color="#4a9eff" icon-bg="rgba(74,158,255,.12)" /></div>
      <div class="col-xl-3 col-md-6"><StatCard label="Enabled" :value="stats.enabled || 0" sub="scheduled/manual" icon="mdi mdi-check-decagram" icon-color="#22d67c" icon-bg="rgba(34,214,124,.12)" /></div>
      <div class="col-xl-3 col-md-6"><StatCard label="Runs" :value="stats.runs || 0" sub="total executions" icon="mdi mdi-counter" icon-color="#22d3ee" icon-bg="rgba(34,211,238,.12)" /></div>
      <div class="col-xl-3 col-md-6"><StatCard label="Failures" :value="stats.failed_runs || 0" sub="needs attention" icon="mdi mdi-alert-circle" icon-color="#f04040" icon-bg="rgba(240,64,64,.12)" /></div>
    </div>

    <!-- Task Templates -->
    <div class="card sc-panel-card mb-4">
      <div class="card-header">
        <h6><i class="mdi mdi-package-variant-closed me-2" style="color:var(--sc-purple)"></i>Quick Templates</h6>
      </div>
      <div class="card-body">
        <div class="row g-2">
          <div class="col-md-3 col-sm-6" v-for="tpl in taskTemplates" :key="tpl.id">
            <button class="btn btn-sm w-100 text-start task-template-btn" @click="applyTemplate(tpl)">
              <i :class="`mdi ${tpl.icon} me-2`" style="color:var(--sc-blue)"></i>
              <div class="template-info">
                <div class="template-name">{{ tpl.name }}</div>
                <div class="template-desc">{{ tpl.description }}</div>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="card sc-panel-card mb-4">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-plus-circle-outline me-2" style="color:var(--sc-blue)"></i>{{ editingId ? 'Edit Task' : 'Create Task' }}</h6>
        <button v-if="!editingId" class="btn btn-sm" style="background:rgba(74,158,255,.1);color:#4a9eff;font-size:.72rem" @click="resetForm">
          <i class="mdi mdi-refresh me-1"></i>Clear Form
        </button>
      </div>
      <div class="card-body">
        <div class="row g-3">
          <div class="col-xl-3 col-md-6">
            <label class="form-label">Name</label>
            <input v-model="form.name" class="form-control form-control-sm" placeholder="Task name" />
          </div>
          <div class="col-xl-5 col-md-6">
            <label class="form-label">Command <span style="color:var(--sc-red);font-size:.72rem">(protected)</span></label>
            <input v-model="form.command" class="form-control form-control-sm font-mono" placeholder="bash command" :class="{ 'is-invalid': commandWarning }" />
            <div v-if="commandWarning" class="form-text" style="color:var(--sc-red);font-size:.7rem">
              <i class="mdi mdi-alert me-1"></i>{{ commandWarning }}
            </div>
          </div>
          <div class="col-xl-2 col-md-6">
            <label class="form-label">Schedule</label>
            <ScSelect v-model="form.schedule_kind" :options="[{value:'manual',label:'Manual'},{value:'interval',label:'Interval'}]" size="sm" />
          </div>
          <div class="col-xl-2 col-md-6">
            <label class="form-label">Interval (sec)</label>
            <input v-model="form.schedule_expr" class="form-control form-control-sm" :disabled="form.schedule_kind !== 'interval'" placeholder="e.g. 3600" />
          </div>
          <div class="col-12">
            <label class="form-label">Description</label>
            <input v-model="form.description" class="form-control form-control-sm" placeholder="What this task does" />
          </div>
        </div>
        <div class="mt-3 d-flex gap-2">
          <button class="btn btn-sm btn-sc-primary" :disabled="saving || commandWarning" @click="saveTask">
            <i :class="`mdi ${saving ? 'mdi-loading mdi-spin' : 'mdi-content-save'} me-1`"></i>{{ editingId ? 'Update Task' : 'Create Task' }}
          </button>
          <button v-if="editingId" class="btn btn-sm" style="background:rgba(240,64,64,.12);color:#f04040" @click="resetForm">Cancel Edit</button>
        </div>
      </div>
    </div>

    <div class="card sc-panel-card mb-4">
      <div class="card-header"><h6><i class="mdi mdi-table-cog me-2" style="color:var(--sc-cyan)"></i>Task Definitions</h6></div>
      <div class="card-body p-0">
        <table class="table mb-0">
          <thead>
            <tr>
              <th>Name</th>
              <th>Command</th>
              <th>Schedule</th>
              <th>Last Run</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="!tasks.length"><td colspan="6" class="text-center py-4 sc-text-muted">No tasks yet</td></tr>
            <tr v-for="t in tasks" :key="t.id">
              <td>
                <div class="task-name">{{ t.name }}</div>
                <div class="task-desc">{{ t.description || '—' }}</div>
              </td>
              <td class="font-mono task-command">
                <Tooltip :label="t.name" :description="t.command" variant="rich" as-child>
                  <span class="d-inline-block w-100 text-truncate">{{ truncateCommand(t.command) }}</span>
                </Tooltip>
              </td>
              <td class="task-meta">
                <span class="badge badge-info me-1">{{ t.schedule_kind }}</span>
                <span v-if="t.schedule_kind === 'interval'" class="font-mono">{{ t.schedule_expr }}s</span>
              </td>
              <td class="task-meta sc-text-muted">{{ t.last_run ? formatAgo(t.last_run.started_at) : 'never' }}</td>
              <td>
                <span class="badge rounded-pill" :class="badgeForTask(t)">{{ textForTask(t) }}</span>
              </td>
              <td>
                <div class="d-flex gap-1 flex-wrap">
                  <button class="btn btn-sm btn-task-run" @click="runNow(t)">Run</button>
                  <button class="btn btn-sm btn-task-edit" @click="editTask(t)">Edit</button>
                  <button class="btn btn-sm btn-task-delete" @click="deleteTask(t)">Delete</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="card sc-panel-card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-text-box-search-outline me-2" style="color:var(--sc-amber)"></i>Recent Runs</h6>
        <div class="d-flex align-items-center gap-2">
          <span class="task-meta sc-text-muted">{{ runsPage * runsPageSize + 1 }}–{{ Math.min((runsPage + 1) * runsPageSize, runs.length) }} of {{ runs.length }}</span>
          <button class="btn btn-sm btn-task-edit" :disabled="runsPage === 0" @click="runsPage--">‹</button>
          <button class="btn btn-sm btn-task-edit" :disabled="(runsPage + 1) * runsPageSize >= runs.length" @click="runsPage++">›</button>
        </div>
      </div>
      <div class="card-body p-0">
        <table class="table mb-0">
          <thead>
            <tr><th>Task</th><th>Started</th><th>Ended</th><th>Status</th><th>Exit</th><th>Output</th></tr>
          </thead>
          <tbody>
            <tr v-if="!runs.length"><td colspan="6" class="text-center py-4 sc-text-muted">No run history yet</td></tr>
            <tr v-for="r in pagedRuns" :key="r.id">
              <td class="task-meta">{{ taskName(r.task_id) }}</td>
              <td class="task-meta sc-text-secondary">{{ formatDate(r.started_at) }}</td>
              <td class="task-meta sc-text-secondary">{{ r.ended_at ? formatDate(r.ended_at) : 'running' }}</td>
              <td><span class="badge rounded-pill" :class="r.status === 'success' ? 'badge-online' : (r.status === 'running' ? 'badge-info' : 'badge-offline')">{{ r.status }}</span></td>
              <td class="font-mono task-meta sc-text-muted">{{ r.exit_code }}</td>
              <td>
                <button class="btn btn-sm btn-task-edit" @click="showOutput(r)">View</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard from '@/components/widgets/stat-card.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import api from '@/services/api'

export default {
  name: 'TasksPage',
  components: { PageHeader, StatCard, Tooltip },

  data() {
    return {
      loading: false,
      saving: false,
      tasks: [],
      runs: [],
      stats: {},
      editingId: null,
      runsPage: 0,
      runsPageSize: 20,
      form: {
        name: '',
        description: '',
        command: '',
        schedule_kind: 'manual',
        schedule_expr: '',
        enabled: true
      },
      // Task templates
      taskTemplates: [
        { id: 'clamav-scan', name: 'ClamAV Scan', icon: 'mdi mdi-virus', description: 'Full system virus scan', command: 'clamscan -r /home', schedule_kind: 'interval', schedule_expr: '86400' },
        { id: 'docker-prune', name: 'Docker Prune', icon: 'mdi mdi-broom', description: 'Remove unused Docker data', command: 'docker system prune -f', schedule_kind: 'interval', schedule_expr: '86400' },
        { id: 'apt-upgrade', name: 'APT Upgrade', icon: 'mdi mdi-arrow-up-bold', description: 'Update and upgrade packages', command: 'apt-get update && apt-get upgrade -y', schedule_kind: 'interval', schedule_expr: '86400' },
        { id: 'log-rotate', name: 'Log Rotation', icon: 'mdi mdi-file-document-edit', description: 'Rotate system logs', command: 'logrotate -f /etc/logrotate.conf', schedule_kind: 'interval', schedule_expr: '86400' },
        { id: 'disk-cleanup', name: 'Disk Cleanup', icon: 'mdi mdi-trash-can', description: 'Clean apt cache and temp files', command: 'apt-get clean && rm -rf /tmp/*', schedule_kind: 'manual', schedule_expr: '' },
        { id: 'backup-check', name: 'Backup Check', icon: 'mdi mdi-backup-restore', description: 'Verify backup integrity', command: 'du -sh /backup 2>/dev/null || echo "No backup dir"', schedule_kind: 'interval', schedule_expr: '86400' }
      ]
    }
  },

  computed: {
    pagedRuns() {
      const start = this.runsPage * this.runsPageSize
      return this.runs.slice(start, start + this.runsPageSize)
    },
    commandWarning() {
      if (!this.form.command) return null
      const cmd = this.form.command.toLowerCase()
      // Mirror the backend blockedPatterns from handlers_terminal.go
      const blocked = [
        'rm -rf /', 'rm -rf /*', 'mkfs', 'dd if=', 'dd of=/dev/',
        ':(){:|:&};:', 'chmod -r 777 /', 'chmod 777 /',
        'rm -rf /etc', 'rm -rf /usr', 'rm -rf /bin',
        'rm -rf /sbin', 'rm -rf /boot', 'rm -rf /var',
        '> /dev/sda', 'shred /dev'
      ]
      for (const pat of blocked) {
        if (cmd.includes(pat)) return `Blocked: "${pat}" is not permitted`
      }
      // Warn (but don't block) on shell operators that the backend flags as high-risk
      const risky = ['&&', '||', ';', ' | ', ' > ', ' < ', '`', '$()']
      for (const op of risky) {
        if (cmd.includes(op)) return `Warning: shell operator "${op.trim()}" — backend may reject this command`
      }
      return null
    }
  },

  async mounted() {
    await this.loadTasks()
  },

  methods: {
    truncateCommand(cmd) {
      return cmd && cmd.length > 48 ? cmd.slice(0, 48) + '…' : cmd
    },
    async loadTasks() {
      this.loading = true
      try {
        const { data } = await api.getTasks()
        this.tasks = data.tasks || []
        this.runs = data.runs || []
        this.stats = data.stats || {}
        this.runsPage = 0
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Failed to load tasks', text: e.response?.data?.error || e.message })
      } finally {
        this.loading = false
      }
    },
    resetForm() {
      this.editingId = null
      this.form = { name: '', description: '', command: '', schedule_kind: 'manual', schedule_expr: '', enabled: true }
    },
    applyTemplate(tpl) {
      this.editingId = null
      this.form = {
        name: tpl.name,
        description: tpl.description,
        command: tpl.command,
        schedule_kind: tpl.schedule_kind,
        schedule_expr: tpl.schedule_expr,
        enabled: true
      }
      // Scroll to form
      this.$nextTick(() => {
        document.querySelector('.sc-panel-card')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })
    },
    editTask(t) {
      this.editingId = t.id
      this.form = {
        name: t.name,
        description: t.description,
        command: t.command,
        schedule_kind: t.schedule_kind,
        schedule_expr: t.schedule_expr,
        enabled: t.enabled
      }
    },
    async saveTask() {
      if (!this.form.name || !this.form.command) {
        this.$swal({ toast: true, position: 'top-end', icon: 'warning', title: 'Name and command are required', showConfirmButton: false, timer: 2200 })
        return
      }
      this.saving = true
      try {
        if (this.editingId) await api.updateTask(this.editingId, this.form)
        else await api.createTask(this.form)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: this.editingId ? 'Task updated' : 'Task created', showConfirmButton: false, timer: 2000 })
        this.resetForm()
        await this.loadTasks()
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Save failed', text: e.response?.data?.error || e.message })
      } finally {
        this.saving = false
      }
    },
    async deleteTask(t) {
      const r = await this.$swal({ title: `Delete ${t.name}?`, icon: 'warning', showCancelButton: true, confirmButtonText: 'Delete' })
      if (!r.isConfirmed) return
      try {
        await api.deleteTask(t.id)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Task deleted', showConfirmButton: false, timer: 1800 })
        await this.loadTasks()
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Delete failed', text: e.response?.data?.error || e.message })
      }
    },
    async runNow(t) {
      try {
        await api.runTaskNow(t.id)
        this.$swal({ toast: true, position: 'top-end', icon: 'info', title: `Started ${t.name}`, showConfirmButton: false, timer: 1800 })
        setTimeout(() => this.loadTasks(), 1000)
      } catch (e) {
        this.$swal({ icon: 'error', title: 'Run failed', text: e.response?.data?.error || e.message })
      }
    },
    taskName(taskId) {
      return this.tasks.find(t => t.id === taskId)?.name || `Task #${taskId}`
    },
    formatDate(ts) {
      if (!ts) return '—'
      return new Date(ts * 1000).toLocaleString()
    },
    formatAgo(ts) {
      if (!ts) return '—'
      const sec = Math.max(0, Math.floor(Date.now() / 1000 - ts))
      if (sec < 60) return `${sec}s ago`
      const min = Math.floor(sec / 60)
      if (min < 60) return `${min}m ago`
      const hrs = Math.floor(min / 60)
      if (hrs < 24) return `${hrs}h ago`
      return `${Math.floor(hrs / 24)}d ago`
    },
    badgeForTask(t) {
      if (!t.last_run) return t.enabled ? 'badge-info' : 'badge-warning'
      if (t.last_run.status === 'success') return 'badge-online'
      if (t.last_run.status === 'running') return 'badge-info'
      return 'badge-offline'
    },
    textForTask(t) {
      if (!t.last_run) return t.enabled ? 'enabled' : 'disabled'
      return t.last_run.status
    },
    showOutput(run) {
      this.$swal({
        title: `${this.taskName(run.task_id)} output`,
        html: `<pre style="text-align:left;max-height:360px;overflow:auto;background:var(--sc-bg-secondary);padding:12px;border-radius:8px;border:1px solid var(--sc-border);color:var(--sc-text)">${(run.output || 'No output').replace(/</g, '&lt;')}</pre>`,
        width: 900,
        confirmButtonText: 'Close'
      })
    }
  }
}
</script>

<style scoped>
.sc-view-tasks :deep(.card-header) { padding: .85rem 1rem; }
.sc-view-tasks :deep(.card-body) { padding: 1rem; }

/* Text utilities */
.sc-text-muted    { color: var(--sc-text-muted, #5a7499); }
.sc-text-secondary { color: var(--sc-text-secondary, #8aa4c8); }

/* Table cell typography */
.task-name    { font-size: .8rem; font-weight: 600; color: var(--sc-text); }
.task-desc    { font-size: .7rem; color: var(--sc-text-muted); }
.task-command { font-size: .72rem; color: var(--sc-text-secondary); max-width: 220px; }
.task-meta    { font-size: .72rem; }

/* Task action buttons */
.btn-task-run    { background: rgba(34,214,124,.12) !important; color: #22d67c !important; padding: 2px 8px !important; font-size: .68rem !important; }
.btn-task-edit   { background: rgba(74,158,255,.12) !important; color: #4a9eff !important; padding: 2px 8px !important; font-size: .68rem !important; }
.btn-task-delete { background: rgba(240,64,64,.12) !important; color: #f04040 !important; padding: 2px 8px !important; font-size: .68rem !important; }
.btn-task-run:disabled,
.btn-task-edit:disabled,
.btn-task-delete:disabled { opacity: .45; cursor: not-allowed; }

/* Task Template Buttons */
.task-template-btn {
  background: var(--sc-bg-secondary);
  border: 1px solid var(--sc-border);
  border-radius: 8px;
  padding: 12px;
  text-align: left;
  transition: all 0.2s ease;
  height: 100%;
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.task-template-btn:hover {
  background: var(--sc-bg-card);
  border-color: var(--sc-blue);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.template-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.template-name {
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--sc-text);
}

.template-desc {
  font-size: 0.7rem;
  color: var(--sc-text-muted);
}
</style>

<template>
  <div class="sc-view sc-view-containers">
    <PageHeader title="Containers" icon="mdi mdi-box" :items="[{text:'Containers',active:true,icon:'mdi mdi-box'}]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" :disabled="pruneBusy" @click="runPrune('all', 'Everything')">
          <i :class="`mdi ${pruneBusy ? 'mdi-loading mdi-spin' : 'mdi-broom'} me-1`"></i> Prune All
        </button>
      </template>
    </PageHeader>

    <div class="card sc-panel-card mb-4">
      <div class="card-body py-2">
        <div class="d-flex flex-wrap gap-2 align-items-center">
          <span style="font-size:0.75rem;color:var(--sc-text-muted)">Cleanup operations:</span>
          <button class="btn btn-sm" style="background:rgba(74,158,255,0.1);color:#4a9eff" :disabled="pruneBusy" @click="runPrune('images', 'Unused Images')">Images</button>
          <button class="btn btn-sm" style="background:rgba(167,139,250,0.12);color:#a78bfa" :disabled="pruneBusy" @click="runPrune('volumes', 'Unused Volumes')">Volumes</button>
          <button class="btn btn-sm" style="background:rgba(34,214,124,0.12);color:#22d67c" :disabled="pruneBusy" @click="runPrune('containers', 'Stopped Containers')">Containers</button>
          <button class="btn btn-sm" style="background:rgba(245,166,35,0.12);color:#f5a623" :disabled="pruneBusy" @click="runPrune('build', 'Build Cache')">Build Cache</button>
        </div>
      </div>
    </div>

    <!-- Docker not available alert -->
    <div v-if="!dockerAvailable" class="alert mb-4" style="background:rgba(240,64,64,0.1);border:1px solid rgba(240,64,64,0.3);color:#f04040;border-radius:8px;padding:1rem 1.25rem">
      <i class="mdi mdi-alert-circle me-2"></i>
      <strong>Docker not available on this system.</strong>
      Docker may not be installed or the Docker daemon is not running.
    </div>

    <!-- Summary cards -->
    <div class="row g-3 mb-4">
      <div class="col-xl-3 col-md-6">
        <StatCard label="Running" :value="containers.filter(c=>c.status==='running').length" sub="containers" icon="mdi mdi-play-circle" icon-color="#22d67c" icon-bg="rgba(34,214,124,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Stopped" :value="containers.filter(c=>c.status!=='running').length" sub="containers" icon="mdi mdi-stop-circle" icon-color="#f04040" icon-bg="rgba(240,64,64,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Images" :value="images.length" sub="total pulled" icon="mdi mdi-layers-outline" icon-color="#4a9eff" icon-bg="rgba(74,158,255,0.12)" />
      </div>
      <div class="col-xl-3 col-md-6">
        <StatCard label="Docker Disk" :value="dockerDisk" sub="total usage" icon="mdi mdi-harddisk" icon-color="#a78bfa" icon-bg="rgba(167,139,250,0.12)" />
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="text-center py-5">
      <div class="spinner-border text-primary" style="width:2rem;height:2rem"></div>
      <div style="color:var(--sc-text-muted);font-size:0.8rem;margin-top:0.5rem">Loading containers…</div>
    </div>

    <template v-else>
      <!-- Containers table -->
      <div class="card sc-panel-card mb-4">
        <div class="card-header d-flex align-items-center justify-content-between">
          <h6><i class="mdi mdi-docker me-2" style="color:#22d3ee"></i>Containers</h6>
          <input v-model="containerFilter" class="form-control form-control-sm" placeholder="Filter…" style="width:160px" />
        </div>
        <div class="card-body p-0">
          <table class="table mb-0">
            <thead>
              <tr><th>Name</th><th>Image</th><th>Status</th><th>Ports</th><th>Actions</th></tr>
            </thead>
            <tbody>
              <tr v-if="filteredContainers.length === 0">
                <td colspan="5" class="text-center py-4" style="color:var(--sc-text-muted);font-size:0.8rem">
                  {{ dockerAvailable ? 'No containers found' : 'Docker not available' }}
                </td>
              </tr>
              <tr v-for="c in filteredContainers" :key="c.id">
                <td>
                  <div class="d-flex align-items-center gap-2">
                    <span class="status-dot" :class="c.status==='running'?'online':'offline'"></span>
                    <span class="font-mono" style="font-size:0.78rem;color:var(--sc-text)">{{ c.name }}</span>
                  </div>
                </td>
                <td style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ c.image }}</td>
                <td>
                  <span class="badge rounded-pill" :class="c.status==='running'?'badge-online':c.status==='paused'?'badge-warning':'badge-offline'">
                    {{ c.statusText || c.status }}
                  </span>
                </td>
                <td class="font-mono" style="font-size:0.72rem;color:var(--sc-blue)">{{ c.ports || '—' }}</td>
                <td>
                  <div class="d-flex gap-1 flex-wrap">
                    <button class="btn btn-sm" style="background:rgba(34,214,124,0.1);color:#22d67c;padding:2px 7px;font-size:0.68rem" @click="startStop(c)" :title="c.status==='running'?'Stop':'Start'">
                      <i :class="`mdi ${c.status==='running'?'mdi-stop':'mdi-play'}`"></i>
                    </button>
                    <button class="btn btn-sm" style="background:rgba(74,158,255,0.1);color:#4a9eff;padding:2px 7px;font-size:0.68rem" @click="restartContainer(c)" title="Restart">
                      <i class="mdi mdi-restart"></i>
                    </button>
                    <button class="btn btn-sm" style="background:rgba(74,158,255,0.1);color:#4a9eff;padding:2px 7px;font-size:0.68rem" @click="viewLogs(c)" title="Logs">
                      <i class="mdi mdi-text-box-outline"></i>
                    </button>
                    <template v-if="c.status === 'running' && parsePorts(c.ports).length">
                      <button
                        v-for="port in parsePorts(c.ports)"
                        :key="port"
                        class="btn btn-sm"
                        style="background:rgba(245,166,35,0.1);color:#f5a623;padding:2px 7px;font-size:0.68rem;white-space:nowrap"
                        :disabled="grantingPort === port"
                        :title="`Grant temporary UFW access for your IP to port ${port}`"
                        @click="grantAccess(c, port)"
                      >
                        <i :class="`mdi ${grantingPort === port ? 'mdi-loading mdi-spin' : 'mdi-shield-key-outline'} me-1`"></i>:{{ port }}
                      </button>
                    </template>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="row g-3">
        <!-- Images -->
        <div class="col-xl-7">
          <div class="card sc-panel-card">
            <div class="card-header"><h6><i class="mdi mdi-layers-outline me-2" style="color:#4a9eff"></i>Images</h6></div>
            <div class="card-body p-0">
              <div v-if="images.length === 0" class="text-center py-4" style="color:var(--sc-text-muted);font-size:0.8rem">
                No image data available
              </div>
              <table v-else class="table mb-0">
                <thead><tr><th>Repository</th><th>Tag</th><th>ID</th><th>Size</th><th>Created</th></tr></thead>
                <tbody>
                  <tr v-for="img in images" :key="img.id">
                    <td class="font-mono" style="font-size:0.78rem;color:var(--sc-text)">{{ img.repo }}</td>
                    <td><span class="badge badge-info" style="font-size:0.65rem">{{ img.tag }}</span></td>
                    <td class="font-mono" style="font-size:0.72rem;color:var(--sc-text-muted)">{{ img.id }}</td>
                    <td style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ img.size }}</td>
                    <td style="font-size:0.72rem;color:var(--sc-text-muted)">{{ img.created }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- Volumes -->
        <div class="col-xl-5">
          <div class="card sc-panel-card">
            <div class="card-header"><h6><i class="mdi mdi-database me-2" style="color:#a78bfa"></i>Volumes</h6></div>
            <div class="card-body p-0">
              <div v-if="volumes.length === 0" class="text-center py-4" style="color:var(--sc-text-muted);font-size:0.8rem">
                No volume data available
              </div>
              <table v-else class="table mb-0">
                <thead><tr><th>Name</th><th>Driver</th><th>Used</th></tr></thead>
                <tbody>
                  <tr v-for="vol in volumes" :key="vol.name">
                    <td class="font-mono" style="font-size:0.75rem;color:var(--sc-text)">{{ vol.name }}</td>
                    <td style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ vol.driver }}</td>
                    <td style="font-size:0.75rem;color:var(--sc-text-muted)">{{ vol.size }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>

      <div v-for="modal in containerLogs" :key="modal.modalId">
        <ContainerLogsModal
          :modal="modal"
          :container="modal.container"
          @focus="bringModalFront"
          @close="closeLogModal"
          @update="updateLogModal"
        />
      </div>
    </template>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import StatCard   from '@/components/widgets/stat-card.vue'
import ContainerLogsModal from '@/components/container-logs-modal.vue'
import api from '@/services/api'

export default {
  name: 'ContainersPage',
  components: { PageHeader, StatCard, ContainerLogsModal },
  data() {
    return {
      loading: false,
      dockerAvailable: true,
      dockerDisk: '—',
      containerFilter: '',
      containers: [],
      images: [],
      volumes: [],
      pruneBusy: false,
      grantingPort: null,
      clientIp: '',
      serverHost: window.location.hostname,
      containerLogs: [],
      nextLogModalId: 1,
      maxModalZ: 1000
    }
  },

  computed: {
    filteredContainers() {
      if (!this.containerFilter) return this.containers
      const f = this.containerFilter.toLowerCase()
      return this.containers.filter(c => c.name.toLowerCase().includes(f) || c.image.toLowerCase().includes(f))
    }
  },

  mounted() {
    this.loadContainers()
    this.fetchClientIp()
    this.detectServerHost()
  },

  methods: {
    async fetchClientIp() {
      try {
        const { data } = await api.getMe()
        if (data?.client_ip) this.clientIp = data.client_ip
      } catch (_) {}
    },

    async detectServerHost() {
      try {
        const { data } = await api.getTunnelableApps()
        if (data?.connection?.host) {
          this.serverHost = data.connection.host
        }
      } catch (_) {
        // Fallback to browser hostname when tunnel service detection is unavailable.
      }
    },

    // Parse backend format "9000->9000/tcp, 9443->9443/tcp" → [9000, 9443]
    // Only include ports that have a public mapping (i.e. host port exists)
    parsePorts(portsStr) {
      if (!portsStr) return []
      // Match patterns like "9000->9000/tcp" — capture the public (host) port
      const re = /(\d+)->\d+\/\w+/g
      const ports = []
      let m
      while ((m = re.exec(portsStr)) !== null) {
        const p = parseInt(m[1])
        if (p && !ports.includes(p)) ports.push(p)
      }
      return ports
    },

    async grantAccess(container, port) {
      const ip = this.clientIp || 'your current IP'
      let durationHours = 3

      if (this.$swal) {
        const html = `
          <div style="margin-top:8px;text-align:left">
            <p style="margin-bottom:8px;color:#c9d8f0;font-size:0.87rem">
              Allow <strong>${container.name}</strong> port <strong>${port}</strong>
              to be accessed directly from <strong>${ip}</strong>.
            </p>
            <label style="font-size:0.8rem;color:#8fa8c8;display:block;margin-bottom:4px">Duration:</label>
            <select id="sc-grant-dur"
              style="width:100%;padding:6px 10px;border-radius:6px;border:1px solid rgba(100,140,200,0.3);background:#0e1c30;color:#c9d8f0;font-size:0.85rem">
              <option value="1">1 hour</option>
              <option value="3" selected>3 hours</option>
              <option value="6">6 hours</option>
              <option value="12">12 hours</option>
              <option value="24">24 hours</option>
            </select>
          </div>`
        const res = await this.$swal({
          title: 'Grant Temporary Access?',
          html,
          icon: 'warning',
          showCancelButton: true,
          confirmButtonText: 'Grant Access',
          cancelButtonText: 'Cancel',
          confirmButtonColor: '#f5a623',
          preConfirm: () => parseInt(document.getElementById('sc-grant-dur')?.value) || 3,
        })
        if (!res.isConfirmed) return
        durationHours = res.value
      } else {
        if (!window.confirm(`Grant access to ${container.name} port ${port} for ${ip}?`)) return
      }

      this.grantingPort = port
      try {
        const { data } = await api.grantTunnelAccess(port, durationHours)
        const grantedIp = data?.ip || ip
        const dh = data?.duration_hours || durationHours
        const expiresAt = data?.expires_at ? new Date(data.expires_at * 1000).toLocaleString() : `in ${dh}h`
        const host = data?.host || this.serverHost
        const browseUrl = `http://${host}:${port}`
        const isProxy = data?.mode === 'nat' || data?.mode === 'proxy'
        const modeNote = isProxy
          ? `<p style="margin-top:8px;padding:6px 10px;border-radius:6px;background:rgba(34,214,124,0.08);color:#22d67c;font-size:0.78rem"><i class="mdi mdi-router-network me-1"></i>Proxy mode: forwarding ${host}:${port} → 127.0.0.1:${port}</p>`
          : `<p style="margin-top:8px;padding:6px 10px;border-radius:6px;background:rgba(74,158,255,0.08);color:#4a9eff;font-size:0.78rem"><i class="mdi mdi-shield-check me-1"></i>UFW rule added for ${grantedIp}</p>`
        this.$swal({
          title: 'Access Granted \u2713',
          html: `<div style="text-align:left;font-size:0.88rem;color:#c9d8f0">
            <p><strong>IP:</strong> ${grantedIp} &nbsp; <strong>Port:</strong> ${port} &nbsp; <strong>Duration:</strong> ${dh}h</p>
            <p><strong>Expires:</strong> ${expiresAt}</p>
            ${modeNote}
            <p style="margin-top:10px">Open: <a href="${browseUrl}" target="_blank" style="color:#4a9eff">${browseUrl}</a></p>
          </div>`,
          icon: 'success',
          confirmButtonText: 'OK'
        })
      } catch (err) {
        const msg = err?.response?.data?.error || err?.response?.data?.message || 'Failed to grant access'
        this.$swal({ title: 'Grant Failed', text: msg, icon: 'error', confirmButtonText: 'OK' })
      } finally {
        this.grantingPort = null
      }
    },

    async loadContainers() {
      this.loading = true
      try {
        const [containersRes, infoRes] = await Promise.allSettled([
          api.getContainers(),
          api.getDockerInfo()
        ])

        if (containersRes.status === 'fulfilled') {
          this.dockerAvailable = true
          const raw = containersRes.value.data || []
          // Backend returns lowercase fields: id, name, image, state, status, ports
          this.containers = raw.map(c => ({
            id: c.id,
            name: c.name || c.id,
            image: c.image,
            status: c.state,      // 'running' | 'exited' | etc.
            statusText: c.status, // 'Up 20 hours' | etc.
            ports: c.ports || ''  // already a formatted string from backend
          }))
        } else {
          this.dockerAvailable = false
          this.containers = []
        }

        if (infoRes.status === 'fulfilled') {
          const info = infoRes.value.data || {}
          // Backend DockerInfo: available, server_version, containers_total,
          // containers_running, image_count — no raw images/volumes list.
          if (info.image_count != null) {
            this.dockerDisk = info.image_count + ' image(s)'
          }
        }
      } catch (err) {
        this.dockerAvailable = false
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to load Docker data', showConfirmButton: false, timer: 3000 })
      } finally {
        this.loading = false
      }
    },

    formatBytes(bytes) {
      if (bytes == null || isNaN(bytes) || bytes === 0) return bytes === 0 ? '0 B' : '—'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      const i = Math.floor(Math.log(bytes) / Math.log(1024))
      return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
    },

    formatDate(ts) {
      if (!ts) return '—'
      const d = new Date(typeof ts === 'number' ? ts * 1000 : ts)
      const days = Math.floor((Date.now() - d.getTime()) / 86400000)
      if (days === 0) return 'today'
      if (days === 1) return 'yesterday'
      return `${days} days ago`
    },

    async startStop(c) {
      try {
        if (c.status === 'running') {
          await api.stopContainer(c.id)
          this.$swal({ toast: true, position: 'top-end', icon: 'info', title: `Stopped ${c.name}`, showConfirmButton: false, timer: 2000 })
        } else {
          await api.startContainer(c.id)
          this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Started ${c.name}`, showConfirmButton: false, timer: 2000 })
        }
        await this.loadContainers()
      } catch (err) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: `Failed to ${c.status === 'running' ? 'stop' : 'start'} ${c.name}`, showConfirmButton: false, timer: 3000 })
      }
    },

    async restartContainer(c) {
      try {
        this.$swal({ toast: true, position: 'top-end', icon: 'info', title: `Restarting ${c.name}…`, showConfirmButton: false, timer: 2000 })
        await api.restartContainer(c.id)
        await this.loadContainers()
      } catch (err) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: `Failed to restart ${c.name}`, showConfirmButton: false, timer: 3000 })
      }
    },

    viewLogs(c) {
      const existing = this.containerLogs.find(modal => modal.container.id === c.id)
      if (existing) {
        this.bringModalFront(existing.modalId)
        existing.minimized = false
        return
      }

      const left = 80 + (this.containerLogs.length * 40) % 320
      const top = 80 + (this.containerLogs.length * 30) % 220
      this.containerLogs.push({
        modalId: `log-${this.nextLogModalId++}`,
        container: c,
        zIndex: ++this.maxModalZ,
        left,
        top,
        width: 880,
        height: 520,
        minimized: false,
        maximized: false
      })
    },
    bringModalFront(modalId) {
      const zIndex = ++this.maxModalZ
      this.containerLogs = this.containerLogs.map(modal => modal.modalId === modalId ? { ...modal, zIndex } : modal)
    },
    closeLogModal(modalId) {
      this.containerLogs = this.containerLogs.filter(modal => modal.modalId !== modalId)
    },
    updateLogModal(modal) {
      this.containerLogs = this.containerLogs.map(current => current.modalId === modal.modalId ? modal : current)
    },

    async runPrune(kind, label) {
      const confirm = await this.$swal({
        title: `Prune ${label}?`,
        text: 'This will remove unused Docker resources and cannot be undone.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Prune',
        confirmButtonColor: '#f04040'
      })
      if (!confirm.isConfirmed) return

      this.pruneBusy = true
      try {
        const { data } = await api.dockerPrune(kind)
        const reclaimed = this.formatBytes(Number(data.reclaimed_space || 0))
        this.$swal({
          toast: true,
          position: 'top-end',
          icon: 'success',
          title: `Pruned ${data.deleted || 0} item(s), reclaimed ${reclaimed}`,
          showConfirmButton: false,
          timer: 3500
        })
        await this.loadContainers()
      } catch (err) {
        this.$swal({
          toast: true,
          position: 'top-end',
          icon: 'error',
          title: err.response?.data?.error || `Failed to prune ${label.toLowerCase()}`,
          showConfirmButton: false,
          timer: 3500
        })
      } finally {
        this.pruneBusy = false
      }
    }
  }
}
</script>

<style scoped>
.sc-panel-card {
  border-radius: 12px;
}

.sc-view-containers :deep(.card-header) {
  padding: 0.85rem 1rem;
}

.sc-view-containers :deep(.card-body) {
  padding: 1rem;
}
</style>

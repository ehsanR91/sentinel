<template>
  <div class="sc-view sc-view-monitoring">
    <PageHeader title="Monitoring" icon="mdi mdi-chart-line" :items="[{text:'Monitoring',active:true,icon:'mdi mdi-chart-line'}]" />

    <!-- ── Resource gauges ──────────────────────────────────────────────────── -->
    <div class="row g-3 mb-4">
      <div class="col-xl-3 col-md-6">
        <div class="card sc-panel-card text-center py-3">
          <div style="font-size:0.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;color:var(--sc-text-muted);margin-bottom:.5rem">CPU</div>
          <MiniRadialGauge :value="cpu" color="#4a9eff" :size="160" />
          <div style="font-size:0.75rem;color:var(--sc-text-secondary);margin-top:-.5rem">{{ cores }} cores • Load {{ loadAvg }}</div>
        </div>
      </div>
      <div class="col-xl-3 col-md-6">
        <div class="card sc-panel-card text-center py-3">
          <div style="font-size:0.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;color:var(--sc-text-muted);margin-bottom:.5rem">Memory</div>
          <MiniRadialGauge :value="ram" color="#a78bfa" :size="160" />
          <div style="font-size:0.75rem;color:var(--sc-text-secondary);margin-top:-.5rem">{{ ramUsed }} / {{ ramTotal }}</div>
        </div>
      </div>
      <div class="col-xl-3 col-md-6">
        <div class="card sc-panel-card text-center py-3">
          <div style="font-size:0.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;color:var(--sc-text-muted);margin-bottom:.5rem">Disk /</div>
          <MiniRadialGauge :value="disk" :color="disk>85 ? '#f04040' : disk>65 ? '#f5a623' : '#22d67c'" :size="160" />
          <div style="font-size:0.75rem;color:var(--sc-text-secondary);margin-top:-.5rem">{{ diskUsed }} / {{ diskTotal }}</div>
        </div>
      </div>
      <div class="col-xl-3 col-md-6">
        <div class="card sc-panel-card text-center py-3">
          <div style="font-size:0.7rem;font-weight:700;text-transform:uppercase;letter-spacing:.06em;color:var(--sc-text-muted);margin-bottom:.5rem">Swap</div>
          <MiniRadialGauge :value="swap" color="#22d3ee" :size="160" />
          <div style="font-size:0.75rem;color:var(--sc-text-secondary);margin-top:-.5rem">{{ swapUsed }} / {{ swapTotal }}</div>
        </div>
      </div>
    </div>

    <!-- ── Full charts ─────────────────────────────────────────────────────── -->
    <div class="row g-3 mb-4">
      <div class="col-12">
        <div class="card sc-panel-card">
          <div class="card-header">
            <h6><i class="mdi mdi-chip me-2" style="color:#4a9eff"></i>CPU & Memory — Last 5 minutes</h6>
          </div>
          <div class="card-body py-2">
            <MiniTimeseriesChart :height="220" :series="timelineSeries" :percent-scale="true" />
          </div>
        </div>
      </div>
    </div>

    <!-- ── Disk partitions + Network ─────────────────────────────────────── -->
    <div class="row g-3 mb-4">
      <div class="col-12 col-xl-5">
        <div class="card sc-panel-card">
          <div class="card-header"><h6><i class="mdi mdi-harddisk me-2" style="color:#f5a623"></i>Disk Partitions</h6></div>
          <div class="card-body">
            <div v-for="part in partitions" :key="part.mount" class="mb-3">
              <div class="d-flex justify-content-between mb-1">
                <button class="btn btn-sm p-0 border-0 font-mono" style="font-size:0.78rem;color:#4a9eff;background:transparent" @click="openDiskUsage(part)">{{ part.mount }}</button>
                <span style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ part.used }} / {{ part.total }} ({{ part.pct }}%)</span>
              </div>
              <div class="progress">
                <div
                  class="progress-bar"
                  :class="part.pct>85?'bg-danger':part.pct>65?'bg-warning':'bg-success'"
                  :style="`width:${part.pct}%`"
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="col-12 col-xl-7">
        <div class="card sc-panel-card">
          <div class="card-header"><h6><i class="mdi mdi-swap-vertical me-2" style="color:#22d67c"></i>Network Interfaces</h6></div>
          <div class="card-body p-0">
            <table class="table mb-0">
              <thead>
                <tr>
                  <th>Interface</th><th>RX</th><th>TX</th><th>RX Total</th><th>TX Total</th><th>Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="iface in interfaces" :key="iface.name">
                  <td class="font-mono" style="font-size:0.78rem">{{ iface.name }}</td>
                  <td style="color:#22d67c;font-size:0.78rem">{{ iface.rx }}</td>
                  <td style="color:#4a9eff;font-size:0.78rem">{{ iface.tx }}</td>
                  <td style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ iface.rxTotal }}</td>
                  <td style="font-size:0.75rem;color:var(--sc-text-secondary)">{{ iface.txTotal }}</td>
                  <td><span class="status-dot" :class="iface.up ? 'online' : 'offline'"></span></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- ── Process list ──────────────────────────────────────────────────── -->
    <div class="card sc-panel-card mb-4">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-format-list-bulleted me-2" style="color:#a78bfa"></i>Top Processes</h6>
        <div class="d-flex gap-2">
          <input v-model="procFilter" class="form-control form-control-sm" placeholder="Filter…" style="width:160px" />
        </div>
      </div>
      <div class="card-body p-0" style="max-height:320px;overflow-y:auto">
        <table class="table mb-0">
          <thead>
            <tr>
              <th style="cursor:pointer" @click="sortProcess('pid')"><span style="font-size:0.72rem">PID</span> <i class="mdi" :class="sortIcon('pid')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('name')"><span style="font-size:0.72rem">Name</span> <i class="mdi" :class="sortIcon('name')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('user')"><span style="font-size:0.72rem">User</span> <i class="mdi" :class="sortIcon('user')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('cpu')"><span style="font-size:0.72rem">CPU %</span> <i class="mdi" :class="sortIcon('cpu')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('mem')"><span style="font-size:0.72rem">MEM %</span> <i class="mdi" :class="sortIcon('mem')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('rss')"><span style="font-size:0.72rem">MEM RSS</span> <i class="mdi" :class="sortIcon('rss')"></i></th>
              <th style="cursor:pointer" @click="sortProcess('status')"><span style="font-size:0.72rem">Status</span> <i class="mdi" :class="sortIcon('status')"></i></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in sortedFilteredProcs" :key="p.pid">
              <td class="font-mono" style="font-size:0.72rem;color:var(--sc-text-muted)">{{ p.pid }}</td>
              <td style="font-size:0.78rem;color:#4a9eff;cursor:pointer;text-decoration:underline" @click="showProcessModal(p)">{{ p.name }}</td>
              <td style="font-size:0.72rem;color:var(--sc-text-secondary)">{{ p.user }}</td>
              <td :style="`font-size:0.78rem;color:${p.cpu>50?'#f04040':p.cpu>20?'#f5a623':'#22d67c'}`">{{ p.cpu.toFixed(1) }}</td>
              <td style="font-size:0.78rem;color:var(--sc-text-secondary)">{{ p.mem.toFixed(1) }}</td>
              <td class="font-mono" style="font-size:0.72rem;color:var(--sc-text-secondary)">{{ p.rss }}</td>
              <td><span class="badge" style="font-size:0.62rem;background:rgba(34,214,124,0.12);color:#22d67c">{{ p.status }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ── Suspicious Process Detection ──────────────────────────────────── -->
    <div class="card sc-panel-card">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6>
          <i class="mdi mdi-shield-alert-outline me-2" style="color:#f04040"></i>
          Suspicious Process Detection
          <span v-if="suspiciousProcs.length > 0" class="badge rounded-pill ms-2" style="background:rgba(240,64,64,0.15);color:#f04040;font-size:0.62rem">
            {{ suspiciousProcs.length }} flagged
          </span>
        </h6>
        <button class="btn btn-sm" style="background:rgba(74,158,255,0.08);color:#4a9eff;font-size:0.72rem" @click="loadSuspicious" :disabled="loadingSuspicious">
          <i :class="`mdi ${loadingSuspicious ? 'mdi-loading mdi-spin' : 'mdi-refresh'} me-1`"></i>Scan
        </button>
      </div>
      <div class="card-body p-0">
        <div v-if="loadingSuspicious" class="text-center py-4" style="color:var(--sc-text-muted)">
          <i class="mdi mdi-loading mdi-spin me-2"></i>Scanning processes…
        </div>
        <div v-else-if="suspiciousProcs.length === 0" class="text-center py-4" style="color:#22d67c;font-size:0.82rem">
          <i class="mdi mdi-shield-check me-2"></i>No suspicious processes detected
        </div>
        <table v-else class="table mb-0">
          <thead>
            <tr><th>Risk</th><th>PID</th><th>Name</th><th>User</th><th>Reason</th><th>Command</th></tr>
          </thead>
          <tbody>
            <tr v-for="p in suspiciousProcs" :key="p.pid">
              <td>
                <span class="badge rounded-pill" :style="riskStyle(p.risk)">{{ p.risk }}</span>
              </td>
              <td class="font-mono" style="font-size:0.72rem;color:var(--sc-text-muted)">{{ p.pid }}</td>
              <td style="font-size:0.78rem;color:var(--sc-text);font-weight:600">{{ p.name }}</td>
              <td style="font-size:0.72rem;color:var(--sc-text-secondary)">{{ p.user }}</td>
              <td style="font-size:0.75rem;color:#f5a623">{{ p.reason }}</td>
              <td class="font-mono" style="font-size:0.68rem;color:var(--sc-text-muted);max-width:240px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
                <Tooltip :label="p.name" :description="p.cmd" variant="rich" as-child>
                  <span class="d-inline-block w-100 text-truncate">{{ p.cmd }}</span>
                </Tooltip>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="showDiskModal" class="sc-modal-overlay" @click.self="showDiskModal=false">
      <div class="sc-modal-card">
        <div class="d-flex align-items-center justify-content-between mb-2">
          <h6 class="mb-0"><i class="mdi mdi-file-tree me-2" style="color:#4a9eff"></i>Disk Usage Drilldown: {{ diskUsagePath }}</h6>
          <button class="btn btn-sm" style="background:rgba(240,64,64,0.12);color:#f04040" @click="showDiskModal=false">
            <i class="mdi mdi-close"></i>
          </button>
        </div>
        <div v-if="diskUsageLoading" class="text-center py-4" style="color:var(--sc-text-muted)">
          <i class="mdi mdi-loading mdi-spin me-2"></i>Analyzing disk usage…
        </div>
        <div v-else-if="diskUsageError" class="alert alert-danger mb-0">{{ diskUsageError }}</div>
        <div v-else>
          <div class="d-flex align-items-center justify-content-between mb-2" style="font-size:0.75rem;color:var(--sc-text-muted)">
            <span>Total: {{ diskUsageTotalHuman }}</span>
            <span>Top {{ diskUsageItems.length }} entries</span>
          </div>
          <div style="max-height:360px;overflow:auto;border:1px solid var(--sc-border);border-radius:8px">
            <table class="table mb-0">
              <thead>
                <tr><th>Path</th><th style="width:140px">Size</th><th style="width:100px">Share</th></tr>
              </thead>
              <tbody>
                <tr v-for="it in diskUsageItems" :key="it.path">
                  <td>
                    <div :style="`padding-left:${Math.max(0, (it.depth - 1) * 12)}px`" class="font-mono" style="font-size:0.72rem;color:var(--sc-text)">
                      {{ it.path }}
                    </div>
                  </td>
                  <td class="font-mono" style="font-size:0.72rem;color:var(--sc-text-secondary)">{{ it.size_human }}</td>
                  <td style="font-size:0.72rem;color:var(--sc-text-muted)">{{ usageShare(it.size) }}</td>
                </tr>
                <tr v-if="diskUsageItems.length === 0">
                  <td colspan="3" class="text-center py-3" style="color:var(--sc-text-muted)">No data available</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Process Details Modal -->
    <div v-if="procModal.show" class="modal fade show d-block" tabindex="-1" style="background-color:rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Process Details: {{ procModal.proc?.name }}</h5>
            <button type="button" class="btn-close btn-close-white" @click="procModal.show = false"></button>
          </div>
          <div class="modal-body">
            <div v-if="procModal.loading" class="text-center py-3">
              <i class="mdi mdi-loading mdi-spin me-2"></i>Loading process details…
            </div>
            <div v-else>
              <div class="row g-3">
                <div class="col-md-6">
                  <div class="mb-3"><strong>PID:</strong> {{ procModal.proc?.pid }}</div>
                  <div class="mb-3"><strong>User:</strong> {{ procModal.proc?.user }}</div>
                  <div class="mb-3"><strong>Status:</strong> <span class="badge" style="background:rgba(34,214,124,0.12);color:#22d67c">{{ procModal.proc?.status }}</span></div>
                  <div class="mb-3"><strong>CPU %:</strong> {{ procModal.proc?.cpu?.toFixed(1) }}%</div>
                  <div class="mb-3"><strong>MEM %:</strong> {{ procModal.proc?.mem?.toFixed(1) }}%</div>
                  <div class="mb-3"><strong>MEM RSS:</strong> {{ procModal.proc?.rss }}</div>
                </div>
                <div class="col-md-6">
                  <div class="mb-3"><strong>Executable Path:</strong> <code style="font-size:0.72rem">{{ procModal.details?.exe || '—' }}</code></div>
                  <div class="mb-3"><strong>Parent PID:</strong> {{ procModal.details?.ppid || '—' }}</div>
                  <div class="mb-3"><strong>Start Time:</strong> {{ procModal.details?.start || '—' }}</div>
                  <div class="mb-3"><strong>Threads:</strong> {{ procModal.details?.threads || '—' }}</div>
                  <div class="mb-3"><strong>Command Line:</strong> <code style="font-size:0.72rem;word-break:break-all">{{ procModal.details?.cmdline || '—' }}</code></div>
                </div>
              </div>
              <div class="mt-3">
                <strong>Description:</strong>
                <p class="mt-1">{{ procModal.description || 'No description available.' }}</p>
              </div>
              <div v-if="procModal.links && procModal.links.length" class="mt-3">
                <strong>References:</strong>
                <ul class="mt-1">
                  <li v-for="(link, i) in procModal.links" :key="i">
                    <a :href="link.url" target="_blank" rel="noopener" style="color:#4a9eff">{{ link.title }}</a>
                  </li>
                </ul>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn btn-secondary" @click="procModal.show = false">Close</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { useDocumentVisibility } from '@vueuse/core'
import { useMetricsStore } from '@/stores/metrics'
import PageHeader from '@/components/page-header.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import MiniRadialGauge from '@/components/ui/mini-radial-gauge.vue'
import MiniTimeseriesChart from '@/components/ui/mini-timeseries-chart.vue'
import api from '@/services/api'

function fmtBytes (b) {
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  if (b >= 1048576)    return (b / 1048576).toFixed(1) + ' MB'
  if (b >= 1024)       return (b / 1024).toFixed(0) + ' KB'
  return b + ' B'
}

export default {
  name: 'MonitoringPage',
  components: { PageHeader, Tooltip, MiniRadialGauge, MiniTimeseriesChart },
  setup() {
    return {
      documentVisibility: useDocumentVisibility(),
      metricsStore: useMetricsStore()
    }
  },

  data() {
    return {
      procFilter: '',
      procSortKey: 'cpu',
      procSortDir: 'desc',
      procTimer: null,
      svcTimer: null,
      suspiciousProcs: [],
      loadingSuspicious: false,
      showDiskModal: false,
      diskUsagePath: '/',
      diskUsageItems: [],
      diskUsageTotal: 0,
      diskUsageTotalHuman: '0 B',
      diskUsageLoading: false,
      diskUsageError: '',
      procModal: { show: false, proc: null, loading: false, details: {}, description: '', links: [] }
    }
  },

  computed: {
    snap() { return this.metricsStore.snap },
    cpuHistory() { return this.metricsStore.cpuHistory },
    ramHistory() { return this.metricsStore.ramHistory },
    processes() { return this.metricsStore.processes },
    services() { return this.metricsStore.services },

    cpu()       { return this.snap.cpu_pct },
    ram()       { return this.snap.ram_pct },
    disk()      { return this.snap.disk_pct },
    swap()      { return this.snap.swap_pct },
    cores()     { return (this.snap.cpu_cores || []).length },
    loadAvg()   { return this.snap.load1.toFixed(2) },
    ramUsed()   { return fmtBytes(this.snap.ram_used) },
    ramTotal()  { return fmtBytes(this.snap.ram_total) },
    diskUsed()  { return fmtBytes(this.snap.disk_used) },
    diskTotal() { return fmtBytes(this.snap.disk_total) },
    swapUsed()  { return fmtBytes(this.snap.swap_used) },
    swapTotal() { return fmtBytes(this.snap.swap_total) },

    partitions() {
      return (this.snap.partitions || []).map(p => ({
        mount: p.mount,
        used:  fmtBytes(p.used),
        total: fmtBytes(p.total),
        pct:   p.pct
      }))
    },

    interfaces() {
      return [{
        name:    'aggregate',
        rx:      fmtBytes(this.snap.net_rx_rate) + '/s',
        tx:      fmtBytes(this.snap.net_tx_rate) + '/s',
        rxTotal: fmtBytes(this.snap.net_rx_total),
        txTotal: fmtBytes(this.snap.net_tx_total),
        up:      true
      }]
    },

    filteredProcs() {
      const procs = (this.processes || []).map(p => ({
        pid:    p.pid,
        name:   p.name,
        user:   p.user,
        cpu:    p.cpu_pct,
        mem:    p.mem_pct,
        rss:    fmtBytes(p.mem_rss),
        status: p.status
      }))
      if (!this.procFilter) return procs
      const f = this.procFilter.toLowerCase()
      return procs.filter(p =>
        p.name.toLowerCase().includes(f) ||
        String(p.pid).includes(f) ||
        p.user.toLowerCase().includes(f)
      )
    },

    sortedFilteredProcs() {
      const list = [...this.filteredProcs]
      const key = this.procSortKey
      const dir = this.procSortDir === 'asc' ? 1 : -1
      list.sort((a, b) => {
        let aVal = a[key]
        let bVal = b[key]
        if (key === 'rss') {
          aVal = parseInt(a.rss) || 0
          bVal = parseInt(b.rss) || 0
        }
        if (aVal < bVal) return -1 * dir
        if (aVal > bVal) return 1 * dir
        return 0
      })
      return list
    },

    timelineSeries() {
      return [
        { name: 'CPU %', data: this.cpuHistory, color: '#4a9eff' },
        { name: 'RAM %', data: this.ramHistory, color: '#a78bfa' }
      ]
    }
  },

  watch: {
    documentVisibility(value) {
      if (value === 'visible') {
        this.metricsStore.fetchProcesses()
        this.metricsStore.fetchServices()
        this.loadSuspicious()
      }
    }
  },

  async mounted() {
    this.metricsStore.startLive()
    await this.metricsStore.fetchProcesses()
    await this.metricsStore.fetchServices()
    this.loadSuspicious()

    this.procTimer = setInterval(() => {
      if (this.documentVisibility !== 'visible') return
      this.metricsStore.fetchProcesses()
    }, 10000)
  },

  beforeUnmount() {
    clearInterval(this.procTimer)
  },

  methods: {
    async showProcessModal(proc) {
      this.procModal = { show: true, proc, loading: false, details: {}, description: '', links: [] }
      // Stub: populate description and links based on common process names
      const name = (proc.name || '').toLowerCase()
      const info = this.getProcessInfo(name)
      this.procModal.description = info.description
      this.procModal.links = info.links
    },
    getProcessInfo(name) {
      const catalog = {
        systemd: { description: 'System and service manager for Linux.', links: [{ title: 'systemd documentation', url: 'https://systemd.io/' }] },
        cron: { description: 'Time-based job scheduler in Unix-like operating systems.', links: [{ title: 'cron manual', url: 'https://man7.org/linux/man-pages/man8/cron.8.html' }] },
        sshd: { description: 'OpenSSH daemon for secure remote login.', links: [{ title: 'OpenSSH', url: 'https://www.openssh.com/' }] },
        nginx: { description: 'High-performance HTTP and reverse proxy server.', links: [{ title: 'nginx.org', url: 'https://nginx.org/' }] },
        apache2: { description: 'Apache HTTP Server.', links: [{ title: 'httpd.apache.org', url: 'https://httpd.apache.org/' }] },
        mysql: { description: 'Popular open-source relational database.', links: [{ title: 'mysql.com', url: 'https://www.mysql.com/' }] },
        postgres: { description: 'Advanced open-source relational database.', links: [{ title: 'postgresql.org', url: 'https://www.postgresql.org/' }] },
        redis: { description: 'In-memory data structure store, used as a database, cache, and message broker.', links: [{ title: 'redis.io', url: 'https://redis.io/' }] },
        docker: { description: 'Platform for developing, shipping, and running applications in containers.', links: [{ title: 'docker.com', url: 'https://www.docker.com/' }] },
        containerd: { description: 'Industry-standard container runtime.', links: [{ title: 'containerd.io', url: 'https://containerd.io/' }] },
        kubelet: { description: 'Node agent that runs pods in Kubernetes.', links: [{ title: 'kubernetes.io', url: 'https://kubernetes.io/' }] },
        rsyslog: { description: 'Rocket-fast system for log processing.', links: [{ title: 'rsyslog.com', url: 'https://www.rsyslog.com/' }] },
        journald: { description: 'Systemd logging service.', links: [{ title: 'systemd.journald', url: 'https://www.freedesktop.org/software/systemd/man/journald.conf.html' }] },
        fail2ban: { description: 'Daemon to ban hostile IP addresses.', links: [{ title: 'fail2ban.org', url: 'https://www.fail2ban.org/' }] },
        ufw: { description: 'Uncomplicated Firewall program for managing Linux firewall rules.', links: [{ title: 'ufw - Ubuntu Wiki', url: 'https://help.ubuntu.com/community/UFW' }] },
        iptables: { description: 'User-space utility program for configuring Linux firewall.', links: [{ title: 'iptables.org', url: 'https://www.iptables.org/' }] },
        networkmanager: { description: 'Network management daemon for Linux.', links: [{ title: 'networkmanager.dev', url: 'https://networkmanager.dev/' }] },
        dhclient: { description: 'Dynamic Host Configuration Protocol client.', links: [{ title: 'ISC DHCP', url: 'https://www.isc.org/dhcp/' }] },
        cups: { description: 'Modular printing system for Unix-like OS.', links: [{ title: 'cups.org', url: 'https://www.cups.org/' }] },
        avahi: { description: 'Service discovery for local networks.', links: [{ title: 'avahi.org', url: 'https://www.avahi.org/' }] },
        gnome: { description: 'GNOME desktop environment components.', links: [{ title: 'gnome.org', url: 'https://www.gnome.org/' }] },
        kde: { description: 'KDE desktop environment components.', links: [{ title: 'kde.org', url: 'https://kde.org/' }] },
        xorg: { description: 'X.Org X Window System server.', links: [{ title: 'x.org', url: 'https://www.x.org/' }] },
        pulseaudio: { description: 'Sound server for POSIX OS.', links: [{ title: 'pulseaudio.org', url: 'https://www.pulseaudio.org/' }] },
        pipewire: { description: 'Server that handles multimedia pipelines.', links: [{ title: 'pipewire.org', url: 'https://pipewire.org/' }] },
        bluetoothd: { description: 'Bluetooth daemon for Linux.', links: [{ title: 'BlueZ', url: 'https://www.bluez.org/' }] },
        accounts: { description: 'AccountsService daemon for user account management.', links: [{ title: 'freedesktop.org AccountsService', url: 'https://www.freedesktop.org/software/AccountsService/' }] },
        polkit: { description: 'PolicyKit authorization framework.', links: [{ title: 'polkit.freedesktop.org', url: 'https://www.freedesktop.org/software/polkit/' }] },
        gdm: { description: 'GNOME Display Manager.', links: [{ title: 'gnome.org', url: 'https://wiki.gnome.org/Projects/GDM' }] },
        sddm: { description: 'Simple Desktop Display Manager.', links: [{ title: 'sddm', url: 'https://github.com/sddm/sddm' }] },
        lightdm: { description: 'Cross-desktop display manager.', links: [{ title: 'canonical.com LightDM', url: 'https://www.canonical.com/projects/lightdm' }] },
        apparmor: { description: 'Mandatory Access Control (MAC) system.', links: [{ title: 'apparmor.net', url: 'https://apparmor.net/' }] },
        selinux: { description: 'Security-Enhanced Linux.', links: [{ title: 'selinuxproject.org', url: 'https://www.selinuxproject.org/' }] }
      }
      return catalog[name] || { description: '', links: [] }
    },
    async loadSuspicious() {
      this.loadingSuspicious = true
      try {
        const { data } = await api.getSuspiciousProcesses()
        this.suspiciousProcs = Array.isArray(data) ? data : []
      } catch (_) {
        this.suspiciousProcs = []
      } finally {
        this.loadingSuspicious = false
      }
    },

    riskStyle(risk) {
      const map = {
        high:   'background:rgba(240,64,64,0.15);color:#f04040',
        medium: 'background:rgba(245,166,35,0.15);color:#f5a623',
        low:    'background:rgba(34,214,124,0.12);color:#22d67c',
      }
      return (map[risk] || map.low) + ';font-size:0.6rem;padding:2px 6px'
    },

    usageShare(size) {
      if (!this.diskUsageTotal || !size) return '0%'
      return `${((size / this.diskUsageTotal) * 100).toFixed(1)}%`
    },

    async openDiskUsage(part) {
      this.showDiskModal = true
      this.diskUsagePath = part.mount
      this.diskUsageItems = []
      this.diskUsageError = ''
      this.diskUsageLoading = true
      try {
        const { data } = await api.getDiskUsage(part.mount, 2, 30)
        this.diskUsageItems = Array.isArray(data.items) ? data.items : []
        this.diskUsageTotal = Number(data.total_size || 0)
        this.diskUsageTotalHuman = data.total_human || '0 B'
      } catch (e) {
        this.diskUsageError = e.response?.data?.error || 'Failed to inspect disk usage'
      } finally {
        this.diskUsageLoading = false
      }
    },

    sortProcess(key) {
      if (this.procSortKey === key) {
        this.procSortDir = this.procSortDir === 'asc' ? 'desc' : 'asc'
      } else {
        this.procSortKey = key
        this.procSortDir = 'desc'
      }
    },

    sortIcon(key) {
      if (this.procSortKey !== key) return 'mdi-sort'
      return this.procSortDir === 'asc' ? 'mdi-sort-ascending' : 'mdi-sort-descending'
    }
  }
}
</script>

<style scoped>
.sc-panel-card {
  border-radius: 12px;
}

.sc-view-monitoring :deep(.card-header) {
  padding: 0.85rem 1rem;
}

.sc-view-monitoring :deep(.card-body) {
  padding: 1rem;
}

.sc-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1600;
  padding: 1rem;
}

.sc-modal-card {
  width: min(900px, 96vw);
  max-height: 85vh;
  overflow: auto;
  background: var(--sc-bg-card);
  border: 1px solid var(--sc-border);
  border-radius: 12px;
  padding: 1rem;
}
</style>

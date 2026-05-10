<template>
  <div class="sc-view sc-view-monitoring">
    <PageHeader title="Monitoring" icon="mdi mdi-chart-line" :items="[{text:'Monitoring',active:true,icon:'mdi mdi-chart-line'}]">
      <template #actions>
        <button class="wgt-reset-btn sc-focus-ring" title="Reset to default layout" @click="resetLayout">
          <i class="mdi mdi-view-grid-outline"></i>
          Reset Layout
        </button>
      </template>
    </PageHeader>

    <draggable
      v-model="widgets"
      item-key="id"
      handle=".wgt-drag"
      class="mon-widget-grid"
      :animation="200"
      ghost-class="wgt-ghost"
      @end="saveLayout"
    >
      <template #item="{ element: w }">
        <div class="wgt-card" :class="[`wgt-span-${w.span}`, { 'wgt-collapsed': w.collapsed }]">

          <!-- header -->
          <div class="wgt-header">
            <span class="wgt-drag" title="Drag to reorder"><i class="mdi mdi-drag-vertical"></i></span>
            <i :class="`mdi ${w.icon} wgt-icon`" :style="`color:${w.iconColor}`"></i>
            <span class="wgt-title">{{ w.title }}</span>
            <span v-if="w.id === 'suspicious' && suspiciousProcs.length" class="mon-badge-danger">{{ suspiciousProcs.length }}</span>
            <div class="wgt-actions">
              <Tooltip :label="w.span===2?'Switch to half width':'Switch to full width'" placement="top" :delay="600" as-child>
                <button class="wgt-btn" @click="toggleSpan(w)">
                  <i :class="`mdi ${w.span===2?'mdi-arrow-collapse-horizontal':'mdi-arrow-expand-horizontal'}`"></i>
                </button>
              </Tooltip>
              <Tooltip :label="w.collapsed?'Expand widget':'Collapse widget'" placement="top" :delay="600" as-child>
                <button class="wgt-btn" @click="toggleCollapse(w)">
                  <i :class="`mdi ${w.collapsed?'mdi-chevron-down':'mdi-chevron-up'}`"></i>
                </button>
              </Tooltip>
            </div>
          </div>

          <!-- body (collapse via max-height transition) -->
          <div class="wgt-body">

            <!-- ── resources ── -->
            <template v-if="w.id === 'resources'">
              <div class="wgt-gauge-row">
                <div class="wgt-gauge-cell">
                  <MiniRadialGauge :value="cpu" color="#4a9eff" :size="72" class="mon-gauge" />
                  <div class="mon-gauge-info">
                    <span class="mon-gauge-label">CPU</span>
                    <span class="mon-gauge-value" style="color:#4a9eff">{{ cpu }}%</span>
                    <span class="mon-gauge-sub">{{ cores }} cores · Load {{ loadAvg }}</span>
                  </div>
                </div>
                <div class="wgt-gauge-cell">
                  <MiniRadialGauge :value="ram" color="#a78bfa" :size="72" class="mon-gauge" />
                  <div class="mon-gauge-info">
                    <span class="mon-gauge-label">Memory</span>
                    <span class="mon-gauge-value" style="color:#a78bfa">{{ ram }}%</span>
                    <span class="mon-gauge-sub">{{ ramUsed }} / {{ ramTotal }}</span>
                  </div>
                </div>
                <div class="wgt-gauge-cell">
                  <MiniRadialGauge :value="disk" :color="disk>85?'#f04040':disk>65?'#f5a623':'#22d67c'" :size="72" class="mon-gauge" />
                  <div class="mon-gauge-info">
                    <span class="mon-gauge-label">Disk /</span>
                    <span class="mon-gauge-value" :style="`color:${disk>85?'#f04040':disk>65?'#f5a623':'#22d67c'}`">{{ disk }}%</span>
                    <span class="mon-gauge-sub">{{ diskUsed }} / {{ diskTotal }}</span>
                  </div>
                </div>
                <div class="wgt-gauge-cell">
                  <MiniRadialGauge :value="swap" color="#22d3ee" :size="72" class="mon-gauge" />
                  <div class="mon-gauge-info">
                    <span class="mon-gauge-label">Swap</span>
                    <span class="mon-gauge-value" style="color:#22d3ee">{{ swap }}%</span>
                    <span class="mon-gauge-sub">{{ swapUsed }} / {{ swapTotal }}</span>
                  </div>
                </div>
                <div class="wgt-gauge-cell wgt-gauge-cell--net">
                  <div class="mon-net-row">
                    <i class="mdi mdi-arrow-down" style="color:#22d67c"></i>
                    <span class="mon-net-val" style="color:#22d67c">{{ netRx }}</span>
                    <span class="mon-net-lbl">RX/s</span>
                  </div>
                  <div class="mon-net-row">
                    <i class="mdi mdi-arrow-up" style="color:#4a9eff"></i>
                    <span class="mon-net-val" style="color:#4a9eff">{{ netTx }}</span>
                    <span class="mon-net-lbl">TX/s</span>
                  </div>
                  <span class="mon-gauge-sub" style="margin-top:4px">Network I/O</span>
                </div>
              </div>
            </template>

            <!-- ── chart ── -->
            <template v-else-if="w.id === 'chart'">
              <div class="wgt-chart-body">
                <MiniTimeseriesChart :height="180" :series="timelineSeries" :percent-scale="true" />
              </div>
            </template>

            <!-- ── disk ── -->
            <template v-else-if="w.id === 'disk'">
              <div class="mon-scroll-body">
                <div v-for="part in partitions" :key="part.mount" class="mon-disk-row">
                  <div class="mon-disk-row__top">
                    <button class="mon-mount-btn" @click="openDiskUsage(part)">{{ part.mount }}</button>
                    <span class="mon-disk-row__size">{{ part.used }} / {{ part.total }} ({{ part.pct }}%)</span>
                  </div>
                  <div class="mon-progress">
                    <div class="mon-progress__bar" :class="part.pct>85?'is-danger':part.pct>65?'is-warn':'is-ok'" :style="`width:${part.pct}%`"></div>
                  </div>
                </div>
              </div>
            </template>

            <!-- ── network ── -->
            <template v-else-if="w.id === 'network'">
              <table class="mon-table">
                <thead>
                  <tr><th>Interface</th><th>RX</th><th>TX</th><th>RX Total</th><th>TX Total</th><th></th></tr>
                </thead>
                <tbody>
                  <tr v-for="iface in interfaces" :key="iface.name">
                    <td class="mon-mono">{{ iface.name }}</td>
                    <td style="color:#22d67c">{{ iface.rx }}</td>
                    <td style="color:#4a9eff">{{ iface.tx }}</td>
                    <td class="mon-muted">{{ iface.rxTotal }}</td>
                    <td class="mon-muted">{{ iface.txTotal }}</td>
                    <td><span class="mon-dot" :class="iface.up?'is-up':'is-down'"></span></td>
                  </tr>
                </tbody>
              </table>
            </template>

            <!-- ── processes ── -->
            <template v-else-if="w.id === 'processes'">
              <div class="wgt-filter-bar">
                <input v-model="procFilter" class="mon-filter-input sc-focus-ring" placeholder="Filter processes…" />
              </div>
              <div class="mon-scroll-body">
                <table class="mon-table">
                  <thead>
                    <tr>
                      <th class="mon-sortable" @click="sortProcess('pid')">PID <i class="mdi" :class="sortIcon('pid')"></i></th>
                      <th class="mon-sortable" @click="sortProcess('name')">Name <i class="mdi" :class="sortIcon('name')"></i></th>
                      <th class="mon-sortable" @click="sortProcess('user')">User <i class="mdi" :class="sortIcon('user')"></i></th>
                      <th class="mon-sortable" @click="sortProcess('cpu')">CPU% <i class="mdi" :class="sortIcon('cpu')"></i></th>
                      <th class="mon-sortable" @click="sortProcess('mem')">MEM% <i class="mdi" :class="sortIcon('mem')"></i></th>
                      <th class="mon-sortable" @click="sortProcess('rss')">RSS <i class="mdi" :class="sortIcon('rss')"></i></th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="p in sortedFilteredProcs" :key="p.pid">
                      <td class="mon-mono mon-muted">{{ p.pid }}</td>
                      <td class="mon-proc-name" @click="showProcessModal(p)">{{ p.name }}</td>
                      <td class="mon-muted">{{ p.user }}</td>
                      <td :style="`color:${p.cpu>50?'#f04040':p.cpu>20?'#f5a623':'#22d67c'}`">{{ p.cpu.toFixed(1) }}</td>
                      <td class="mon-muted">{{ p.mem.toFixed(1) }}</td>
                      <td class="mon-mono mon-muted">{{ p.rss }}</td>
                      <td><span class="mon-status-badge">{{ p.status }}</span></td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>

            <!-- ── suspicious ── -->
            <template v-else-if="w.id === 'suspicious'">
              <div class="wgt-filter-bar">
                <button class="mon-scan-btn sc-focus-ring" :disabled="loadingSuspicious" @click="loadSuspicious">
                  <i :class="`mdi ${loadingSuspicious?'mdi-loading mdi-spin':'mdi-refresh'}`"></i>
                  Scan now
                </button>
              </div>
              <div v-if="loadingSuspicious" class="mon-empty"><i class="mdi mdi-loading mdi-spin"></i> Scanning…</div>
              <div v-else-if="!suspiciousProcs.length" class="mon-empty mon-empty--ok">
                <i class="mdi mdi-shield-check"></i> No suspicious processes detected
              </div>
              <div v-else class="mon-scroll-body">
                <table class="mon-table">
                  <thead>
                    <tr><th>Risk</th><th>PID</th><th>Name</th><th>User</th><th>Reason</th><th>Command</th></tr>
                  </thead>
                  <tbody>
                    <tr v-for="p in suspiciousProcs" :key="p.pid">
                      <td><span class="mon-risk-badge" :class="`is-${p.risk}`">{{ p.risk }}</span></td>
                      <td class="mon-mono mon-muted">{{ p.pid }}</td>
                      <td class="mon-bold">{{ p.name }}</td>
                      <td class="mon-muted">{{ p.user }}</td>
                      <td style="color:#f5a623">{{ p.reason }}</td>
                      <td class="mon-mono mon-muted mon-cmd">
                        <Tooltip :label="p.name" :description="p.cmd" variant="rich" as-child>
                          <span>{{ p.cmd }}</span>
                        </Tooltip>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </template>

          </div>
        </div>
      </template>
    </draggable>

    <!-- Disk usage drilldown modal -->
    <div v-if="showDiskModal" class="sc-modal-overlay" @click.self="showDiskModal=false">
      <div class="sc-modal-card">
        <div class="sc-modal-head">
          <span><i class="mdi mdi-file-tree" style="color:#4a9eff"></i> Disk Usage: {{ diskUsagePath }}</span>
          <button class="mon-close-btn" @click="showDiskModal=false"><i class="mdi mdi-close"></i></button>
        </div>
        <div v-if="diskUsageLoading" class="mon-empty"><i class="mdi mdi-loading mdi-spin"></i> Analyzing…</div>
        <div v-else-if="diskUsageError" class="alert alert-danger mb-0">{{ diskUsageError }}</div>
        <div v-else>
          <div class="sc-modal-meta">
            <span>Total: {{ diskUsageTotalHuman }}</span>
            <span>Top {{ diskUsageItems.length }} entries</span>
          </div>
          <div class="sc-modal-scroll">
            <table class="mon-table">
              <thead><tr><th>Path</th><th>Size</th><th>Share</th></tr></thead>
              <tbody>
                <tr v-for="it in diskUsageItems" :key="it.path">
                  <td class="mon-mono" :style="`padding-left:${Math.max(0,(it.depth-1)*12)}px`">{{ it.path }}</td>
                  <td class="mon-mono mon-muted">{{ it.size_human }}</td>
                  <td class="mon-muted">{{ usageShare(it.size) }}</td>
                </tr>
                <tr v-if="!diskUsageItems.length"><td colspan="3" class="mon-empty">No data</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Process details modal -->
    <div v-if="procModal.show" class="sc-modal-overlay" @click.self="procModal.show=false">
      <div class="sc-modal-card">
        <div class="sc-modal-head">
          <span><i class="mdi mdi-cpu-64-bit" style="color:#4a9eff"></i> Process: {{ procModal.proc?.name }}</span>
          <button class="mon-close-btn" @click="procModal.show=false"><i class="mdi mdi-close"></i></button>
        </div>
        <div v-if="procModal.loading" class="mon-empty"><i class="mdi mdi-loading mdi-spin"></i> Loading…</div>
        <div v-else class="sc-modal-grid">
          <div>
            <div class="sc-modal-row"><span>PID</span><span>{{ procModal.proc?.pid }}</span></div>
            <div class="sc-modal-row"><span>User</span><span>{{ procModal.proc?.user }}</span></div>
            <div class="sc-modal-row"><span>Status</span><span class="mon-status-badge">{{ procModal.proc?.status }}</span></div>
            <div class="sc-modal-row"><span>CPU</span><span>{{ procModal.proc?.cpu?.toFixed(1) }}%</span></div>
            <div class="sc-modal-row"><span>MEM</span><span>{{ procModal.proc?.mem?.toFixed(1) }}%</span></div>
            <div class="sc-modal-row"><span>RSS</span><span>{{ procModal.proc?.rss }}</span></div>
          </div>
          <div>
            <div class="sc-modal-row"><span>Executable</span><code class="mon-mono">{{ procModal.details?.exe || '—' }}</code></div>
            <div class="sc-modal-row"><span>Parent PID</span><span>{{ procModal.details?.ppid || '—' }}</span></div>
            <div class="sc-modal-row"><span>Started</span><span>{{ procModal.details?.start || '—' }}</span></div>
            <div class="sc-modal-row"><span>Threads</span><span>{{ procModal.details?.threads || '—' }}</span></div>
            <div class="sc-modal-row"><span>Cmdline</span><code class="mon-mono">{{ procModal.details?.cmdline || '—' }}</code></div>
          </div>
          <div v-if="procModal.description" class="sc-modal-desc">
            <strong>Description</strong>
            <p>{{ procModal.description }}</p>
          </div>
          <div v-if="procModal.links?.length" class="sc-modal-desc">
            <strong>References</strong>
            <ul>
              <li v-for="(link, i) in procModal.links" :key="i">
                <a :href="link.url" target="_blank" rel="noopener" style="color:#4a9eff">{{ link.title }}</a>
              </li>
            </ul>
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
import draggable from 'vuedraggable'

const DEFAULT_WIDGETS = [
  { id: 'resources',  title: 'System Resources',      icon: 'mdi-gauge',                iconColor: '#4a9eff', collapsed: false, span: 2 },
  { id: 'chart',      title: 'CPU & Memory Trend',    icon: 'mdi-chart-areaspline',     iconColor: '#4a9eff', collapsed: false, span: 2 },
  { id: 'disk',       title: 'Disk Partitions',       icon: 'mdi-harddisk',             iconColor: '#f5a623', collapsed: false, span: 1 },
  { id: 'network',    title: 'Network Interfaces',    icon: 'mdi-swap-vertical',        iconColor: '#22d67c', collapsed: false, span: 1 },
  { id: 'processes',  title: 'Top Processes',         icon: 'mdi-format-list-bulleted', iconColor: '#a78bfa', collapsed: false, span: 2 },
  { id: 'suspicious', title: 'Suspicious Processes',  icon: 'mdi-shield-alert-outline', iconColor: '#f04040', collapsed: false, span: 2 }
]

function fmtBytes (b) {
  if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB'
  if (b >= 1048576)    return (b / 1048576).toFixed(1) + ' MB'
  if (b >= 1024)       return (b / 1024).toFixed(0) + ' KB'
  return b + ' B'
}

export default {
  name: 'MonitoringPage',
  components: { PageHeader, Tooltip, MiniRadialGauge, MiniTimeseriesChart, draggable },
  setup() {
    return {
      documentVisibility: useDocumentVisibility(),
      metricsStore: useMetricsStore()
    }
  },

  data() {
    return {
      widgets: DEFAULT_WIDGETS.map(w => ({ ...w })),
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

    netRx() { return fmtBytes(this.snap.net_rx_rate) + '/s' },
    netTx() { return fmtBytes(this.snap.net_tx_rate) + '/s' },

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
    this.loadLayout()
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
      // kept for backward compat but badge class now handles this
      return ''
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
    },

    toggleSpan(w) {
      w.span = w.span === 2 ? 1 : 2
      this.saveLayout()
    },

    toggleCollapse(w) {
      w.collapsed = !w.collapsed
      this.saveLayout()
    },

    saveLayout() {
      try {
        localStorage.setItem('mon-widget-layout', JSON.stringify(
          this.widgets.map(w => ({ id: w.id, collapsed: w.collapsed, span: w.span }))
        ))
      } catch (_) {}
    },

    loadLayout() {
      try {
        const saved = JSON.parse(localStorage.getItem('mon-widget-layout') || '[]')
        if (!saved.length) return
        const stateMap = Object.fromEntries(saved.map((s, i) => [s.id, { ...s, _order: i }]))
        this.widgets = this.widgets
          .sort((a, b) => {
            const oa = stateMap[a.id]?._order ?? 99
            const ob = stateMap[b.id]?._order ?? 99
            return oa - ob
          })
          .map(w => {
            const s = stateMap[w.id]
            if (!s) return w
            return { ...w, collapsed: s.collapsed ?? w.collapsed, span: s.span ?? w.span }
          })
      } catch (_) {}
    },

    resetLayout() {
      try { localStorage.removeItem('mon-widget-layout') } catch (_) {}
      this.widgets = DEFAULT_WIDGETS.map(w => ({ ...w }))
    }
  }
}
</script>

<style scoped>
/* ── Widget grid ─────────────────────────────────────────────────────── */
.mon-widget-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  align-items: start;
}

/* ── Widget card ─────────────────────────────────────────────────────── */
.wgt-card {
  background: var(--surface-1);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: box-shadow 0.2s;
}

.wgt-card:has(.wgt-drag:active) {
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
}

.wgt-span-1 { grid-column: span 1; }
.wgt-span-2 { grid-column: span 2; }

.wgt-ghost {
  opacity: 0.2;
  border: 2px dashed var(--accent, #4a9eff) !important;
  background: transparent !important;
}

/* ── Widget header ───────────────────────────────────────────────────── */
.wgt-header {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.55rem 0.85rem;
  border-bottom: 1px solid var(--border-subtle);
  user-select: none;
  transition: border-color 0.25s;
}

.wgt-collapsed .wgt-header {
  border-bottom-color: transparent;
}

.wgt-drag {
  cursor: grab;
  color: var(--text-tertiary);
  font-size: 1.1rem;
  padding: 1px 2px;
  border-radius: 4px;
  transition: color 0.15s, background 0.15s;
  flex-shrink: 0;
  line-height: 1;
}

.wgt-drag:hover  { color: var(--text-primary); background: var(--surface-3); }
.wgt-drag:active { cursor: grabbing; }

.wgt-icon {
  font-size: 0.95rem;
  flex-shrink: 0;
}

.wgt-title {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-primary);
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.wgt-actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.wgt-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--text-tertiary);
  cursor: pointer;
  font-size: 0.82rem;
  transition: background 0.15s, color 0.15s;
}

.wgt-btn:hover { background: var(--surface-3); color: var(--text-primary); }

/* ── Widget body + collapse animation ────────────────────────────────── */
.wgt-body {
  overflow: hidden;
  max-height: 2400px;
  opacity: 1;
  transition:
    max-height 0.3s cubic-bezier(0.4, 0, 0.2, 1),
    opacity    0.2s ease;
}

.wgt-collapsed .wgt-body {
  max-height: 0;
  opacity: 0;
  pointer-events: none;
}

/* ── Gauge row (resources widget) ────────────────────────────────────── */
.wgt-gauge-row {
  display: flex;
  flex-wrap: wrap;
}

.wgt-gauge-cell {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.85rem 1rem;
  flex: 1;
  min-width: 160px;
  border-right: 1px solid var(--border-subtle);
}

.wgt-gauge-cell:last-child { border-right: none; }

.wgt-gauge-cell--net {
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 0.3rem;
}

/* ── Chart body ──────────────────────────────────────────────────────── */
.wgt-chart-body {
  padding: 0.5rem 0.25rem 0.75rem;
}

/* ── Sub-header bars (filter, scan) ─────────────────────────────────── */
.wgt-filter-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0.45rem 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-2);
}

/* ── Reset layout button (page header slot) ─────────────────────────── */
.wgt-reset-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 11px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.wgt-reset-btn:hover { background: var(--surface-3); color: var(--text-primary); }

/* ── Scroll-limited widget content ──────────────────────────────────── */
.mon-scroll-body {
  max-height: 280px;
  overflow-y: auto;
}

/* ── OLD gauge card (resource strip kept for gauge cell reuse) ───────── */
.mon-gauge-strip {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.mon-gauge-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  background: var(--surface-1);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
}

.mon-gauge-card--net {
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 0.3rem;
}

.mon-gauge {
  flex-shrink: 0;
}

.mon-gauge-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.mon-gauge-label {
  font-size: 0.68rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-tertiary);
}

.mon-gauge-value {
  font-size: 1.35rem;
  font-weight: 700;
  line-height: 1.1;
}

.mon-gauge-sub {
  font-size: 0.71rem;
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mon-net-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.mon-net-val {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1;
}

.mon-net-lbl {
  font-size: 0.68rem;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ── Two-column main grid ──────────────────────────────────────────── */
.mon-main-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: 0.75rem;
  align-items: start;
}

.mon-chart-col,
.mon-side-col {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* ── Shared card chrome ───────────────────────────────────────────── */
.mon-card {
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.mon-card-header {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.65rem 1rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.mon-card-header .mdi {
  font-size: 1rem;
}

.mon-card-body {
  padding: 0.85rem 1rem;
}

.mon-card-body--chart {
  padding: 0.5rem 0.5rem 0.75rem;
}

.mon-card-body--scroll {
  max-height: 280px;
  overflow-y: auto;
  padding: 0;
}

/* ── Disk partition rows ──────────────────────────────────────────── */
.mon-disk-row {
  padding: 0.45rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.mon-disk-row:last-child {
  border-bottom: none;
}

.mon-disk-row__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.mon-disk-row__size {
  font-size: 0.71rem;
  color: var(--text-tertiary);
}

.mon-mount-btn {
  background: none;
  border: none;
  padding: 0;
  font-family: var(--font-mono, monospace);
  font-size: 0.73rem;
  color: #4a9eff;
  cursor: pointer;
  text-align: left;
}

.mon-mount-btn:hover {
  text-decoration: underline;
}

.mon-progress {
  height: 4px;
  border-radius: 2px;
  background: var(--surface-3);
  overflow: hidden;
}

.mon-progress__bar {
  height: 100%;
  border-radius: 2px;
  transition: width 0.4s ease;
}

.mon-progress__bar.is-ok     { background: #22d67c; }
.mon-progress__bar.is-warn   { background: #f5a623; }
.mon-progress__bar.is-danger { background: #f04040; }

/* ── Tables ──────────────────────────────────────────────────────── */
.mon-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.75rem;
}

.mon-table th {
  padding: 0.45rem 0.75rem;
  background: var(--surface-2);
  color: var(--text-tertiary);
  font-weight: 600;
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid var(--border-subtle);
  white-space: nowrap;
}

.mon-table td {
  padding: 0.4rem 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
  vertical-align: middle;
}

.mon-table tbody tr:last-child td {
  border-bottom: none;
}

.mon-table tbody tr:hover {
  background: var(--surface-2);
}

.mon-sortable {
  cursor: pointer;
  user-select: none;
}

.mon-sortable:hover {
  color: var(--text-primary);
}

.mon-mono  { font-family: var(--font-mono, monospace); }
.mon-muted { color: var(--text-tertiary); }
.mon-bold  { font-weight: 600; color: var(--text-primary); }

.mon-proc-name {
  color: #4a9eff;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
}

.mon-proc-name:hover {
  color: #7bbdff;
}

.mon-cmd {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mon-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.mon-dot.is-up   { background: #22d67c; }
.mon-dot.is-down { background: #f04040; }

/* ── Status / risk badges ─────────────────────────────────────────── */
.mon-status-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 0.62rem;
  font-weight: 600;
  background: rgba(34, 214, 124, 0.12);
  color: #22d67c;
}

.mon-risk-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 0.62rem;
  font-weight: 600;
  text-transform: capitalize;
}

.mon-risk-badge.is-high   { background: rgba(240,64,64,0.15);   color: #f04040; }
.mon-risk-badge.is-medium { background: rgba(245,166,35,0.15);  color: #f5a623; }
.mon-risk-badge.is-low    { background: rgba(34,214,124,0.12);  color: #22d67c; }

.mon-badge-danger {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 0.62rem;
  font-weight: 700;
  background: rgba(240,64,64,0.15);
  color: #f04040;
}

/* ── Filter input in card header ─────────────────────────────────── */
.mon-filter-input {
  margin-left: auto;
  width: 140px;
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  color: var(--text-primary);
  font-size: 0.72rem;
  outline: none;
}

.mon-filter-input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(74,158,255,0.12);
}

/* ── Scan button ─────────────────────────────────────────────────── */
.mon-scan-btn {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: rgba(74,158,255,0.06);
  color: #4a9eff;
  font-size: 0.72rem;
  cursor: pointer;
  transition: background 0.15s;
}

.mon-scan-btn:hover:not(:disabled) {
  background: rgba(74,158,255,0.12);
}

.mon-scan-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

/* ── Empty state ─────────────────────────────────────────────────── */
.mon-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 1.5rem;
  font-size: 0.8rem;
  color: var(--text-tertiary);
}

.mon-empty--ok {
  color: #22d67c;
}

/* ── Modal overlay ───────────────────────────────────────────────── */
.sc-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1600;
  padding: 1rem;
}

.sc-modal-card {
  width: min(860px, 96vw);
  max-height: 85vh;
  overflow: auto;
  background: var(--surface-1);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  padding: 1rem;
}

.sc-modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  font-size: 0.88rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 0.75rem;
}

.sc-modal-head .mdi {
  margin-right: 4px;
}

.mon-close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: rgba(240,64,64,0.1);
  color: #f04040;
  cursor: pointer;
  font-size: 1rem;
  transition: background 0.15s;
}

.mon-close-btn:hover { background: rgba(240,64,64,0.2); }

.sc-modal-meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.72rem;
  color: var(--text-tertiary);
  margin-bottom: 0.5rem;
}

.sc-modal-scroll {
  max-height: 360px;
  overflow: auto;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.sc-modal-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.sc-modal-desc {
  grid-column: 1 / -1;
}

.sc-modal-row {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.35rem 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 0.78rem;
}

.sc-modal-row > span:first-child {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

/* ── Responsive ──────────────────────────────────────────────────── */
@media (max-width: 1100px) {
  .mon-gauge-strip {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 900px) {
  .mon-main-grid {
    grid-template-columns: 1fr;
  }

  .mon-gauge-strip {
    grid-template-columns: repeat(2, 1fr);
  }

  .sc-modal-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 900px) {
  .mon-widget-grid { grid-template-columns: 1fr; }
  .wgt-span-1, .wgt-span-2 { grid-column: span 1; }
}

@media (max-width: 600px) {
  .wgt-gauge-row { flex-direction: column; }
  .wgt-gauge-cell { border-right: none; border-bottom: 1px solid var(--border-subtle); min-width: 0; }
  .wgt-gauge-cell:last-child { border-bottom: none; }
}
</style>

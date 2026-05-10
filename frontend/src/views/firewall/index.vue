<template>
  <div>
    <PageHeader title="Firewall" icon="mdi mdi-shield-lock" :items="[{text:'Firewall',active:true,icon:'mdi mdi-shield-lock'}]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" @click="showAddRule = true">
          <i class="mdi mdi-plus me-1"></i> Add Rule
        </button>
      </template>
    </PageHeader>

    <!-- UFW status bar -->
    <div class="card mb-4" :style="`border-color:${ufwActive?'rgba(34,214,124,0.3)':'rgba(240,64,64,0.3)'}`">
      <div class="card-body py-2 d-flex align-items-center gap-3">
        <span class="status-dot" :class="ufwActive?'online':'offline'"></span>
        <span style="font-weight:600;font-size:0.85rem;color:#c9d8f0">UFW {{ ufwActive ? 'Active' : 'Inactive' }}</span>
        <span style="font-size:0.78rem;color:#5a7499">Default policy: <code>DENY incoming</code> / <code>ALLOW outgoing</code></span>
        <div class="ms-auto d-flex gap-2">
          <button class="btn btn-sm" style="background:rgba(34,214,124,0.12);color:#22d67c;font-size:0.72rem" @click="enableUfw">Enable</button>
          <button class="btn btn-sm btn-sc-danger" @click="disableUfw">Disable</button>
          <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;font-size:0.72rem" @click="reloadUfw">Reload</button>
        </div>
      </div>
    </div>

    <!-- Loading state (initial load only) -->
    <div v-if="initialLoading" class="text-center py-5">
      <div class="spinner-border text-primary" style="width:2rem;height:2rem"></div>
      <div style="color:#5a7499;font-size:0.8rem;margin-top:0.5rem">Loading firewall rules…</div>
    </div>

    <template v-else>
      <div class="row g-3 mb-4">
        <!-- UFW Rules table -->
        <div class="col-xl-8">
          <div class="card">
            <div class="card-header d-flex align-items-center justify-content-between">
              <h6><i class="mdi mdi-wall me-2" style="color:#4a9eff"></i>UFW Rules ({{ rules.length }})</h6>
              <div class="d-flex gap-2 align-items-center">
                <select v-model="ruleFilter" class="form-select form-select-sm" style="width:120px;font-size:0.75rem">
                  <option value="">All</option>
                  <option value="allow">Allow</option>
                  <option value="deny">Deny</option>
                </select>
                <input v-model="ruleSearch" class="form-control form-control-sm" placeholder="Search…" style="width:140px" />
                <Tooltip label="Refresh rules" as-child>
                  <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;font-size:0.72rem;padding:4px 8px" @click="loadRules">
                    <i class="mdi mdi-refresh"></i>
                  </button>
                </Tooltip>
              </div>
            </div>
            <div class="card-body p-0">
              <table class="table mb-0">
                <thead>
                  <tr><th>#</th><th>To</th><th>Action</th><th>From</th><th>Comment</th><th>Actions</th></tr>
                </thead>
                <tbody>
                  <tr v-if="filteredRules.length === 0">
                    <td colspan="6" class="text-center py-4" style="color:#5a7499;font-size:0.8rem">No rules found</td>
                  </tr>
                  <tr v-for="rule in filteredRules" :key="rule.number">
                    <td style="font-size:0.72rem;color:#5a7499">{{ rule.number }}</td>
                    <td class="font-mono" style="font-size:0.78rem;color:#c9d8f0">{{ rule.to }}</td>
                    <td>
                      <span class="badge rounded-pill" :class="rule.action.toUpperCase()==='ALLOW'?'badge-online':'badge-offline'">
                        {{ rule.action }}
                      </span>
                    </td>
                    <td class="font-mono" style="font-size:0.78rem;color:#8aa4c8">{{ rule.from }}</td>
                    <td style="font-size:0.75rem;color:#5a7499">{{ rule.comment }}</td>
                    <td>
                      <button class="btn btn-sm btn-sc-danger" style="font-size:0.68rem;padding:2px 8px" @click="deleteRule(rule.number)">
                        <i class="mdi mdi-delete-outline"></i>
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <!-- Quick block + stats -->
        <div class="col-xl-4">
        <div class="card mb-3">
        <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-block-helper me-2" style="color:#f04040"></i>Quick Block IP</h6>
        <Tooltip
          label="Bulk block help"
          description="Block single IPs, subnets (CIDR), or multiple IPs at once. Use commas or newlines to separate multiple IPs. Example: 192.168.1.1, 10.0.0.0/8"
          variant="rich"
          as-child
        >
          <button type="button" class="btn p-0 border-0 bg-transparent text-muted" style="cursor:pointer;font-size:1.1rem" @click.stop="showBulkBlockModal = true" aria-label="Bulk block help">
            <i class="mdi mdi-information-outline"></i>
          </button>
        </Tooltip>
        </div>
        <div class="card-body">
        <div class="mb-2">
        <input v-model="blockIp" class="form-control" placeholder="e.g. 45.152.66.102" />
        </div>
        <button class="btn btn-sc-danger w-100" @click="blockIpNow" :disabled="!blockIp">
        <i class="mdi mdi-block-helper me-1"></i> Block IP
        </button>
        <div class="mt-2">
        <input v-model="blockSubnet" class="form-control" placeholder="e.g. 45.152.0.0/16 (subnet)" />
        </div>
        <button class="btn btn-sc-danger w-100 mt-2" @click="blockSubnetNow" :disabled="!blockSubnet">
        <i class="mdi mdi-block-helper me-1"></i> Block Subnet
        </button>
        <button class="btn btn-sc-primary w-100 mt-2" @click="showBulkBlockModal = true">
        <i class="mdi mdi-ip me-1"></i> Bulk Block IPs
        </button>
        </div>
        </div>

          <div class="card">
            <div class="card-header"><h6><i class="mdi mdi-chart-bar me-2" style="color:#4a9eff"></i>Stats</h6></div>
            <div class="card-body">
              <div class="d-flex justify-content-between mb-2" style="font-size:0.8rem">
                <span style="color:#8aa4c8">Total rules</span>
                <span style="color:#c9d8f0;font-weight:600">{{ rules.length }}</span>
              </div>
              <div class="d-flex justify-content-between mb-2" style="font-size:0.8rem">
                <span style="color:#8aa4c8">Allow rules</span>
                <span style="color:#22d67c;font-weight:600">{{ rules.filter(r=>r.action.toUpperCase()==='ALLOW').length }}</span>
              </div>
              <div class="d-flex justify-content-between mb-2" style="font-size:0.8rem">
                <span style="color:#8aa4c8">Deny rules</span>
                <span style="color:#f04040;font-weight:600">{{ rules.filter(r=>r.action.toUpperCase()==='DENY').length }}</span>
              </div>
              <div class="d-flex justify-content-between" style="font-size:0.8rem">
                <span style="color:#8aa4c8">IPv6 rules</span>
                <span style="color:#4a9eff;font-weight:600">{{ rules.filter(r=>r.from&&r.from.includes(':')).length }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Active connections -->
      <div class="card">
        <div class="card-header d-flex align-items-center justify-content-between">
          <h6><i class="mdi mdi-connection me-2" style="color:#22d3ee"></i>Active Connections</h6>
          <div class="d-flex align-items-center gap-2">
            <span style="font-size:0.75rem;color:#5a7499">{{ connections.length }} total</span>
            <Tooltip label="Refresh connections" as-child>
              <button class="btn btn-sm" style="background:rgba(74,158,255,0.12);color:#4a9eff;font-size:0.72rem;padding:4px 8px" @click="loadRules">
                <i class="mdi mdi-refresh"></i>
              </button>
            </Tooltip>
          </div>
        </div>
        <div class="card-body p-0" style="max-height:280px;overflow-y:auto">
          <table class="table mb-0">
            <thead><tr><th>Proto</th><th>Local</th><th>Remote</th><th>State</th></tr></thead>
            <tbody>
              <tr v-if="connections.length === 0">
                <td colspan="4" class="text-center py-3" style="color:#5a7499;font-size:0.8rem">No active connections</td>
              </tr>
              <tr v-for="(c, idx) in connections" :key="idx">
                <td style="font-size:0.75rem;color:#22d3ee;font-weight:600">{{ c.protocol }}</td>
                <td class="font-mono" style="font-size:0.75rem;color:#c9d8f0">{{ c.local_addr }}</td>
                <td class="font-mono" style="font-size:0.75rem;color:#8aa4c8">{{ c.remote_addr }}</td>
                <td>
                  <span class="badge" :style="`background:rgba(${getConnColor(c.state)}? '34,214,124' : '74,158,255'},0.12);color:${getConnColor(c.state)? '#22d67c' : '#4a9eff'};font-size:0.62rem`">{{ formatState(c.state) }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Bulk block IPs modal -->
    <div v-if="showBulkBlockModal" class="modal d-block" style="background:rgba(0,0,0,0.7)">
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title"><i class="mdi mdi-ip me-2" style="color:#f04040"></i>Bulk Block IPs</h5>
            <button class="btn-close" @click="showBulkBlockModal=false"></button>
          </div>
          <div class="modal-body">
            <p class="text-muted" style="font-size:0.85rem">Enter IPs, subnets (CIDR), or IP ranges. Use commas or newlines to separate entries.</p>
            <textarea v-model="bulkIps" class="form-control" rows="8"
                      placeholder="192.168.1.100&#10;10.0.0.0/8&#10;45.152.66.102, 185.220.101.1, 185.220.101.2&#10;172.16.0.0/12"></textarea>
            <div class="mt-2 text-muted" style="font-size:0.75rem">
              <i class="mdi mdi-information-outline"></i> Examples:
              <ul class="mb-0 mt-1" style="list-style:none;padding-left:0">
                <li>• Single IP: <code>192.168.1.100</code></li>
                <li>• Subnet (CIDR): <code>10.0.0.0/8</code></li>
                <li>• Comma-separated: <code>1.1.1.1, 2.2.2.2, 3.3.3.3</code></li>
              </ul>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm" style="background:#1e2d4a;color:#8aa4c8" @click="showBulkBlockModal=false">Cancel</button>
            <button class="btn btn-sm btn-sc-danger" @click="bulkBlockIpsNow" :disabled="bulkBlocking || !bulkIps.trim()">
              <span v-if="bulkBlocking" class="spinner-border spinner-border-sm me-1"></span>
              <i class="mdi mdi-block-helper me-1"></i> Block All
            </button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- Add rule modal -->
    <div v-if="showAddRule" class="modal d-block" style="background:rgba(0,0,0,0.7)">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Add UFW Rule</h5>
            <button class="btn-close" @click="showAddRule=false"></button>
          </div>
          <div class="modal-body">
            <div class="row g-3">
              <div class="col-6">
                <label class="form-label">Action</label>
                <select v-model="newRule.action" class="form-select">
                  <option>allow</option><option>deny</option><option>reject</option>
                </select>
              </div>
              <div class="col-6">
                <label class="form-label">Direction</label>
                <select v-model="newRule.direction" class="form-select">
                  <option>in</option><option>out</option>
                </select>
              </div>
              <div class="col-6">
                <label class="form-label">Port / Service</label>
                <input v-model="newRule.port" class="form-control" placeholder="e.g. 80/tcp or ssh" />
              </div>
              <div class="col-6">
                <label class="form-label">From IP (optional)</label>
                <input v-model="newRule.from" class="form-control" placeholder="any" />
              </div>
              <div class="col-12">
                <label class="form-label">Comment</label>
                <input v-model="newRule.comment" class="form-control" placeholder="Optional comment" />
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm" style="background:#1e2d4a;color:#8aa4c8" @click="showAddRule=false">Cancel</button>
            <button class="btn btn-sm btn-sc-primary" @click="addRule" :disabled="addingRule">
              <span v-if="addingRule" class="spinner-border spinner-border-sm me-1"></span>
              Add Rule
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

export default {
  name: 'FirewallPage',
  setup() {
    return {
      documentVisibility: useDocumentVisibility()
    }
  },
  components: { PageHeader, Tooltip },
  data() {
  return {
  loading: false,
  initialLoading: true,
  addingRule: false,
  bulkBlocking: false,
  ufwActive: false,
  ruleFilter: '',
  ruleSearch: '',
  blockIp: '',
  blockSubnet: '',
  bulkIps: '',
  showAddRule: false,
  showBulkBlockModal: false,
  newRule: { action: 'allow', direction: 'in', port: '', from: '', comment: '' },
  rules: [],
  connections: []
  }
  },

  computed: {
    filteredRules() {
      let r = this.rules
      if (this.ruleFilter) r = r.filter(x => x.action.toLowerCase() === this.ruleFilter)
      if (this.ruleSearch) {
        const s = this.ruleSearch.toLowerCase()
        r = r.filter(x =>
          (x.to || '').toLowerCase().includes(s) ||
          (x.from || '').toLowerCase().includes(s) ||
          (x.comment || '').toLowerCase().includes(s)
        )
      }
      return r
    }
  },

  mounted() {
    this.loadRules(true)
    this.connTimer = setInterval(() => {
      if (this.documentVisibility !== 'visible') return
      this.loadRules(false)
    }, 15000)
  },
  watch: {
    documentVisibility(value) {
      if (value === 'visible') {
        this.loadRules(false)
      }
    }
  },
  
  beforeUnmount() {
    if (this.connTimer) clearInterval(this.connTimer)
  },
  
  methods: {
    formatState(state) {
      // Normalize connection state display
      if (!state) return 'UNKNOWN'
      const s = state.toUpperCase()
      if (s === 'ESTAB') return 'ESTABLISHED'
      if (s === 'LISTEN') return 'LISTENING'
      if (s === 'TIME_WAIT') return 'TIME-WAIT'
      if (s === 'CLOSE_WAIT') return 'CLOSE-WAIT'
      return s
    },
  
    getConnColor(state) {
      const s = state?.toUpperCase()
      return s === 'ESTAB' || s === 'ESTABLISHED'
    },
  
    async loadRules(isInitial = false) {
      if (isInitial) this.initialLoading = true
      try {
        const res = await api.getFirewallRules()
        const data = res.data
        this.ufwActive = data.enabled ?? false
        const newRules = data.rules || []
        const newConns = Array.isArray(data.connections) ? data.connections : []
        if (isInitial) {
          this.rules = newRules
          this.connections = newConns
        } else {
          this._mergeById(this.rules, newRules, 'number')
          this._mergeByIdx(this.connections, newConns)
        }
      } catch (err) {
        if (isInitial) {
          this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to load firewall rules', showConfirmButton: false, timer: 3000 })
        }
      } finally {
        if (isInitial) this.initialLoading = false
      }
    },

    _mergeById(current, incoming, key) {
      const inMap = new Map(incoming.map(r => [r[key], r]))
      // update/add
      incoming.forEach(r => {
        const idx = current.findIndex(c => c[key] === r[key])
        if (idx >= 0) Object.assign(current[idx], r)
        else current.push(r)
      })
      // remove stale
      for (let i = current.length - 1; i >= 0; i--) {
        if (!inMap.has(current[i][key])) current.splice(i, 1)
      }
    },

    _mergeByIdx(current, incoming) {
      incoming.forEach((r, i) => {
        if (i < current.length) Object.assign(current[i], r)
        else current.push(r)
      })
      if (current.length > incoming.length) current.splice(incoming.length)
    },

    reloadUfw() {
      this.$swal({ toast: true, position: 'top-end', icon: 'info', title: 'UFW management done via terminal', showConfirmButton: false, timer: 2500 })
    },

    enableUfw() {
      this.$swal({ toast: true, position: 'top-end', icon: 'info', title: 'UFW management done via terminal', showConfirmButton: false, timer: 2500 })
    },

    disableUfw() {
      this.$swal({ toast: true, position: 'top-end', icon: 'info', title: 'UFW management done via terminal', showConfirmButton: false, timer: 2500 })
    },

    async blockIpNow() {
      if (!this.blockIp) return
      const ip = this.blockIp
      try {
        await api.addFirewallRule({ port: 'any', action: 'deny', from: ip })
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Blocked ${ip}`, showConfirmButton: false, timer: 2000 })
        this.blockIp = ''
        await this.loadRules()
      } catch (err) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to block IP', showConfirmButton: false, timer: 3000 })
      }
    },

    async blockSubnetNow() {
      if (!this.blockSubnet) return
      const subnet = this.blockSubnet
      try {
      await api.addFirewallRule({ port: 'any', action: 'deny', from: subnet })
      this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `Blocked ${subnet}`, showConfirmButton: false, timer: 2000 })
      this.blockSubnet = ''
      await this.loadRules()
      } catch (err) {
      this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to block subnet', showConfirmButton: false, timer: 3000 })
      }
      },
    
      async bulkBlockIpsNow() {
      if (!this.bulkIps.trim()) return
      this.bulkBlocking = true
      // Parse IPs: split by comma or newline, trim, filter empty
      const ips = this.bulkIps.split(/[,\n]/).map(s => s.trim()).filter(s => s)
      const results = { success: [], failed: [] }
      
      for (const ip of ips) {
        try {
          await api.addFirewallRule({ port: 'any', action: 'deny', from: ip })
          results.success.push(ip)
        } catch (err) {
          results.failed.push(ip)
        }
      }
      
      this.bulkBlocking = false
      this.showBulkBlockModal = false
      this.bulkIps = ''
      
      // Show result notification
      if (results.success.length > 0 && results.failed.length === 0) {
        this.$swal({ toast: true, position: 'top-end', icon: 'success',
          title: `Blocked ${results.success.length} IPs successfully`, showConfirmButton: false, timer: 3000 })
      } else if (results.success.length > 0 && results.failed.length > 0) {
        this.$swal({ toast: true, position: 'top-end', icon: 'warning',
          title: `Blocked ${results.success.length}, failed: ${results.failed.length}`, showConfirmButton: false, timer: 4000 })
      } else {
        this.$swal({ toast: true, position: 'top-end', icon: 'error',
          title: `Failed to block ${results.failed.length} IPs`, showConfirmButton: false, timer: 4000 })
      }
      await this.loadRules()
      },
    
      deleteRule(number) {
      this.$swal({
        title: 'Delete rule?',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#f04040',
        confirmButtonText: 'Delete'
      }).then(async r => {
        if (r.isConfirmed) {
          try {
            await api.deleteFirewallRule(number)
            this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Rule deleted', showConfirmButton: false, timer: 2000 })
            await this.loadRules()
          } catch (err) {
            this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to delete rule', showConfirmButton: false, timer: 3000 })
          }
        }
      })
    },

    async addRule() {
      this.addingRule = true
      try {
        await api.addFirewallRule({
          port: this.newRule.port,
          protocol: this.newRule.direction === 'in' ? 'tcp' : this.newRule.direction,
          action: this.newRule.action,
          from: this.newRule.from || 'any',
          comment: this.newRule.comment
        })
        this.showAddRule = false
        this.newRule = { action: 'allow', direction: 'in', port: '', from: '', comment: '' }
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'Rule added', showConfirmButton: false, timer: 2000 })
        await this.loadRules()
      } catch (err) {
        this.$swal({ toast: true, position: 'top-end', icon: 'error', title: 'Failed to add rule', showConfirmButton: false, timer: 3000 })
      } finally {
        this.addingRule = false
      }
    }
  }
}
</script>

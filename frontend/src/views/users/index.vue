<template>
  <div>
    <PageHeader title="Users" icon="mdi mdi-account-group" :items="[{text:'Users',active:true,icon:'mdi mdi-account-group'}]">
      <template #actions>
        <button class="btn btn-sm btn-sc-primary" @click="showNewUser = true">
          <i class="mdi mdi-account-plus-outline me-1"></i> Add User
        </button>
      </template>
    </PageHeader>

    <!-- RBAC legend -->
    <div class="card mb-4" style="background:rgba(74,158,255,0.05);border-color:rgba(74,158,255,0.2)">
      <div class="card-body py-2">
        <div class="d-flex flex-wrap gap-3 align-items-center">
          <span style="font-size:0.75rem;color:#5a7499;font-weight:600">Roles:</span>
          <span v-for="role in roles" :key="role.name" class="d-flex align-items-center gap-1" style="font-size:0.75rem">
            <span class="badge" :style="`background:rgba(${role.rgb},0.15);color:rgba(${role.rgb},1)`">{{ role.name }}</span>
            <span style="color:#5a7499">{{ role.desc }}</span>
          </span>
        </div>
      </div>
    </div>

    <!-- Users table -->
    <div class="card mb-4">
      <div class="card-header d-flex align-items-center justify-content-between">
        <h6><i class="mdi mdi-account-group me-2" style="color:#4a9eff"></i>Users ({{ users.length }})</h6>
        <div v-if="loading" class="spinner-border spinner-border-sm text-info" role="status"></div>
      </div>
      <div class="card-body p-0">
        <div v-if="loadError" class="alert alert-danger m-3 mb-0">{{ loadError }}</div>
        <table class="table mb-0">
          <thead>
            <tr><th>User</th><th>Role</th><th>Created</th><th>2FA</th><th>Actions</th></tr>
          </thead>
          <tbody>
            <tr v-if="!loading && users.length === 0">
              <td colspan="5" class="text-center" style="color:#5a7499;font-size:0.8rem;padding:1.5rem">No users found</td>
            </tr>
            <tr v-for="user in users" :key="user.id">
              <td>
                <div class="d-flex align-items-center gap-2">
                  <div :style="`width:30px;height:30px;border-radius:6px;background:linear-gradient(135deg,rgba(${roleRgb(user.role)},0.5),rgba(${roleRgb(user.role)},0.2));display:flex;align-items:center;justify-content:center;font-size:0.7rem;font-weight:700;color:#fff`">
                    {{ user.username.slice(0,2).toUpperCase() }}
                  </div>
                  <div>
                    <div style="font-size:0.8rem;font-weight:600;color:#c9d8f0">{{ user.username }}</div>
                    <div style="font-size:0.7rem;color:#5a7499">{{ user.email }}</div>
                  </div>
                </div>
              </td>
              <td>
                <span class="badge" :style="`background:rgba(${roleRgb(user.role)},0.15);color:rgba(${roleRgb(user.role)},1);font-size:0.65rem`">{{ user.role }}</span>
              </td>
              <td style="font-size:0.75rem;color:#8aa4c8">{{ formatDate(user.created_at) }}</td>
              <td>
                <span class="badge" :class="user.totp_enabled ? 'badge-online' : 'badge-offline'">{{ user.totp_enabled ? '2FA' : 'None' }}</span>
              </td>
              <td>
                <div class="d-flex gap-1">
                  <button class="btn btn-sm" style="background:rgba(74,158,255,0.1);color:#4a9eff;font-size:0.68rem;padding:2px 8px" @click="openEditUser(user)">
                    <i class="mdi mdi-pencil-outline"></i>
                  </button>
                  <button v-if="user.username !== 'admin'" class="btn btn-sm btn-sc-danger" @click="deleteUser(user)">
                    <i class="mdi mdi-delete-outline"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Permissions matrix -->
    <div class="card">
      <div class="card-header"><h6><i class="mdi mdi-shield-account me-2" style="color:#a78bfa"></i>RBAC Permission Matrix</h6></div>
      <div class="card-body p-0">
        <div style="overflow-x:auto">
          <table class="table mb-0" style="min-width:700px">
            <thead>
              <tr>
                <th style="width:200px">Module</th>
                <th v-for="role in ['superadmin','admin','operator','viewer']" :key="role" class="text-center">
                  <span class="badge" :style="`background:rgba(${roleRgb(role)},0.15);color:rgba(${roleRgb(role)},1);font-size:0.65rem`">{{ role }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="perm in permissions" :key="perm.module">
                <td style="font-size:0.78rem;color:#c9d8f0">{{ perm.module }}</td>
                <td v-for="role in ['superadmin','admin','operator','viewer']" :key="role" class="text-center">
                  <i :class="perm[role] ? 'mdi mdi-check-circle' : 'mdi mdi-minus-circle'" :style="`color:${perm[role]?'#22d67c':'#2d3748'};font-size:1rem`"></i>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Add user modal -->
    <div v-if="showNewUser" class="modal d-block" style="background:rgba(0,0,0,0.7)">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Add User</h5>
            <button class="btn-close" @click="closeNewUser"></button>
          </div>
          <div class="modal-body">
            <div class="row g-3">
              <div class="col-6">
                <label class="form-label">Username</label>
                <input v-model="newUser.username" class="form-control" />
              </div>
              <div class="col-6">
                <label class="form-label">Email</label>
                <input v-model="newUser.email" type="email" class="form-control" />
              </div>
              <div class="col-6">
                <label class="form-label">Role</label>
                <select v-model="newUser.role" class="form-select">
                  <option>viewer</option><option>operator</option><option>admin</option>
                </select>
              </div>
              <div class="col-6">
                <label class="form-label">Password</label>
                <input v-model="newUser.password" type="password" class="form-control" />
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm" style="background:#1e2d4a;color:#8aa4c8" @click="closeNewUser">Cancel</button>
            <button class="btn btn-sm btn-sc-primary" :disabled="creating" @click="createUser">
              <span v-if="creating" class="spinner-border spinner-border-sm me-1"></span>
              Create
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Edit user modal -->
    <div v-if="showEditUser" class="modal d-block" style="background:rgba(0,0,0,0.7)">
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Edit User — {{ editingUser && editingUser.username }}</h5>
            <button class="btn-close" @click="closeEditUser"></button>
          </div>
          <div class="modal-body">
            <div class="row g-3">
              <div class="col-6">
                <label class="form-label">Role</label>
                <select v-model="editForm.role" class="form-select">
                  <option>viewer</option><option>operator</option><option>admin</option><option>superadmin</option>
                </select>
              </div>
              <div class="col-6">
                <label class="form-label">Email</label>
                <input v-model="editForm.email" type="email" class="form-control" />
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-sm" style="background:#1e2d4a;color:#8aa4c8" @click="closeEditUser">Cancel</button>
            <button class="btn btn-sm btn-sc-primary" :disabled="saving" @click="saveEditUser">
              <span v-if="saving" class="spinner-border spinner-border-sm me-1"></span>
              Save
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import PageHeader from '@/components/page-header.vue'
import api from '@/services/api'

export default {
  name: 'UsersPage',
  components: { PageHeader },

  data() {
    return {
      loading:      false,
      loadError:    null,

      showNewUser:  false,
      creating:     false,
      newUser:      { username: '', email: '', role: 'viewer', password: '' },

      showEditUser: false,
      saving:       false,
      editingUser:  null,
      editForm:     { role: '', email: '' },

      users: [],

      roles: [
        { name: 'superadmin', rgb: '240,64,64',   desc: 'Full access' },
        { name: 'admin',      rgb: '245,166,35',  desc: 'Most operations' },
        { name: 'operator',   rgb: '74,158,255',  desc: 'Run tasks, view all' },
        { name: 'viewer',     rgb: '138,164,200', desc: 'Read-only access' }
      ],

      permissions: [
        { module: 'Dashboard',       superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Security Center', superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Firewall',        superadmin: true,  admin: true,  operator: false, viewer: false },
        { module: 'Monitoring',      superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Containers',      superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Logs',            superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Terminal',        superadmin: true,  admin: true,  operator: false, viewer: false },
        { module: 'Tasks (run)',     superadmin: true,  admin: true,  operator: true,  viewer: false },
        { module: 'Alerts',          superadmin: true,  admin: true,  operator: true,  viewer: true  },
        { module: 'Users (manage)',  superadmin: true,  admin: true,  operator: false, viewer: false },
        { module: 'Settings',        superadmin: true,  admin: false, operator: false, viewer: false },
        { module: 'Audit Logs',      superadmin: true,  admin: true,  operator: true,  viewer: false }
      ]
    }
  },

  mounted() {
    this.loadUsers()
  },

  methods: {
    async loadUsers() {
      this.loading   = true
      this.loadError = null
      try {
        const res    = await api.getUsers()
        this.users   = res.data
      } catch (err) {
        this.loadError = err.response?.data?.error || 'Failed to load users'
      } finally {
        this.loading = false
      }
    },

    roleRgb(r) {
      return { superadmin: '240,64,64', admin: '245,166,35', operator: '74,158,255', viewer: '138,164,200' }[r] || '138,164,200'
    },

    formatDate(ts) {
      if (!ts) return '—'
      try {
        return new Date(ts).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
      } catch {
        return ts
      }
    },

    // ── Edit ──────────────────────────────────────────────────────────────────
    openEditUser(user) {
      this.editingUser = user
      this.editForm    = { role: user.role, email: user.email }
      this.showEditUser = true
    },

    closeEditUser() {
      this.showEditUser = false
      this.editingUser  = null
      this.editForm     = { role: '', email: '' }
    },

    async saveEditUser() {
      if (!this.editingUser) return
      this.saving = true
      try {
        await api.updateUser(this.editingUser.id, this.editForm)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: 'User updated', showConfirmButton: false, timer: 2000 })
        this.closeEditUser()
        await this.loadUsers()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Error', text: err.response?.data?.error || 'Failed to update user' })
      } finally {
        this.saving = false
      }
    },

    // ── Delete ────────────────────────────────────────────────────────────────
    async deleteUser(user) {
      const result = await this.$swal({
        title: `Delete ${user.username}?`,
        text: 'This action cannot be undone.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonColor: '#f04040',
        confirmButtonText: 'Delete'
      })
      if (!result.isConfirmed) return
      try {
        await api.deleteUser(user.id)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `${user.username} deleted`, showConfirmButton: false, timer: 2000 })
        await this.loadUsers()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Error', text: err.response?.data?.error || 'Failed to delete user' })
      }
    },

    // ── Create ────────────────────────────────────────────────────────────────
    closeNewUser() {
      this.showNewUser = false
      this.newUser     = { username: '', email: '', role: 'viewer', password: '' }
    },

    async createUser() {
      if (!this.newUser.username || !this.newUser.password) {
        this.$swal({ toast: true, position: 'top-end', icon: 'warning', title: 'Username and password are required', showConfirmButton: false, timer: 2500 })
        return
      }
      this.creating = true
      try {
        await api.createUser(this.newUser)
        this.$swal({ toast: true, position: 'top-end', icon: 'success', title: `User ${this.newUser.username} created`, showConfirmButton: false, timer: 2000 })
        this.closeNewUser()
        await this.loadUsers()
      } catch (err) {
        this.$swal({ icon: 'error', title: 'Error', text: err.response?.data?.error || 'Failed to create user' })
      } finally {
        this.creating = false
      }
    }
  }
}
</script>

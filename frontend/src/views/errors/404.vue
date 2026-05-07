<template>
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;background:#060c18;padding:2rem">
    <div style="text-align:center;max-width:480px">
      <!-- Glowing 404 number -->
      <div style="font-size:7rem;font-weight:900;line-height:1;color:#0d1321;text-shadow:0 0 60px rgba(74,158,255,0.25);letter-spacing:-4px;user-select:none">
        404
      </div>

      <!-- Divider -->
      <div style="width:60px;height:2px;background:linear-gradient(90deg,transparent,#4a9eff,transparent);margin:1.2rem auto"></div>

      <h4 style="color:#c9d8f0;font-weight:600;margin-bottom:.5rem">Page Not Found</h4>
      <p style="color:#5a7499;font-size:0.88rem;margin-bottom:2rem;line-height:1.6">
        The route <code style="color:#4a9eff;background:rgba(74,158,255,0.1);padding:2px 6px;border-radius:4px">{{ $route.path }}</code>
        doesn't exist. It may have been moved or you followed a broken link.
      </p>

      <div class="d-flex gap-2 justify-content-center">
        <button class="btn btn-sc-primary" @click="$router.push('/dashboard')">
          <i class="mdi mdi-home me-2"></i>Go to Dashboard
        </button>
        <button class="btn" style="background:rgba(74,158,255,0.08);color:#4a9eff;border:1px solid rgba(74,158,255,0.2)" @click="$router.back()">
          <i class="mdi mdi-arrow-left me-1"></i>Go Back
        </button>
      </div>

      <!-- Quick links -->
      <div style="margin-top:2.5rem;padding-top:1.5rem;border-top:1px solid #1e2d4a">
        <p style="font-size:0.75rem;color:#5a7499;margin-bottom:0.75rem;text-transform:uppercase;letter-spacing:.08em">Quick Links</p>
        <div class="d-flex flex-wrap gap-2 justify-content-center">
          <a v-for="link in quickLinks" :key="link.path"
            style="font-size:0.78rem;color:#4a9eff;text-decoration:none;padding:4px 10px;border:1px solid rgba(74,158,255,0.2);border-radius:4px;background:rgba(74,158,255,0.06);transition:background .15s"
            @click.prevent="$router.push(link.path)"
            href="#">
            <i :class="link.icon" class="me-1"></i>{{ link.label }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'NotFoundPage',
  data() {
    return {
      quickLinks: [
        { path: '/dashboard',  icon: 'mdi mdi-view-dashboard-outline', label: 'Dashboard'  },
        { path: '/security',   icon: 'mdi mdi-shield-check-outline',   label: 'Security'   },
        { path: '/firewall',   icon: 'mdi mdi-wall-fire',              label: 'Firewall'   },
        { path: '/monitoring', icon: 'mdi mdi-monitor-eye',            label: 'Monitoring' },
        { path: '/logs',       icon: 'mdi mdi-text-box-outline',       label: 'Logs'       },
        { path: '/alerts',     icon: 'mdi mdi-bell-alert-outline',     label: 'Alerts'     },
      ]
    }
  }
}
</script>

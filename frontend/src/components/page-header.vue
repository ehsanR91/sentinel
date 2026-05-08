<template>
  <header class="page-header-sc sc-surface sc-focus-ring">
    <div class="page-header-main">
      <div class="page-header-title-wrap">
        <div class="page-header-icon" v-if="icon" aria-hidden="true">
          <i :class="icon"></i>
        </div>
        <div>
          <h1 class="page-title">{{ title }}</h1>
          <nav class="page-breadcrumbs" aria-label="Breadcrumb">
            <button type="button" class="page-back-link" @click="goBack">
              <i class="mdi mdi-arrow-left"></i>
              Back
            </button>
            <ol v-if="showBreadcrumbTrail" class="breadcrumb-list">
              <li class="breadcrumb-item">
                <router-link to="/dashboard" class="breadcrumb-link d-flex align-items-center gap-1">
                  <i class="mdi mdi-home-outline"></i>
                  SentinelCore
                </router-link>
              </li>
              <li
                v-for="(item, i) in items"
                :key="i"
                class="breadcrumb-item"
                :class="{ active: item.active }"
              >
                <router-link v-if="item.href && !item.active" :to="item.href" class="breadcrumb-link d-flex align-items-center gap-1">
                  <i v-if="item.icon" :class="item.icon"></i>
                  {{ item.text }}
                </router-link>
                <span v-else class="d-flex align-items-center gap-1">
                  <i v-if="item.icon" :class="item.icon"></i>
                  {{ item.text }}
                </span>
              </li>
            </ol>
            <div v-else class="page-breadcrumb-home">
              <i class="mdi mdi-home-outline" aria-hidden="true"></i>
              <span>SentinelCore</span>
            </div>
          </nav>
        </div>
      </div>
      <div class="page-header-actions">
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<script>
export default {
  name: 'PageHeader',
  props: {
    title: { type: String, required: true },
    items: { type: Array, default: () => [] },
    icon: { type: String, default: '' }
  },
  computed: {
    showBreadcrumbTrail () {
      if (!this.items.length) return false
      if (this.items.length > 1) return true
      const [only] = this.items
      return !only?.active || only?.text !== this.title
    }
  },
  methods: {
    goBack () {
      if (window.history.length > 1) {
        this.$router.back()
      } else {
        this.$router.push('/dashboard')
      }
    }
  }
}
</script>

<style scoped>
.page-header-sc {
  padding: 16px 18px;
}

.page-header-main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-16);
  flex-wrap: wrap;
}

.page-header-title-wrap {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.page-header-icon {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-md);
  background: var(--accent-muted);
  color: var(--accent);
  font-size: 20px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 24px;
  line-height: var(--line-height-tight);
  font-weight: 600;
  color: var(--text-primary);
}

.page-breadcrumbs {
  margin-top: 6px;
}

.page-breadcrumb-home {
  display: inline-flex;
  align-items: center;
  gap: var(--space-6);
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.breadcrumb-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
  align-items: center;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.breadcrumb-item + .breadcrumb-item::before {
  content: "/";
  padding-right: var(--space-8);
  color: var(--text-tertiary);
}

.breadcrumb-item.active {
  color: var(--text-primary);
}

.breadcrumb-link {
  color: var(--text-secondary);
  text-decoration: none;
  transition: color 0.15s ease;
}

.breadcrumb-link:hover {
  color: var(--text-primary);
}

.page-header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
  align-items: center;
  justify-content: flex-end;
}

.page-back-link {
  display: none;
  background: none;
  border: 0;
  padding: 0;
  color: var(--text-secondary);
  font-size: var(--font-size-13);
  align-items: center;
  gap: var(--space-4);
}

@media (max-width: 768px) {
  .page-header-sc {
    padding: 14px 16px;
  }

  .page-title {
    font-size: 20px;
  }

  .page-header-actions {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .page-header-title-wrap {
    gap: var(--space-12);
  }

  .breadcrumb-list {
    display: none;
  }

  .page-back-link {
    display: inline-flex;
  }
}
</style>

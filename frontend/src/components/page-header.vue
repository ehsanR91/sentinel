<template>
  <div class="page-header-sc d-flex align-items-center justify-content-between">
    <div>
      <h4 class="page-title d-flex align-items-center gap-2">
        <i v-if="icon" :class="icon"></i>
        {{ title }}
      </h4>
      <nav>
        <ol class="breadcrumb">
          <li class="breadcrumb-item">
            <router-link to="/dashboard" class="d-flex align-items-center gap-1">
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
            <router-link v-if="item.href && !item.active" :to="item.href" class="d-flex align-items-center gap-1">
              <i v-if="item.icon" :class="item.icon"></i>
              {{ item.text }}
            </router-link>
            <span v-else class="d-flex align-items-center gap-1">
              <i v-if="item.icon" :class="item.icon"></i>
              {{ item.text }}
            </span>
          </li>
        </ol>
      </nav>
    </div>
    <slot name="actions" />
  </div>
</template>

<script>
export default {
  name: 'PageHeader',
  props: {
    title: { type: String, required: true },
    items: { type: Array, default: () => [] },
    icon: { type: String, default: '' }
  }
}
</script>

<style scoped>
.page-header-sc {
  padding: 1rem;
  background: var(--sc-glass-bg-2);
  border: 1px solid var(--sc-glass-border);
  border-radius: 0.85rem;
  backdrop-filter: blur(var(--sc-glass-blur));
  -webkit-backdrop-filter: blur(var(--sc-glass-blur));
}

.page-title {
  margin: 0 0 0.5rem 0;
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--sc-text);
}

.breadcrumb {
  margin: 0;
  padding: 0;
  background: transparent;
  list-style: none;
  display: flex;
  flex-wrap: wrap;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  color: var(--sc-text-muted);
}

.breadcrumb-item + .breadcrumb-item::before {
  content: "/";
  padding: 0 0.5rem;
  color: var(--sc-text-muted);
}

.breadcrumb-item.active {
  color: var(--sc-text);
}

.breadcrumb-item a {
  color: var(--sc-blue);
  text-decoration: none;
  transition: color 0.15s ease;
}

.breadcrumb-item a:hover {
  color: var(--sc-primary);
}

@media (max-width: 768px) {
  .page-header-sc {
    padding: 0.85rem;
    align-items: flex-start !important;
    flex-direction: column;
    gap: 0.75rem;
  }

  .page-title {
    font-size: 1.1rem;
    margin-bottom: 0.35rem;
  }
}
</style>

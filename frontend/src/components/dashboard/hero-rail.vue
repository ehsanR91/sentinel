<template>
  <div class="hero-rail-wrapper">
    <!-- Scroll Left Indicator -->
    <button
      v-show="canScrollLeft"
      class="hero-rail-nav hero-rail-nav--left"
      aria-label="Scroll left"
      @click="scrollBy(-1)"
    >
      <i class="mdi mdi-chevron-left" aria-hidden="true"></i>
    </button>

    <section class="hero-rail-container">
      <div 
        class="hero-rail" 
        ref="railEl"
        @scroll="handleScroll"
      >
        <slot :activeId="activeCardId" :toggle="toggleCard"></slot>
      </div>
    </section>

    <!-- Scroll Right Indicator -->
    <button
      v-show="canScrollRight"
      class="hero-rail-nav hero-rail-nav--right"
      aria-label="Scroll right"
      @click="scrollBy(1)"
    >
      <i class="mdi mdi-chevron-right" aria-hidden="true"></i>
    </button>
  </div>
</template>

<script>
import { ref, provide, onMounted, onUnmounted, nextTick } from 'vue'

export default {
  name: 'HeroCardRail',
  setup() {
    const railEl = ref(null)
    const canScrollLeft = ref(false)
    const canScrollRight = ref(false)

    // Accordion State
    const activeCardId = ref(null)

    const registerCard = (id) => {
      if (!activeCardId.value) {
        activeCardId.value = id
      }
    }

    const toggleCard = (id) => {
      activeCardId.value = activeCardId.value === id ? null : id
    }

    provide('heroRail', {
      activeCardId,
      registerCard,
      toggleCard
    })

    // Scroll Indicators Handlers
    const handleScroll = () => {
      if (!railEl.value) return
      const { scrollLeft, scrollWidth, clientWidth } = railEl.value
      canScrollLeft.value = scrollLeft > 10
      // Allow a small margin of error (2px)
      canScrollRight.value = Math.ceil(scrollLeft + clientWidth) < scrollWidth - 2
    }

    const scrollBy = (direction) => {
      if (!railEl.value) return
      // Scroll by 80% of the visible width
      const amount = railEl.value.clientWidth * 0.8 * direction
      railEl.value.scrollBy({ left: amount, behavior: 'smooth' })
    }

    // Window resize handling
    const resizeObserver = new ResizeObserver(() => {
      handleScroll()
    })

    onMounted(async () => {
      await nextTick()
      if (railEl.value) {
        resizeObserver.observe(railEl.value)
        handleScroll()
      }
    })

    onUnmounted(() => {
      if (railEl.value) {
        resizeObserver.unobserve(railEl.value)
      }
      resizeObserver.disconnect()
    })

    return {
      railEl,
      canScrollLeft,
      canScrollRight,
      handleScroll,
      scrollBy
    }
  }
}
</script>

<style scoped>
.hero-rail-wrapper {
  position: relative;
  width: 100%;
  display: flex;
  align-items: center;
}

.hero-rail-container {
  width: 100%;
  overflow: hidden;
  position: relative;
  border-radius: 24px;
}

.hero-rail {
  display: flex;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: none; /* Firefox */
  -ms-overflow-style: none; /* IE and Edge */
  gap: 16px;
  padding-bottom: 12px; /* Extra space for focus rings/shadows */
  scroll-behavior: smooth;
  align-items: stretch;
}

[data-theme="light"] .hero-rail-container {
  overflow: clip;
}

[data-theme="light"] .hero-rail {
  padding: 1px 1px 8px;
}

.hero-rail::-webkit-scrollbar {
  display: none;
}

/* Ensure children behave like horizontal cards */
.hero-rail > :deep(*) {
  flex: 0 0 85%; /* Default for mobile: cards take 85% width */
  scroll-snap-align: center;
  transition: flex 0.4s cubic-bezier(0.4, 0, 0.2, 1), min-width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  min-width: 0;
}

/* Accordion expanded/collapsed behaviors applied via classes to slots */
.hero-rail > :deep(.is-expanded) {
  flex: 1 0 85%;
}
.hero-rail > :deep(.is-collapsed) {
  flex: 0 0 40%;
  cursor: pointer;
}
.hero-rail > :deep(.is-collapsed:hover) {
  opacity: 0.9;
}

/* Scroll Nav Indicators */
.hero-rail-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10;
  background-color: var(--bg-surface, #ffffff);
  border: 1px solid var(--border-color, #e0e0e0);
  color: var(--text-primary, #333333);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: var(--shadow-md);
  transition: all 0.2s ease;
}

.hero-rail-nav:hover {
  background-color: var(--bg-surface-hover, #f5f5f5);
  box-shadow: var(--shadow-lg);
}

.hero-rail-nav--left {
  left: -20px;
}

.hero-rail-nav--right {
  right: -20px;
}

/* 
  Desktop layout — all three side-by-side at >=1280px; 
  collapse still available.
*/
@media (min-width: 1280px) {
  .hero-rail {
    /* If they fit side-by-side perfectly, disable snap */
    scroll-snap-type: none;
    overflow-x: visible; /* Show entirely rather than scrolling if they fit */
    gap: 24px;
    padding-bottom: 0; /* Remove bottom scroll shadow padding if visible */
  }

  .hero-rail > :deep(*) {
    /* Default unexpanded desktop state is usually equal width */
    flex: 1 1 0;
  }
  
  /* Desktop Accordion layout behavior */
  .hero-rail > :deep(.is-expanded) {
    flex: 2 1 0; /* Expanded takes up twice the space */
  }
  
  .hero-rail > :deep(.is-collapsed) {
    flex: 1 1 0; /* Collapsed takes basic space */
  }

  .hero-rail-nav {
    display: none; /* Turn off scroll buttons on extra large screens */
  }
}
</style>
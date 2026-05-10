<template>
  <div id="page-topbar" :class="{ 'sidebar-collapsed': sidebarCollapsed }">
    <div class="topbar-left">
      <Tooltip :label="sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'" :shortcut="sidebarCollapsed ? ']' : '['" as-child>
        <button
          class="topbar-btn d-none d-lg-flex"
          aria-label="Toggle sidebar"
          @click="toggleCollapse"
        >
          <i class="mdi mdi-menu" aria-hidden="true"></i>
        </button>
      </Tooltip>
      <button class="topbar-btn d-lg-none" aria-label="Toggle sidebar" @click="$emit('toggle-sidebar')">
        <i class="mdi mdi-menu" aria-hidden="true"></i>
      </button>

      <nav aria-label="Breadcrumb" class="topbar-breadcrumb d-none d-md-flex">
        <span class="topbar-breadcrumb__prefix">SentinelCore</span>
        <i class="mdi mdi-chevron-right" aria-hidden="true"></i>
        <span class="topbar-breadcrumb__current">{{ currentPage }}</span>
      </nav>
    </div>

    <div class="topbar-right">
      <div class="topbar-system-cluster" role="group" aria-label="System status and actions">
        <Tooltip label="Live telemetry" :description="liveTooltip" variant="rich" as-child>
          <button
            class="topbar-live-pill sc-focus-ring"
            :class="{ 'is-offline': !wsConnected, 'is-paused': livePaused }"
            :aria-label="liveTooltip"
            @click="toggleLiveState"
          >
            <StatusDot :status="wsConnected && !livePaused ? 'online' : 'offline'" />
            <span>{{ liveLabel }}</span>
          </button>
        </Tooltip>

        <Tooltip v-if="updateAvailable" label="Update available" description="Reload to apply the newest installed assets." variant="rich" as-child>
          <button
            class="topbar-btn"
            aria-label="Update available"
            @click="refreshAppVersion"
          >
            <i class="mdi mdi-update" aria-hidden="true"></i>
          </button>
        </Tooltip>
        <Tooltip v-if="installAvailable" label="Install SentinelCore" description="Install the app for a standalone desktop-like launch experience." variant="rich" as-child>
          <button
            class="topbar-btn"
            aria-label="Install SentinelCore"
            @click="onInstallClick"
          >
            <i class="mdi mdi-download" aria-hidden="true"></i>
          </button>
        </Tooltip>
        <Tooltip v-else-if="showIosInstallHint" label="Add to Home Screen" description="Safari requires the Share sheet to install this app on iOS." variant="rich" as-child>
          <button
            class="topbar-btn"
            aria-label="Add SentinelCore to Home Screen"
            @click="openInstallHelp"
          >
            <i class="mdi mdi-information-outline" aria-hidden="true"></i>
          </button>
        </Tooltip>
        <Tooltip v-if="lockPinSet" label="Lock screen" description="Activate the secure lock overlay." shortcut="Space" variant="rich" as-child>
          <button
            class="topbar-btn"
            aria-label="Lock screen"
            @click="lockScreen"
          >
            <i class="mdi mdi-lock-outline" aria-hidden="true"></i>
          </button>
        </Tooltip>

        <div class="position-relative">
          <Tooltip label="Quick Mount" description="Generate SSH tunnel commands for server-side web apps." variant="rich" as-child>
            <button
              class="topbar-btn topbar-btn--quick-mount d-none d-md-flex"
              aria-label="Quick Mount"
              @click="toggleQuickMount"
            >
              <i class="mdi mdi-lan-connect" aria-hidden="true"></i>
            </button>
          </Tooltip>

          <div v-if="showQuickMount" class="dropdown-menu dropdown-menu-end show quick-mount-panel" style="top:44px;right:0;position:absolute;min-width:420px;max-width:96vw;padding:0">
            <div class="d-flex align-items-center justify-content-between px-3 py-2" style="border-bottom:1px solid var(--sc-border, #1e2d4a)">
              <span style="font-weight:600;font-size:0.82rem;display:flex;align-items:center;gap:6px">
                <i class="mdi mdi-lan-connect" style="color:#22d67c"></i> Quick Mount
              </span>
              <div class="d-flex align-items-center gap-2">
                <Tooltip label="Quick Mount help" description="See how local tunnel commands are constructed and used." variant="rich" as-child>
                  <button class="btn btn-sm p-0" style="color:#4a9eff;font-size:0.72rem" @click="openQuickMountHelp">
                    <i class="mdi mdi-help-circle-outline"></i>
                  </button>
                </Tooltip>
                <button class="btn btn-sm p-0" style="color:#5a7499;font-size:0.72rem" :disabled="loadingTunnels" @click="refreshTunnelApps">
                  <i :class="`mdi mdi-refresh${loadingTunnels ? ' mdi-spin' : ''}`"></i>
                </button>
                <button class="btn btn-sm p-0" style="color:#5a7499" @click="showQuickMount = false"><i class="mdi mdi-close"></i></button>
              </div>
            </div>

            <div class="px-3 py-2" style="background:rgba(74,158,255,0.04);border-bottom:1px solid var(--sc-border, #1e2d4a);font-size:0.72rem">
              <div class="d-flex align-items-center gap-2 flex-wrap">
                <span style="color:#5a7499;white-space:nowrap">SSH user</span>
                <input v-model="mountSshUser" type="text" class="form-control form-control-sm" style="width:90px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="detectedSshUser || 'user'">
                <span style="color:#5a7499;white-space:nowrap">port</span>
                <input v-model="mountSshPort" type="text" class="form-control form-control-sm" style="width:55px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="String(detectedSshPort || 22)">
                <span style="color:#5a7499;white-space:nowrap">host</span>
                <input v-model="mountSshHost" type="text" class="form-control form-control-sm" style="width:130px;height:24px;font-size:0.72rem;font-family:monospace;padding:1px 6px" :placeholder="detectedSshHost || serverHost">
              </div>
              <div class="mt-1" style="font-size:0.65rem;color:#5a7499">
                Auto-detected: <code style="color:#8aa4c8">{{ effectiveSshUser }}</code>@<code style="color:#8aa4c8">{{ effectiveSshHost }}</code> -p <code style="color:#8aa4c8">{{ effectiveSshPort }}</code>
                <span v-if="detectedSshUserSource" style="margin-left:6px;opacity:0.85">({{ detectedSshUserSource }})</span>
              </div>
            </div>

            <div style="max-height:360px;overflow-y:auto;padding:8px 0">
              <div v-if="loadingTunnels" class="text-center py-4" style="font-size:0.78rem;color:#5a7499">
                <span class="spinner-border spinner-border-sm me-1"></span>Detecting apps…
              </div>
              <div v-else-if="!tunnelApps.length" class="text-center py-4 px-3" style="font-size:0.78rem;color:#5a7499">
                <i class="mdi mdi-lan-disconnect d-block mb-1" style="font-size:1.4rem;opacity:0.4"></i>
                No web-accessible apps detected.<br>
                <span style="font-size:0.68rem">Grafana, Portainer, Prometheus, Netdata and similar apps appear here when running.</span>
              </div>
              <template v-else>
                <div v-for="cat in mountCategories" :key="cat">
                  <div class="px-3 py-1" style="font-size:0.65rem;font-weight:600;color:#5a7499;text-transform:uppercase;letter-spacing:.06em">{{ cat }}</div>
                  <div
                    v-for="app in tunnelApps.filter(a => a.category === cat)"
                    :key="`${app.name}-${app.port}`"
                    class="d-flex align-items-center justify-content-between px-3 py-2 quick-mount-row"
                  >
                    <div class="d-flex align-items-center gap-2" style="min-width:0">
                      <i :class="`mdi ${app.icon}`" :style="`color:${app.color};font-size:1.05rem;flex-shrink:0`"></i>
                      <div style="min-width:0">
                        <div style="font-size:0.78rem;font-weight:500;color:var(--sc-text, #c9d8f0)">{{ app.name }}</div>
                        <code style="font-size:0.65rem;color:#5a7499">localhost:{{ app.port }} → server:{{ app.port }}</code>
                      </div>
                      <span class="badge ms-1" :style="`background:rgba(${app.source==='docker'?'36,150,237':'34,214,124'},0.1);color:${app.source==='docker'?'#2496ed':'#22d67c'};font-size:0.58rem;padding:1px 5px`">{{ app.source }}</span>
                    </div>
                    <div class="d-flex align-items-center gap-1 ms-2">
                      <button
                        class="btn btn-sm flex-shrink-0"
                        style="background:rgba(245,166,35,0.08);color:#f5a623;border:1px solid rgba(245,166,35,0.2);font-size:0.65rem;white-space:nowrap;padding:2px 8px"
                        :disabled="grantingPort === app.port"
                        @click="grantAccess(app)"
                      >
                        <i :class="`mdi mdi-${grantingPort===app.port?'loading mdi-spin':'shield-key-outline'} me-1`"></i>Grant Access
                      </button>
                      <button
                        class="btn btn-sm flex-shrink-0"
                        :style="`background:rgba(34,214,124,0.07);color:${mountCopied===app.name+app.port?'#22d67c':'#5a7499'};border:1px solid rgba(34,214,124,0.15);font-size:0.65rem;white-space:nowrap;padding:2px 8px`"
                        @click="copyMount(app)"
                      >
                        <i :class="`mdi mdi-${mountCopied===app.name+app.port?'check':'content-copy'} me-1`"></i>{{ mountCopied === app.name + app.port ? 'Copied!' : 'Copy' }}
                      </button>
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <div class="px-3 py-2" style="border-top:1px solid var(--sc-border, #1e2d4a);font-size:0.67rem;color:#5a7499">
              <i class="mdi mdi-information-outline me-1"></i>Run the copied command on <strong style="color:#8aa4c8">your local machine</strong>, then open <code style="color:#4a9eff">http://localhost:&lt;port&gt;</code>.
            </div>
          </div>
        </div>
      </div>

      <div class="position-relative me-1 d-none d-md-block topbar-search">
        <div class="search-input-wrapper">
          <i class="mdi mdi-magnify search-icon"></i>
          <input
            v-model="searchQuery"
            type="text"
            class="search-input"
            placeholder="Search…"
            @focus="showSearchResults = true"
            @keydown.esc="showSearchResults = false; searchQuery = ''"
            @keydown.down.prevent="navigateSearch(1)"
            @keydown.up.prevent="navigateSearch(-1)"
            @keydown.enter.prevent="selectSearchResult"
          >
        </div>
        <div v-if="showSearchResults && searchResults.length > 0" class="search-results-dropdown">
          <div class="search-results-header">Quick Navigation</div>
          <button
            v-for="(result, idx) in searchResults"
            :key="`${result.group}-${result.route}`"
            type="button"
            class="search-result-item"
            :class="{ active: searchActiveIndex === idx }"
            @mouseenter="searchActiveIndex = idx"
            @click="navigateToSearchResult(result)"
          >
            <i :class="result.icon || 'mdi mdi-compass-outline'"></i>
            <span>{{ result.label }}</span>
            <span class="search-result-path">{{ result.group }}</span>
          </button>
        </div>
      </div>

      <Popover
        :model-value="openPopover === 'alerts'"
        title="Alerts"
        subtitle="Realtime feed"
        :width="460"
        :max-width="520"
        panel-class="topbar-popover topbar-popover--alerts"
        body-class="topbar-popover__body"
        @update:modelValue="setPopoverState('alerts', $event)"
        @after-open="handleAlertsOpened"
      >
        <template #trigger="{ toggle, triggerRef, triggerAttrs, open }">
          <Tooltip label="Alerts" :description="`${bellUnreadCount} unread alerts`" variant="rich" as-child>
            <button
              :ref="triggerRef"
              class="topbar-btn topbar-btn--bell sc-focus-ring"
              :class="{ 'is-active': open, 'has-critical-pulse': bellPulseActive }"
              v-bind="triggerAttrs"
              aria-label="Alerts"
              @click="toggle"
            >
              <i class="mdi mdi-bell-outline" aria-hidden="true"></i>
              <span v-if="bellUnreadCount" class="topbar-badge" :class="`topbar-badge--${alertBadgeTone}`">{{ badgeCountLabel }}</span>
            </button>
          </Tooltip>
        </template>

        <div class="topbar-alerts" @click="noop">
          <div class="topbar-alerts__meta">
            <div class="topbar-alerts__summary">
              <strong>{{ unreadCount }} unread</strong>
              <span>{{ alertVisibleGroups.length }} visible</span>
            </div>
            <button v-if="unreadCount" type="button" class="topbar-link-button" @click="markAllAsRead">Mark all read</button>
          </div>

          <div class="visually-hidden" aria-live="polite">{{ alertAnnouncement }}</div>

          <div class="topbar-tabs" role="tablist" aria-label="Alert filters">
            <button
              v-for="tab in alertTabs"
              :key="tab.id"
              type="button"
              class="topbar-tab sc-focus-ring"
              :class="{ active: alertTab === tab.id }"
              role="tab"
              :aria-selected="String(alertTab === tab.id)"
              @click="alertTab = tab.id"
            >
              <span>{{ tab.label }}</span>
              <strong>{{ tab.count }}</strong>
            </button>
          </div>

          <button v-if="queuedAlerts.length" type="button" class="topbar-new-pill sc-focus-ring" @click="applyQueuedAlerts">
            ↑ {{ queuedAlerts.length }} new
          </button>

          <div ref="alertList" class="topbar-alerts__list" @scroll="onAlertListScroll">
            <div v-if="loadingAlerts" class="topbar-state topbar-state--loading">
              <span class="spinner-border spinner-border-sm"></span>
              <span>Loading alerts…</span>
            </div>
            <EmptyState
              v-else-if="alertError"
              icon="mdi mdi-alert-circle-outline"
              title="Alert feed unavailable"
              :description="alertError"
            >
              <template #actions>
                <AppButton variant="secondary" size="sm" icon="mdi mdi-refresh" label="Retry" @click="syncAlerts(true)" />
              </template>
            </EmptyState>
            <EmptyState
              v-else-if="!alertVisibleGroups.length"
              icon="mdi mdi-bell-off-outline"
              title="No alerts here"
              description="New system and security activity will appear here when it arrives."
            />
            <template v-else>
              <div class="topbar-alerts__items" :style="virtualAlertWrapperStyle">
                <article
                  v-for="group in visibleAlertGroups"
                  :key="group.id"
                  class="topbar-alert-row sc-focus-ring"
                  :class="{ 'is-unread': !group.read }"
                  @click="openAlertGroup(group)"
                >
                  <div class="topbar-alert-row__icon" :class="`severity-${severityTone(group.base.severity)}`">
                    <i :class="severityIcon(group.base.severity)" aria-hidden="true"></i>
                  </div>
                  <div class="topbar-alert-row__content">
                    <div class="topbar-alert-row__head">
                      <div class="topbar-alert-row__summary">
                        <div class="topbar-alert-row__title">{{ summarizeAlert(group.base) }}</div>
                        <div class="topbar-alert-row__subtitle">{{ alertSubtitle(group) }}</div>
                      </div>
                      <div class="topbar-alert-row__meta-actions" @click.stop>
                        <span v-if="group.count > 1" class="topbar-count-pill">{{ group.count }}</span>
                        <span v-if="!group.read" class="topbar-unread-dot" aria-hidden="true"></span>
                        <details class="topbar-row-menu">
                          <summary class="topbar-row-menu__trigger sc-focus-ring" aria-label="Alert actions">
                            <i class="mdi mdi-dots-horizontal"></i>
                          </summary>
                          <div class="topbar-row-menu__body">
                            <button type="button" class="dropdown-item" @click="markGroupRead(group)">Mark read</button>
                            <button type="button" class="dropdown-item" @click="dismissGroup(group)">Dismiss</button>
                            <button type="button" class="dropdown-item" @click="snoozeGroup(group, 5)">Snooze 5m</button>
                            <button type="button" class="dropdown-item" @click="snoozeGroup(group, 60)">Snooze 1h</button>
                            <button type="button" class="dropdown-item" @click="muteRule(group)">Mute rule</button>
                            <button type="button" class="dropdown-item" :disabled="!group.base.ip" @click="blockSourceIp(group)">Block IP</button>
                            <button type="button" class="dropdown-item" @click="copyAlertJson(group)">Copy JSON</button>
                          </div>
                        </details>
                      </div>
                    </div>
                    <button
                      v-if="group.count > 1"
                      type="button"
                      class="topbar-link-button topbar-link-button--small"
                      @click.stop="toggleAlertGroup(group.id)"
                    >
                      {{ expandedAlertGroups[group.id] ? 'Hide repeats' : `Show ${group.count - 1} repeats` }}
                    </button>
                    <div v-if="expandedAlertGroups[group.id]" class="topbar-alert-expansion">
                      <div v-for="item in group.items" :key="item.id" class="topbar-alert-expansion__item">
                        <span>{{ summarizeAlert(item) }}</span>
                        <time>{{ timeAgo(item.ts) }}</time>
                      </div>
                    </div>
                  </div>
                </article>
              </div>
            </template>
          </div>

          <div class="topbar-alerts__footer">
            <button type="button" class="topbar-link-button" @click="openAlertsPage">Open full Alerts page</button>
          </div>
        </div>
      </Popover>

      <Popover
        :model-value="openPopover === 'user'"
        :width="280"
        :min-width="280"
        :max-width="280"
        panel-class="topbar-popover topbar-popover--user"
        body-class="topbar-popover__body topbar-popover__body--user"
        @update:modelValue="setPopoverState('user', $event)"
        @after-open="handleUserMenuOpened"
      >
        <template #trigger="{ toggle, triggerRef, triggerAttrs, open }">
          <button
            :ref="triggerRef"
            class="user-menu sc-focus-ring"
            :class="{ 'is-active': open }"
            v-bind="triggerAttrs"
            aria-label="User menu"
            @click="toggle"
          >
            <UserAvatar :name="displayName" :src="avatarUrl" :status="presence.status" size="md" />
            <span class="user-name-wrap d-none d-xl-flex">
              <Tooltip :label="displayName" as-child>
                <span class="user-name">{{ displayName }}</span>
              </Tooltip>
              <span class="user-role">{{ userRole }} · {{ presenceDisplayLabel }}</span>
            </span>
            <i class="mdi mdi-chevron-down d-none d-xl-inline" aria-hidden="true"></i>
          </button>
        </template>

        <div v-if="userMenuView === 'main'" class="user-popover">
          <header class="user-popover__header">
            <div class="user-popover__hero">
              <UserAvatar :name="displayName" :src="avatarUrl" :status="presence.status" size="sm" />
              <div class="user-popover__meta">
                <Tooltip :label="displayName" as-child>
                  <strong>{{ displayName }}</strong>
                </Tooltip>
                <span class="user-popover__meta-line">
                  <StatusDot :status="presence.status" />
                  <span>{{ userRole }} · {{ presenceDisplayLabel }}</span>
                </span>
                <button type="button" class="user-popover__last-login" :title="lastLoginLabel" @click="openAuditForCurrentUser">
                  {{ lastLoginLabel }}
                </button>
              </div>
            </div>
          </header>

          <div class="user-popover__scroll-area">
            <section class="user-popover__section">
              <div class="user-popover__section-label">Account</div>
              <button type="button" class="user-popover__item sc-focus-ring" @click="goToAccount">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-account-circle-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Profile</span>
                </span>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="goToSettings('general')">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-cog-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Settings</span>
                </span>
                <span class="user-popover__item-meta">⌘,</span>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="openTokensInfo">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-key-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">API tokens</span>
                </span>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="goToSettings('access-control')">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-shield-key-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">2FA management</span>
                </span>
                <span class="user-popover__item-meta">{{ me.totp_enabled ? 'On' : 'Off' }}</span>
              </button>
            </section>

            <section class="user-popover__section">
              <div class="user-popover__section-label">Workspace</div>
              <button type="button" class="user-popover__item sc-focus-ring" @click="userMenuView = 'servers'">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-server-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Switch server</span>
                </span>
                <i class="mdi mdi-chevron-right user-popover__item-meta-icon" aria-hidden="true"></i>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="openAuditForCurrentUser">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-file-document-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Audit logs</span>
                </span>
              </button>
            </section>

            <section class="user-popover__section">
              <div class="user-popover__section-label">Preferences</div>
              <div class="user-popover__setting-row">
                <span class="user-popover__setting-label">Theme</span>
                <SegmentedControl :model-value="currentThemePref" :options="themeSegmentOptions" label="Theme" @update:modelValue="applyTheme" />
              </div>
              <div class="user-popover__setting-row">
                <span class="user-popover__setting-label">Density</span>
                <SegmentedControl :model-value="sidebarDensity" :options="densitySegmentOptions" label="Density" @update:modelValue="applyDensity" />
              </div>
              <div class="user-popover__setting-row">
                <span class="user-popover__setting-label">Status</span>
                <SegmentedControl :model-value="presence.status" :options="presenceSegmentOptions" label="Presence" @update:modelValue="applyPresence" />
              </div>
              <div class="user-popover__setting-row user-popover__setting-row--stacked">
                <span class="user-popover__setting-label">Auto-away</span>
                <SegmentedControl :model-value="presence.autoAwayMinutes" :options="autoAwayOptions" label="Auto-away" @update:modelValue="updateAutoAway" />
              </div>
            </section>

            <section class="user-popover__section">
              <div class="user-popover__section-label">Help</div>
              <button type="button" class="user-popover__item sc-focus-ring" @click="showShortcutHelp">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-keyboard-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Keyboard shortcuts</span>
                </span>
                <span class="user-popover__item-meta">⌘K</span>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="openWhatsNew">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-star-four-points-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">What’s new</span>
                </span>
                <span v-if="!whatsNewSeen" class="user-popover__item-meta">New</span>
              </button>
              <button type="button" class="user-popover__item sc-focus-ring" @click="openDocs">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-help-circle-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Help & docs</span>
                </span>
              </button>
              <button v-if="lockPinSet" type="button" class="user-popover__item sc-focus-ring" @click="lockScreen">
                <span class="user-popover__item-main">
                  <i class="mdi mdi-lock-outline" aria-hidden="true"></i>
                  <span class="user-popover__item-label">Lock screen</span>
                </span>
                <span class="user-popover__item-meta">Space</span>
              </button>
            </section>
          </div>

          <button type="button" class="user-popover__signout sc-focus-ring" @click="confirmSignOut">
            <i class="mdi mdi-logout"></i>
            <span>Sign out</span>
          </button>
        </div>

        <div v-else class="user-popover user-popover--servers">
          <div class="user-popover__subheader">
            <button type="button" class="topbar-link-button" @click="userMenuView = 'main'">
              <i class="mdi mdi-chevron-left"></i>
              Back
            </button>
            <strong>Switch server</strong>
            <button type="button" class="topbar-link-button" @click="promptAddServer">Add server</button>
          </div>
          <div class="user-popover__server-search">
            <i class="mdi mdi-magnify"></i>
            <input v-model.trim="serverSearch" class="sc-focus-ring" type="search" placeholder="Search servers">
          </div>
          <div class="user-popover__server-list">
            <button
              v-for="server in filteredServerEntries"
              :key="server.url"
              type="button"
              class="user-popover__server-item sc-focus-ring"
              :class="{ active: server.url === currentOrigin }"
              @click="switchServer(server)"
            >
              <div>
                <div class="user-popover__server-head">
                  <strong>{{ server.name }}</strong>
                  <div class="user-popover__server-health">
                    <StatusDot :status="server.statusDot" />
                    <span>{{ server.healthLabel }}</span>
                  </div>
                </div>
                <p>{{ server.url }}</p>
                <div class="user-popover__server-stats">
                  <span>CPU {{ server.metrics.cpu }}</span>
                  <span>RAM {{ server.metrics.ram }}</span>
                </div>
              </div>
              <i class="mdi mdi-chevron-right"></i>
            </button>
            <button v-if="linkedAccounts.length" type="button" class="user-popover__add-account sc-focus-ring" @click="openAccountSwitcher">
              Add account
            </button>
          </div>
        </div>
      </Popover>
    </div>

    <Teleport to="body">
      <div v-if="showQuickMountHelp" class="quick-mount-help-backdrop" @click.self="showQuickMountHelp = false">
        <div class="quick-mount-help-modal">
          <div class="quick-mount-help-header">
            <h6 class="mb-0 d-flex align-items-center gap-2">
              <i class="mdi mdi-help-circle-outline" style="color:#4a9eff"></i>
              How To Use Quick Mount
            </h6>
            <button class="btn btn-sm p-0" style="color:#5a7499" @click="showQuickMountHelp = false">
              <i class="mdi mdi-close"></i>
            </button>
          </div>
          <div class="quick-mount-help-body">
            <p style="font-size:0.82rem;color:#8aa4c8;margin-bottom:10px">
              Quick Mount creates an SSH local tunnel so apps running on your server become reachable from your computer.
            </p>
            <ol class="quick-mount-help-steps">
              <li>Set your SSH connection values in Quick Mount: <code>user</code>, <code>port</code>, and <code>host</code>.</li>
              <li>Click <strong>Copy</strong> next to any app. You will get a command like <code>{{ mountExampleCommand }}</code>.</li>
              <li>Paste and run that command in your local terminal.</li>
              <li>Keep that SSH session open, then visit <code>http://localhost:&lt;port&gt;</code>.</li>
            </ol>
            <div class="quick-mount-help-note">
              <i class="mdi mdi-shield-lock-outline"></i>
              <span>The tunnel is encrypted over SSH and closes automatically when you end the SSH session.</span>
            </div>
            <div class="quick-mount-help-example">
              <div style="font-size:0.7rem;color:#5a7499;margin-bottom:5px">Example command</div>
              <code>{{ mountExampleCommand }}</code>
            </div>
          </div>
          <div class="quick-mount-help-footer">
            <button class="btn btn-sm" style="background:rgba(34,214,124,0.08);color:#22d67c;border:1px solid rgba(34,214,124,0.2)" @click="copyMountExample">
              <i :class="`mdi mdi-${mountHelpCopied ? 'check' : 'content-copy'} me-1`"></i>{{ mountHelpCopied ? 'Copied' : 'Copy Example Command' }}
            </button>
            <button class="btn btn-sm btn-sc-primary" @click="showQuickMountHelp = false">Got it</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showPwaInstallModal" class="pwa-install-backdrop" @click.self="closeInstallHelp">
        <div class="pwa-install-modal">
          <div class="pwa-install-header">
            <h5><i class="mdi mdi-share-variant" style="color:#4a9eff"></i> Add SentinelCore to Home Screen</h5>
            <button class="btn btn-sm p-0" style="color:#5a7499" @click="closeInstallHelp"><i class="mdi mdi-close"></i></button>
          </div>
          <div class="pwa-install-body">
            <p>Safari does not support the native install prompt. Use the Share button and choose <strong>Add to Home Screen</strong>.</p>
            <ol>
              <li>Tap the <strong>Share</strong> button in Safari.</li>
              <li>Scroll and select <strong>Add to Home Screen</strong>.</li>
              <li>Tap <strong>Add</strong> to install SentinelCore.</li>
            </ol>
            <p class="text-muted">Once installed, the app launches as a standalone experience with offline support.</p>
          </div>
          <div class="pwa-install-footer">
            <button class="btn btn-sm btn-sc-primary" @click="closeInstallHelp">Got it</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import api from '@/services/api'
import { pwaState, promptInstall, reloadApp } from '@/plugins/pwa'
import { useMetricsStore } from '@/stores/metrics'
import Popover from '@/components/ui/popover.vue'
import Tooltip from '@/components/ui/tooltip.vue'
import AppButton from '@/components/ui/app-button.vue'
import EmptyState from '@/components/ui/empty-state.vue'
import StatusDot from '@/components/ui/status-dot.vue'
import UserAvatar from '@/components/ui/user-avatar.vue'
import SegmentedControl from '@/components/settings/fields/segmented-control.vue'
import { navigationSearchEntries, settingsCommandEntries } from '@/components/menu'
import { formatAlertMeta, groupAlerts, summarizeAlert } from '@/utils/formatters'
import { getUserPresence, setUserPresence } from '@/utils/user-presence'

const SERVERS_KEY = 'sidebar:servers'
const SERVER_META_KEY = 'topbar:server-meta'
const WHATS_NEW_KEY = 'topbar:whats-new-seen'
const ALERT_MUTES_KEY = 'topbar:muted-rules'
const ALERT_SNOOZE_KEY = 'topbar:snoozed-rules'
const AUTH_CHANNEL = 'sentinel-auth'

function safeParse(value, fallback) {
  try {
    return JSON.parse(value ?? '')
  } catch {
    return fallback
  }
}

function uniqueBy(items, keyFn) {
  const seen = new Set()
  return items.filter(item => {
    const key = keyFn(item)
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

function mergeAlerts(nextAlerts, currentAlerts) {
  const map = new Map(currentAlerts.map(alert => [alert.id, alert]))
  nextAlerts.forEach(alert => {
    map.set(alert.id, { ...map.get(alert.id), ...alert })
  })
  return Array.from(map.values()).sort((left, right) => (right.ts || 0) - (left.ts || 0))
}

function alertRuleKey(alert) {
  return [alert.type || '', alert.source || '', summarizeAlert(alert)].join('::')
}

export default {
  name: 'Topbar',
  setup() {
    return {
      metricsStore: useMetricsStore()
    }
  },
  components: {
    Popover,
    Tooltip,
    AppButton,
    EmptyState,
    StatusDot,
    UserAvatar,
    SegmentedControl
  },
  emits: ['toggle-sidebar'],
  data() {
    return {
      openPopover: '',
      searchQuery: '',
      showSearchResults: false,
      searchActiveIndex: -1,
      showQuickMount: false,
      showQuickMountHelp: false,
      showPwaInstallModal: false,
      mountHelpCopied: false,
      tunnelApps: [],
      loadingTunnels: false,
      mountCopied: '',
      grantingPort: null,
      mountClientIp: '',
      mountSshUser: '',
      mountSshPort: '22',
      mountSshHost: '',
      detectedSshUser: '',
      detectedSshUserSource: '',
      detectedSshPort: 22,
      detectedSshHost: '',
      alerts: [],
      queuedAlerts: [],
      expandedAlertGroups: {},
      loadingAlerts: false,
      alertError: '',
      alertTab: 'all',
      alertScrollTop: 0,
      alertViewportHeight: 360,
      alertPollTimer: null,
      bellPulseActive: false,
      bellPulseTimer: null,
      alertAnnouncement: '',
      userMenuView: 'main',
      serverSearch: '',
      me: { username: '', role: '', email: '', client_ip: '', totp_enabled: false },
      lastLoginEntry: null,
      presence: { status: 'online', autoAwayMinutes: 15, avatarUrl: '' },
      presenceAutoChanged: false,
      lastActivityAt: Date.now(),
      presenceInterval: null,
      whatsNewSeen: JSON.parse(localStorage.getItem(WHATS_NEW_KEY) || 'false'),
      knownServers: safeParse(localStorage.getItem(SERVERS_KEY), []),
      serverMetaCache: safeParse(localStorage.getItem(SERVER_META_KEY), {}),
      linkedAccounts: safeParse(localStorage.getItem('sc_linked_accounts'), []),
      livePaused: false,
      authChannel: null,
      handleKeyboardShortcuts: null,
      onActivity: null,
      abortController: null
    }
  },
  computed: {
    presenceDisplayLabel() {
      const map = { online: 'Online', away: 'Away', dnd: 'DND' }
      return map[this.presence?.status] || this.presence?.status || ''
    },
    sidebarCollapsed() {
      return this.$store.state.layout.sidebarCollapsed
    },
    sidebarDensity() {
      return this.$store.state.layout.sidebarDensity
    },
    wsConnected() {
      return this.metricsStore.wsConnected
    },
    liveSummary() {
      return this.metricsStore.liveSummary || { unreadAlerts: 0, activeBans: 0 }
    },
    metricsSnap() {
      return this.metricsStore.snap || {}
    },
    currentThemePref() {
      return this.$store.state.layout.theme
    },
    currentUser() {
      return this.$store.getters['auth/user'] || safeParse(sessionStorage.getItem('sc_user'), {})
    },
    loginAt() {
      return Number(this.currentUser?.loginAt || 0)
    },
    currentUsername() {
      return this.me.username || this.currentUser?.username || 'admin'
    },
    displayName() {
      return this.currentUsername
    },
    userRole() {
      return String(this.me.role || this.currentUser?.role || 'operator').toLowerCase()
    },
    avatarUrl() {
      return this.presence.avatarUrl || ''
    },
    currentPage() {
      return this.$route.meta?.title || 'SentinelCore'
    },
    lockPinSet() {
      return this.$store.getters['lock/lockPinSet']
    },
    installAvailable() {
      return pwaState.isSupported && pwaState.installRequested && !pwaState.installed && !pwaState.isStandalone && !pwaState.isIos
    },
    showIosInstallHint() {
      return pwaState.isIos && !pwaState.isStandalone
    },
    updateAvailable() {
      return pwaState.updateAvailable
    },
    serverHost() {
      return window.location.hostname
    },
    currentOrigin() {
      return window.location.origin
    },
    currentServerLabel() {
      return this.metricsSnap.hostname || this.serverHost
    },
    mountCategories() {
      return [...new Set(this.tunnelApps.map(a => a.category))]
    },
    effectiveSshUser() {
      return this.mountSshUser || this.detectedSshUser || 'user'
    },
    effectiveSshHost() {
      return this.mountSshHost || this.detectedSshHost || this.serverHost
    },
    effectiveSshPort() {
      const port = Number.parseInt(this.mountSshPort, 10)
      return !Number.isNaN(port) && port > 0 && port <= 65535 ? port : (this.detectedSshPort || 22)
    },
    mountExampleCommand() {
      const portFlag = String(this.effectiveSshPort) !== '22' ? ` -p ${this.effectiveSshPort}` : ''
      return `ssh -L 9090:localhost:9090${portFlag} ${this.effectiveSshUser}@${this.effectiveSshHost}`
    },
    searchResults() {
      const query = this.searchQuery.trim().toLowerCase()
      if (!query) return []
      const entries = [
        ...navigationSearchEntries(),
        ...settingsCommandEntries.map(entry => ({
          label: entry.label,
          route: entry.route,
          icon: 'mdi mdi-cog-outline',
          group: entry.group,
          keywords: entry.keywords
        }))
      ]
      return uniqueBy(
        entries.filter(entry => {
          const haystack = `${entry.label} ${entry.route} ${entry.group} ${(entry.keywords || []).join(' ')}`.toLowerCase()
          return haystack.includes(query)
        }),
        entry => `${entry.group}:${entry.route}`
      ).slice(0, 8)
    },
    liveLabel() {
      if (this.livePaused) return 'Paused'
      return this.wsConnected ? 'Live' : 'Offline'
    },
    liveTooltip() {
      const lastTick = this.metricsSnap.ts ? this.timeAgo(this.metricsSnap.ts) : 'No data yet'
      if (this.livePaused) return `Live updates paused. Last tick ${lastTick}. Click to resume.`
      return `${this.wsConnected ? 'Connected' : 'Disconnected'} · Last tick ${lastTick}. Click to ${this.livePaused ? 'resume' : 'pause'}.`
    },
    unreadCount() {
      return this.visibleAlerts.filter(alert => !alert.read).length
    },
    bellUnreadCount() {
      return this.wsConnected ? Number(this.liveSummary.unreadAlerts || 0) : this.unreadCount
    },
    badgeCountLabel() {
      return this.bellUnreadCount >= 10 ? '9+' : String(this.bellUnreadCount)
    },
    alertBadgeTone() {
      const unread = this.visibleAlerts.filter(alert => !alert.read)
      if (unread.some(alert => ['emergency', 'critical'].includes(alert.severity))) return 'critical'
      if (unread.some(alert => alert.severity === 'warning')) return 'warn'
      if (this.bellUnreadCount > 0) return 'warn'
      return 'neutral'
    },
    visibleAlerts() {
      return this.alerts.filter(alert => !this.isRuleSuppressed(alert))
    },
    alertTabs() {
      const counts = {
        all: this.visibleAlerts.length,
        unread: this.visibleAlerts.filter(alert => !alert.read).length,
        critical: this.visibleAlerts.filter(alert => ['emergency', 'critical'].includes(alert.severity)).length,
        warning: this.visibleAlerts.filter(alert => alert.severity === 'warning').length
      }
      return [
        { id: 'all', label: 'All', count: counts.all },
        { id: 'unread', label: 'Unread', count: counts.unread },
        { id: 'critical', label: 'Critical', count: counts.critical },
        { id: 'warning', label: 'Warning', count: counts.warning }
      ]
    },
    groupedAlertGroups() {
      return groupAlerts(this.visibleAlerts)
    },
    alertVisibleGroups() {
      return this.groupedAlertGroups.filter(group => {
        if (this.alertTab === 'unread') return !group.read
        if (this.alertTab === 'critical') return ['emergency', 'critical'].includes(group.base.severity)
        if (this.alertTab === 'warning') return group.base.severity === 'warning'
        return true
      })
    },
    shouldVirtualizeAlerts() {
      return !Object.keys(this.expandedAlertGroups).some(key => this.expandedAlertGroups[key]) && this.alertVisibleGroups.length > 40
    },
    virtualItemHeight() {
      return 94
    },
    virtualRange() {
      if (!this.shouldVirtualizeAlerts) {
        return { start: 0, end: this.alertVisibleGroups.length }
      }
      const visibleCount = Math.ceil(this.alertViewportHeight / this.virtualItemHeight) + 6
      const start = Math.max(0, Math.floor(this.alertScrollTop / this.virtualItemHeight) - 3)
      return {
        start,
        end: Math.min(this.alertVisibleGroups.length, start + visibleCount)
      }
    },
    visibleAlertGroups() {
      return this.shouldVirtualizeAlerts
        ? this.alertVisibleGroups.slice(this.virtualRange.start, this.virtualRange.end)
        : this.alertVisibleGroups
    },
    virtualAlertWrapperStyle() {
      if (!this.shouldVirtualizeAlerts) return {}
      const top = this.virtualRange.start * this.virtualItemHeight
      const bottom = Math.max(0, (this.alertVisibleGroups.length - this.virtualRange.end) * this.virtualItemHeight)
      return {
        paddingTop: `${top}px`,
        paddingBottom: `${bottom}px`
      }
    },
    lastLoginLabel() {
      if (this.lastLoginEntry) {
        return `Last login ${this.timeAgo(this.lastLoginEntry.ts)} from ${this.lastLoginEntry.ip || this.me.client_ip || 'unknown IP'}`
      }
      if (this.loginAt) {
        return `Signed in ${this.timeAgo(Math.floor(this.loginAt / 1000))} from ${this.me.client_ip || 'unknown IP'}`
      }
      return 'Last login not available yet'
    },
    themeSegmentOptions() {
      return [
        { label: 'Light', value: 'light' },
        { label: 'Dark', value: 'dark' },
        { label: 'System', value: 'system' }
      ]
    },
    densitySegmentOptions() {
      return [
        { label: 'Cozy', value: 'comfortable' },
        { label: 'Compact', value: 'compact' }
      ]
    },
    presenceSegmentOptions() {
      return [
        { label: 'Online', value: 'online' },
        { label: 'Away', value: 'away' },
        { label: 'DND', value: 'dnd' }
      ]
    },
    autoAwayOptions() {
      return [
        { label: '5m', value: 5 },
        { label: '15m', value: 15 },
        { label: '30m', value: 30 },
        { label: '60m', value: 60 }
      ]
    },
    filteredServerEntries() {
      const query = this.serverSearch.trim().toLowerCase()
      return this.serverEntries.filter(server => {
        if (!query) return true
        return `${server.name} ${server.url}`.toLowerCase().includes(query)
      })
    },
    serverEntries() {
      const baseEntries = this.knownServers.length
        ? this.knownServers
        : [{ id: 'current', name: this.currentServerLabel, url: this.currentOrigin }]
      return uniqueBy(baseEntries, entry => entry.url).map(server => {
        const meta = this.serverMetaCache[server.url] || {}
        const isCurrent = server.url === this.currentOrigin
        const metrics = isCurrent
          ? {
              cpu: `${Number(this.metricsSnap.cpu_pct || 0).toFixed(0)}%`,
              ram: `${Number(this.metricsSnap.ram_pct || 0).toFixed(0)}%`
            }
          : {
              cpu: meta.cpu || '—',
              ram: meta.ram || '—'
            }
        const healthState = isCurrent ? (meta.healthState || (this.wsConnected ? 'online' : 'offline')) : (meta.healthState || 'offline')
        return {
          ...server,
          metrics,
          healthLabel: meta.healthLabel || (isCurrent ? (this.wsConnected ? 'Current' : 'Offline') : 'Cached'),
          statusDot: healthState
        }
      })
    }
  },
  watch: {
    '$route.fullPath'() {
      this.openPopover = ''
      this.showQuickMount = false
      this.showSearchResults = false
      this.userMenuView = 'main'
    },
    currentUsername: {
      immediate: true,
      handler(username) {
        this.presence = getUserPresence(username)
      }
    },
    metricsSnap: {
      deep: true,
      handler() {
        this.persistCurrentServerMeta()
      }
    },
    bellUnreadCount(value, oldValue) {
      if (value !== oldValue) {
        this.alertAnnouncement = `${value} unread alerts`
      }
    }
  },
  mounted() {
    this.seedKnownServers()
    this.abortController = new AbortController()
    this.loadMountClientIp()
    this.loadProfileContext()
    this.syncAlerts(true)
    this.handleKeyboardShortcuts = event => this.onGlobalKeydown(event)
    document.addEventListener('keydown', this.handleKeyboardShortcuts)
    document.addEventListener('pointerdown', this.onGlobalPointerDown, true)
    this.registerPresenceActivity()
    this.presenceInterval = window.setInterval(this.checkAutoAway, 60000)
    if (window.BroadcastChannel) {
      this.authChannel = new window.BroadcastChannel(AUTH_CHANNEL)
      this.authChannel.addEventListener('message', this.onAuthChannelMessage)
    }
  },
  beforeUnmount() {
    this.abortController?.abort()
    document.removeEventListener('keydown', this.handleKeyboardShortcuts)
    document.removeEventListener('pointerdown', this.onGlobalPointerDown, true)
    window.clearInterval(this.alertPollTimer)
    window.clearInterval(this.presenceInterval)
    window.removeEventListener('mousemove', this.onActivity, true)
    window.removeEventListener('mousedown', this.onActivity, true)
    window.removeEventListener('touchstart', this.onActivity, true)
    window.removeEventListener('keydown', this.onActivity, true)
    if (this.authChannel) {
      this.authChannel.removeEventListener('message', this.onAuthChannelMessage)
      this.authChannel.close()
    }
  },
  methods: {
    formatAlertMeta,
    summarizeAlert,
    noop() {},
    toggleCollapse() {
      this.$store.commit('layout/TOGGLE_COLLAPSED')
    },
    setPopoverState(name, value) {
      this.openPopover = value ? name : (this.openPopover === name ? '' : this.openPopover)
      if (value) {
        this.showQuickMount = false
        if (name !== 'user') {
          this.userMenuView = 'main'
        }
      }
    },
    onGlobalPointerDown(event) {
      const target = event.target
      if (!this.$el?.contains(target) && !target.closest('.sc-popover-panel')) {
        this.showSearchResults = false
      }
    },
    onGlobalKeydown(event) {
      const isTypingTarget = ['INPUT', 'TEXTAREA', 'SELECT'].includes(event.target?.tagName) || event.target?.isContentEditable
      if (!isTypingTarget && (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        window.dispatchEvent(new CustomEvent('sentinel:command-palette-open'))
        return
      }
      if (!isTypingTarget && event.ctrlKey && event.key === '/') {
        event.preventDefault()
        this.focusSearch()
        return
      }
      if (!isTypingTarget && (event.key === ' ' || (event.ctrlKey && event.key.toLowerCase() === 'l')) && this.lockPinSet) {
        event.preventDefault()
        this.lockScreen()
        return
      }
      if (event.key === 'Escape') {
        this.openPopover = ''
        this.showQuickMount = false
        this.showSearchResults = false
        if (this.showQuickMountHelp) {
          this.showQuickMountHelp = false
        }
      }
    },
    focusSearch() {
      this.showSearchResults = true
      this.$nextTick(() => this.$el?.querySelector('.search-input')?.focus())
    },
    navigateSearch(direction) {
      if (!this.searchResults.length) return
      const next = this.searchActiveIndex + direction
      if (next < 0) {
        this.searchActiveIndex = this.searchResults.length - 1
      } else if (next >= this.searchResults.length) {
        this.searchActiveIndex = 0
      } else {
        this.searchActiveIndex = next
      }
    },
    selectSearchResult() {
      const item = this.searchResults[this.searchActiveIndex]
      if (item) {
        this.navigateToSearchResult(item)
      }
    },
    navigateToSearchResult(result) {
      this.$router.push(result.route)
      this.searchQuery = ''
      this.showSearchResults = false
      this.searchActiveIndex = -1
    },
    toggleLiveState() {
      this.livePaused = !this.livePaused
      if (this.livePaused) {
        this.metricsStore.stopLive()
      } else {
        this.metricsStore.startLive()
      }
    },
    applyTheme(value) {
      this.$store.commit('layout/SET_THEME', value)
    },
    applyDensity(value) {
      this.$store.commit('layout/SET_SIDEBAR_DENSITY', value)
    },
    applyPresence(value) {
      this.presence.status = value
      this.presenceAutoChanged = false
      this.savePresenceState()
    },
    updateAutoAway(value) {
      this.presence.autoAwayMinutes = value
      this.savePresenceState()
    },
    savePresenceState() {
      this.presence = setUserPresence(this.currentUsername, this.presence)
    },
    registerPresenceActivity() {
      this.onActivity = () => {
        this.lastActivityAt = Date.now()
        if (this.presenceAutoChanged && this.presence.status === 'away') {
          this.presence.status = 'online'
          this.presenceAutoChanged = false
          this.savePresenceState()
        }
      }
      ;['mousemove', 'mousedown', 'touchstart', 'keydown'].forEach(type => {
        window.addEventListener(type, this.onActivity, true)
      })
    },
    checkAutoAway() {
      if (this.presence.status !== 'online' || this.presenceAutoChanged) return
      const minutes = Number(this.presence.autoAwayMinutes || 15)
      if (!minutes || minutes < 1) return
      if (Date.now() - this.lastActivityAt >= minutes * 60 * 1000) {
        this.presence.status = 'away'
        this.presenceAutoChanged = true
        this.savePresenceState()
      }
    },
    async loadProfileContext() {
      try {
        const [{ data: me }, { data: logs }] = await Promise.all([
          api.getMe(),
          api.getAuditLogs({ limit: 200 })
        ])
        this.me = me || this.me
        const lastLogin = (Array.isArray(logs) ? logs : (logs.logs || []))
          .filter(log => log.username === (me.username || this.currentUsername) && log.success)
          .sort((left, right) => (right.ts || 0) - (left.ts || 0))[0]
        this.lastLoginEntry = lastLogin || null
      } catch (error) {
        if (error.response?.status !== 401) {
          console.error('Failed to load profile context:', error)
        }
      }
    },
    handleUserMenuOpened() {
      this.userMenuView = 'main'
      this.persistCurrentServerMeta(true)
      if (!this.me.username) {
        this.loadProfileContext()
      }
    },
    goToAccount() {
      this.openPopover = ''
      this.$router.push('/settings/general')
    },
    goToSettings(section) {
      this.openPopover = ''
      this.$router.push(`/settings/${section}`)
    },
    openTokensInfo() {
      this.$swal?.fire({
        title: 'API tokens',
        text: 'Per-user API token management is not exposed by the current agent yet.',
        icon: 'info',
        confirmButtonText: 'OK'
      })
    },
    showShortcutHelp() {
      this.$swal?.fire({
        title: 'Keyboard shortcuts',
        html: `
          <div style="text-align:left;font-size:0.92rem;line-height:1.7">
            <div><strong>⌘K / Ctrl+K</strong> open command palette</div>
            <div><strong>⌘,</strong> open settings</div>
            <div><strong>Space</strong> lock screen</div>
            <div><strong>Ctrl+/</strong> focus topbar search</div>
          </div>
        `,
        icon: 'info',
        confirmButtonText: 'Close'
      })
    },
    openWhatsNew() {
      this.whatsNewSeen = true
      localStorage.setItem(WHATS_NEW_KEY, 'true')
      this.openPopover = ''
      this.$router.push('/updates')
    },
    openDocs() {
      window.open('https://github.com/ehsanR91/sentinel#readme', '_blank', 'noopener')
    },
    openAuditForCurrentUser() {
      const query = {
        user: this.currentUsername,
        result: 'success'
      }
      if (this.lastLoginEntry?.ip) {
        query.search = `ip:${this.lastLoginEntry.ip}`
      }
      this.openPopover = ''
      this.$router.push({ path: '/audit-logs', query })
    },
    openAccountSwitcher() {
      this.$swal?.fire({ title: 'Switch account', text: 'Linked account switching is not configured yet.', icon: 'info' })
    },
    severityTone(severity) {
      if (severity === 'emergency' || severity === 'critical') return 'critical'
      if (severity === 'warning') return 'warn'
      return 'info'
    },
    severityIcon(severity) {
      if (severity === 'emergency') return 'mdi mdi-alert-octagon-outline'
      if (severity === 'critical') return 'mdi mdi-alert-circle-outline'
      if (severity === 'warning') return 'mdi mdi-alert-outline'
      return 'mdi mdi-information-outline'
    },
    alertSubtitle(group) {
      const meta = formatAlertMeta(group.base)
      const parts = [group.base.source || 'system', this.timeAgo(group.latestTs)]
      parts.push(...meta)
      return parts.filter(Boolean).join(' · ')
    },
    timeAgo(ts) {
      if (!ts) return 'just now'
      const diff = Math.max(0, Math.floor(Date.now() / 1000) - Number(ts))
      if (diff < 60) return `${diff}s ago`
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    },
    handleAlertsOpened() {
      this.$nextTick(() => {
        this.alertViewportHeight = this.$refs.alertList?.clientHeight || 360
      })
      this.syncAlerts(true)
    },
    isRuleSuppressed(alert) {
      const rule = alertRuleKey(alert)
      const muted = safeParse(localStorage.getItem(ALERT_MUTES_KEY), [])
      const snoozed = safeParse(localStorage.getItem(ALERT_SNOOZE_KEY), {})
      return muted.includes(rule) || Number(snoozed[rule] || 0) > Date.now()
    },
    async syncAlerts(forceOpenLoad = false) {
      if (!this.$store.getters['auth/loggedIn']) return
      this.loadingAlerts = forceOpenLoad
      this.alertError = ''
      try {
        const { data } = await api.getAlerts()
        const nextAlerts = Array.isArray(data) ? data.slice(0, 250) : []
        const currentIds = new Set(this.alerts.map(alert => alert.id))
        const newAlerts = nextAlerts.filter(alert => !currentIds.has(alert.id))
        const merged = mergeAlerts(nextAlerts, this.alerts)
        if (newAlerts.length && this.openPopover === 'alerts' && this.alertScrollTop > 24) {
          this.queuedAlerts = mergeAlerts(newAlerts, this.queuedAlerts)
        } else {
          this.alerts = merged
        }
        if (newAlerts.some(alert => ['emergency', 'critical'].includes(alert.severity))) {
          this.triggerBellPulse()
          if (this.openPopover !== 'alerts') {
            if (this.presence.status === 'dnd') {
              this.$swal?.fire({ toast: true, position: 'top-end', timer: 2500, showConfirmButton: false, icon: 'info', title: 'DND active — sound suppressed' })
            } else {
              this.$swal?.fire({ toast: true, position: 'top-end', timer: 3000, showConfirmButton: false, icon: 'warning', title: `${newAlerts.length} new critical alert${newAlerts.length > 1 ? 's' : ''}` })
            }
          }
        }
      } catch (error) {
        if (error.response?.status !== 401) {
          this.alertError = error.response?.data?.detail || error.message || 'Unable to load alerts.'
        }
      } finally {
        this.loadingAlerts = false
      }
    },
    triggerBellPulse() {
      this.bellPulseActive = false
      window.clearTimeout(this.bellPulseTimer)
      this.$nextTick(() => {
        this.bellPulseActive = true
        this.bellPulseTimer = window.setTimeout(() => {
          this.bellPulseActive = false
        }, 1400)
      })
    },
    onAlertListScroll() {
      this.alertScrollTop = this.$refs.alertList?.scrollTop || 0
      this.alertViewportHeight = this.$refs.alertList?.clientHeight || 360
    },
    applyQueuedAlerts() {
      this.alerts = mergeAlerts(this.queuedAlerts, this.alerts)
      this.queuedAlerts = []
      this.$nextTick(() => {
        if (this.$refs.alertList) {
          this.$refs.alertList.scrollTop = 0
          this.alertScrollTop = 0
        }
      })
    },
    toggleAlertGroup(groupId) {
      this.expandedAlertGroups = {
        ...this.expandedAlertGroups,
        [groupId]: !this.expandedAlertGroups[groupId]
      }
    },
    async markAllAsRead() {
      const ids = this.visibleAlerts.filter(alert => !alert.read).map(alert => alert.id)
      if (!ids.length) return
      await api.markAlertsAsRead(ids)
      this.alerts = this.alerts.map(alert => (ids.includes(alert.id) ? { ...alert, read: true } : alert))
    },
    async markGroupRead(group) {
      const ids = group.items.filter(item => !item.read).map(item => item.id)
      if (!ids.length) return
      await api.markAlertsAsRead(ids)
      this.alerts = this.alerts.map(alert => (ids.includes(alert.id) ? { ...alert, read: true } : alert))
    },
    async dismissGroup(group) {
      await this.markGroupRead(group)
    },
    snoozeGroup(group, minutes) {
      const key = alertRuleKey(group.base)
      const store = safeParse(localStorage.getItem(ALERT_SNOOZE_KEY), {})
      store[key] = Date.now() + (minutes * 60 * 1000)
      localStorage.setItem(ALERT_SNOOZE_KEY, JSON.stringify(store))
      this.alerts = this.alerts.slice()
    },
    muteRule(group) {
      const key = alertRuleKey(group.base)
      const store = safeParse(localStorage.getItem(ALERT_MUTES_KEY), [])
      if (!store.includes(key)) {
        localStorage.setItem(ALERT_MUTES_KEY, JSON.stringify([...store, key]))
      }
      this.alerts = this.alerts.slice()
    },
    async blockSourceIp(group) {
      if (!group.base.ip) return
      const result = await this.$swal?.fire({
        title: `Block ${group.base.ip}?`,
        text: 'A firewall block will be created for this IP.',
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: 'Block IP'
      })
      if (result && !result.isConfirmed) return
      await api.banIp(group.base.ip)
      this.$swal?.fire({ toast: true, position: 'top-end', timer: 2500, showConfirmButton: false, icon: 'success', title: `${group.base.ip} blocked` })
    },
    copyAlertJson(group) {
      navigator.clipboard.writeText(JSON.stringify(group.items, null, 2)).catch(() => {})
    },
    async openAlertGroup(group) {
      await this.markGroupRead(group)
      this.openPopover = ''
      this.$router.push('/alerts')
    },
    openAlertsPage() {
      this.openPopover = ''
      this.$router.push('/alerts')
    },
    async confirmSignOut() {
      const hasUnsavedChanges = !!window.__sc_unsaved_changes__
      const recentLogin = this.loginAt ? (Date.now() - this.loginAt) < (5 * 60 * 1000) : false
      if (!hasUnsavedChanges && !recentLogin) {
        await this.performLogout()
        return
      }

      const result = await this.$swal?.fire({
        title: 'Sign out?',
        text: hasUnsavedChanges
          ? 'You have unsaved changes in another view.'
          : 'You signed in recently. Confirm to avoid accidental sign-out.',
        icon: 'warning',
        showCancelButton: true,
        showDenyButton: hasUnsavedChanges && typeof window.__sc_save_dirty_view__ === 'function',
        confirmButtonText: hasUnsavedChanges ? 'Discard & sign out' : 'Sign out',
        denyButtonText: 'Save & sign out',
        cancelButtonText: 'Cancel'
      })
      if (!result) return
      if (result.isDenied && typeof window.__sc_save_dirty_view__ === 'function') {
        await window.__sc_save_dirty_view__()
        await this.performLogout()
        return
      }
      if (result.isConfirmed) {
        await this.performLogout()
      }
    },
    async performLogout() {
      try {
        await api.logout()
      } catch {
        // Ignore transport failures during logout.
      }
      this.authChannel?.postMessage({ type: 'signout' })
      this.$store.dispatch('auth/logout')
      this.openPopover = ''
      this.$router.push('/login')
    },
    onAuthChannelMessage(event) {
      if (event.data?.type === 'signout') {
        this.openPopover = ''
        this.$store.dispatch('auth/logout')
        this.$router.push('/login')
      }
    },
    seedKnownServers() {
      if (!this.knownServers.some(server => server.url === this.currentOrigin)) {
        this.knownServers = [{ id: 'current', name: this.currentServerLabel, url: this.currentOrigin }, ...this.knownServers]
        localStorage.setItem(SERVERS_KEY, JSON.stringify(uniqueBy(this.knownServers, entry => entry.url)))
      }
    },
    persistCurrentServerMeta(forceHealth = false) {
      const entry = {
        ...this.serverMetaCache[this.currentOrigin],
        cpu: `${Number(this.metricsSnap.cpu_pct || 0).toFixed(0)}%`,
        ram: `${Number(this.metricsSnap.ram_pct || 0).toFixed(0)}%`,
        healthLabel: this.wsConnected ? 'Current' : 'Offline',
        healthState: this.wsConnected ? 'online' : 'offline'
      }
      this.serverMetaCache = {
        ...this.serverMetaCache,
        [this.currentOrigin]: entry
      }
      localStorage.setItem(SERVER_META_KEY, JSON.stringify(this.serverMetaCache))
      if (forceHealth) {
        api.getHealth().then(({ data }) => {
          const overall = data?.overall_status || 'warning'
          const statusMap = { healthy: 'online', warning: 'away', critical: 'dnd' }
          this.serverMetaCache = {
            ...this.serverMetaCache,
            [this.currentOrigin]: {
              ...entry,
              healthLabel: overall,
              healthState: statusMap[overall] || 'offline'
            }
          }
          localStorage.setItem(SERVER_META_KEY, JSON.stringify(this.serverMetaCache))
        }).catch(() => {})
      }
    },
    promptAddServer() {
      const name = window.prompt('Server label')
      if (!name) return
      const url = window.prompt('Server URL (for example https://sentinel.example.com)')
      if (!url) return
      try {
        const normalized = new URL(url).origin
        if (this.knownServers.some(server => server.url === normalized)) return
        this.knownServers = [...this.knownServers, { id: `${Date.now()}`, name, url: normalized }]
        localStorage.setItem(SERVERS_KEY, JSON.stringify(this.knownServers))
      } catch {
        window.alert('Enter a valid URL for the target SentinelCore instance.')
      }
    },
    switchServer(server) {
      if (!server?.url) return
      if (server.url === this.currentOrigin) {
        this.userMenuView = 'main'
        return
      }
      try {
        window.location.assign(`${server.url}${this.$route.fullPath}`)
      } catch {
        this.$swal?.fire({ toast: true, position: 'top-end', timer: 2500, showConfirmButton: false, icon: 'error', title: 'Server switch failed' })
      }
    },
    lockScreen() {
      this.openPopover = ''
      if (!this.lockPinSet) {
        this.$router.push('/settings/access-control')
        return
      }
      window.dispatchEvent(new CustomEvent('sentinel:lock'))
    },
    async onInstallClick() {
      try {
        await promptInstall()
      } catch (err) {
        console.error('PWA install prompt failed', err)
      }
    },
    openInstallHelp() {
      this.showPwaInstallModal = true
    },
    closeInstallHelp() {
      this.showPwaInstallModal = false
    },
    async refreshAppVersion() {
      if (pwaState.needsRefresh) {
        await reloadApp()
      }
    },
    async loadMountClientIp() {
      try {
        const { data } = await api.getMe()
        this.mountClientIp = data?.client_ip || ''
      } catch {
        this.mountClientIp = ''
      }
    },
    async toggleQuickMount() {
      this.showQuickMount = !this.showQuickMount
      this.openPopover = ''
      if (this.showQuickMount && !this.tunnelApps.length) {
        await this.refreshTunnelApps()
      }
    },
    openQuickMountHelp() {
      this.showQuickMountHelp = true
    },
    copyMountExample() {
      navigator.clipboard.writeText(this.mountExampleCommand).catch(() => {})
      this.mountHelpCopied = true
      window.setTimeout(() => { this.mountHelpCopied = false }, 1800)
    },
    async refreshTunnelApps() {
      this.loadingTunnels = true
      try {
        const { data } = await api.getTunnelableApps()
        if (Array.isArray(data)) {
          this.tunnelApps = data
          return
        }
        this.tunnelApps = Array.isArray(data?.apps) ? data.apps : []
        this.detectedSshHost = data?.connection?.host || this.detectedSshHost
        this.detectedSshPort = Number.parseInt(data?.connection?.ssh_port, 10) || this.detectedSshPort || 22
        this.detectedSshUser = data?.connection?.ssh_user || this.detectedSshUser
        this.detectedSshUserSource = data?.connection?.ssh_user_source || this.detectedSshUserSource
      } catch {
        this.tunnelApps = []
      } finally {
        this.loadingTunnels = false
      }
    },
    copyMount(app) {
      const portFlag = String(this.effectiveSshPort) !== '22' ? ` -p ${this.effectiveSshPort}` : ''
      const command = `ssh -L ${app.port}:localhost:${app.port}${portFlag} ${this.effectiveSshUser}@${this.effectiveSshHost}`
      navigator.clipboard.writeText(command).catch(() => {})
      const key = app.name + app.port
      this.mountCopied = key
      window.setTimeout(() => { if (this.mountCopied === key) this.mountCopied = '' }, 2000)
    },
    async grantAccess(app) {
      const ip = this.mountClientIp || 'your current IP'
      let durationHours = 3
      if (this.$swal) {
        const response = await this.$swal.fire({
          title: 'Grant temporary access?',
          html: `
            <div style="margin-top:10px;text-align:left">
              <p style="margin-bottom:6px;color:#c9d8f0;font-size:0.87rem">Allow <strong>${app.name}</strong> (port ${app.port}) to be accessed directly from <strong>${ip}</strong>.</p>
              <label style="font-size:0.8rem;color:#8fa8c8;display:block;margin-bottom:4px">Duration</label>
              <select id="sc-grant-duration" style="width:100%;padding:6px 10px;border-radius:6px;border:1px solid rgba(100,140,200,0.3);background:#0e1c30;color:#c9d8f0;font-size:0.85rem">
                <option value="1">1 hour</option>
                <option value="3" selected>3 hours</option>
                <option value="6">6 hours</option>
                <option value="12">12 hours</option>
                <option value="24">24 hours</option>
              </select>
            </div>
          `,
          icon: 'warning',
          showCancelButton: true,
          confirmButtonText: 'Grant access',
          preConfirm: () => parseInt(document.getElementById('sc-grant-duration')?.value) || 3
        })
        if (!response.isConfirmed) return
        durationHours = response.value
      } else if (!window.confirm(`Allow browser access to ${app.name} on port ${app.port} for IP ${ip}?`)) {
        return
      }

      this.grantingPort = app.port
      try {
        const { data } = await api.grantTunnelAccess(app.port, durationHours)
        const grantedIp = data?.ip || this.mountClientIp || 'your IP'
        const expiresAt = data?.expires_at ? new Date(data.expires_at * 1000).toLocaleString() : `in ${durationHours}h`
        const browseUrl = `http://${this.effectiveSshHost}:${app.port}`
        await this.$swal?.fire({
          title: 'Access granted',
          html: `<div style="text-align:left;font-size:0.87rem;color:#c9d8f0"><p><strong>IP:</strong> ${grantedIp} &nbsp; <strong>Port:</strong> ${app.port}</p><p><strong>Expires:</strong> ${expiresAt}</p><p style="margin-top:10px">Open: <a href="${browseUrl}" target="_blank" style="color:#4a9eff">${browseUrl}</a></p></div>`,
          icon: 'success',
          confirmButtonText: 'OK'
        })
      } catch (error) {
        const message = error?.response?.data?.error || error?.response?.data?.message || 'Failed to grant temporary access'
        await this.$swal?.fire({ title: 'Grant failed', text: message, icon: 'error', confirmButtonText: 'OK' })
      } finally {
        this.grantingPort = null
      }
    }
  }
}
</script>

<style scoped>
.topbar-breadcrumb {
  align-items: center;
  gap: 0.45rem;
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.topbar-breadcrumb__prefix {
  color: var(--text-tertiary);
}

.topbar-breadcrumb__current {
  color: var(--text-primary);
  font-weight: 600;
}

.topbar-system-cluster {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.topbar-live-pill {
  min-height: 36px;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0 0.7rem;
  border: 1px solid color-mix(in srgb, #22d67c 22%, var(--border-subtle));
  border-radius: 999px;
  background: color-mix(in srgb, #22d67c 10%, transparent);
  color: var(--text-primary);
}

.topbar-live-pill.is-paused,
.topbar-live-pill.is-offline {
  border-color: var(--border-subtle);
  background: color-mix(in srgb, var(--surface-2) 88%, transparent);
}

.topbar-btn {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--text-secondary);
  position: relative;
}

.topbar-btn:hover,
.topbar-btn.is-active,
.user-menu:hover,
.user-menu.is-active {
  background: var(--accent-muted);
  color: var(--text-primary);
}

.topbar-btn--quick-mount {
  background: rgba(34,214,124,0.06);
  border: 1px solid rgba(34,214,124,0.18);
  color: #22d67c;
}

.topbar-btn--bell i,
.user-menu i {
  font-size: 1.1rem;
}

.topbar-badge {
  position: absolute;
  top: 3px;
  right: 2px;
  min-width: 18px;
  height: 18px;
  padding: 0 0.3rem;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 10.5px;
  font-weight: 600;
  color: #fff;
}

.topbar-badge--critical {
  background: #f04040;
}

.topbar-badge--warn {
  background: #f5a623;
}

.topbar-badge--neutral {
  background: var(--text-tertiary);
}

.has-critical-pulse::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: 14px;
  border: 2px solid rgba(240, 64, 64, 0.35);
  animation: bellPulse 1.2s ease-out 1;
}

.user-menu {
  min-height: 40px;
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.2rem 0.45rem 0.2rem 0.2rem;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--text-primary);
}

.user-name-wrap {
  flex-direction: column;
  align-items: flex-start;
  gap: 0.05rem;
  min-width: 0;
}

.user-name {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.875rem;
  font-weight: 600;
}

.user-role {
  font-size: 0.72rem;
  color: var(--text-secondary);
  text-transform: lowercase;
}

.topbar-search {
  margin-left: 0.35rem;
}

.search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-tertiary);
  pointer-events: none;
}

.search-input {
  width: 212px;
  min-height: 38px;
  padding: 0.55rem 0.75rem 0.55rem 2.15rem;
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  background: var(--surface-2);
  color: var(--text-primary);
}

.search-input:focus {
  width: 272px;
  outline: none;
}

.search-results-dropdown {
  position: absolute;
  top: 48px;
  right: 0;
  width: min(340px, calc(100vw - 1rem));
  max-height: 400px;
  overflow-y: auto;
  background: var(--surface-1);
  border: 1px solid var(--border-subtle);
  border-radius: 14px;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.28);
  z-index: 1100;
}

.search-results-header {
  padding: 0.8rem 0.95rem;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 0.74rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-tertiary);
}

.search-result-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.7rem;
  padding: 0.8rem 0.95rem;
  border: 0;
  background: transparent;
  text-align: left;
  color: var(--text-primary);
}

.search-result-item:hover,
.search-result-item.active {
  background: color-mix(in srgb, var(--accent) 10%, transparent);
}

.search-result-path {
  margin-left: auto;
  font-size: 0.72rem;
  color: var(--text-tertiary);
}

.topbar-popover :deep(.sc-popover-body) {
  padding-top: 0;
}

.topbar-popover--user {
  width: min(280px, calc(100vw - 24px));
}

.topbar-popover__body--user {
  padding: 10px 10px 12px !important;
  max-height: calc(100vh - 80px);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.topbar-alerts {
  display: grid;
  gap: 0.65rem;
}

.topbar-alerts__meta,
.topbar-alerts__footer,
.user-popover__subheader {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.topbar-alerts__summary {
  display: grid;
  gap: 0;
}

.topbar-alerts__summary strong,
.user-popover__meta strong,
.user-popover__item strong,
.user-popover__setting-row strong,
.user-popover__server-item strong {
  color: var(--text-primary);
}

.topbar-alerts__summary span,
.topbar-link-button,
.topbar-link-button--small,
.topbar-alert-row__subtitle,
.user-popover__meta span,
.user-popover__item p,
.user-popover__setting-row p,
.user-popover__server-item p,
.user-popover__server-stats,
.user-popover__select,
.user-popover__section-label,
.user-popover__last-login {
  color: var(--text-secondary);
  font-size: 0.78rem;
}

.topbar-link-button,
.topbar-link-button--small,
.user-popover__last-login {
  border: 0;
  background: transparent;
  padding: 0;
  text-align: left;
}

.topbar-tabs {
  display: flex;
  gap: 0.35rem;
  overflow-x: auto;
  padding-bottom: 0.1rem;
}

.topbar-tab {
  min-height: 32px;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0 0.65rem;
  border: 1px solid var(--border-subtle);
  border-radius: 999px;
  background: var(--surface-2);
  color: var(--text-secondary);
  font-size: 12px;
}

.topbar-tab.active {
  border-color: color-mix(in srgb, var(--accent) 36%, var(--border-subtle));
  background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
  color: var(--text-primary);
}

.topbar-new-pill {
  min-height: 30px;
  justify-self: start;
  padding: 0 0.65rem;
  border: 1px solid color-mix(in srgb, var(--accent) 36%, var(--border-subtle));
  border-radius: 999px;
  background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
  color: var(--text-primary);
  font-size: 12px;
}

.topbar-alerts__list {
  max-height: 440px;
  overflow-y: auto;
  padding-right: 0.1rem;
}

.topbar-alerts__items {
  display: grid;
  gap: 0.45rem;
}

.topbar-alert-row {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  gap: 0.65rem;
  min-height: 56px;
  padding: 0.55rem 0.7rem;
  border: 1px solid var(--border-subtle);
  border-left: 3px solid transparent;
  border-radius: 12px;
  background: var(--surface-2);
}

.topbar-alert-row:hover,
.user-popover__item:hover,
.user-popover__server-item:hover {
  background: var(--surface-hover, color-mix(in srgb, var(--surface-2) 88%, transparent));
}

.topbar-alert-row.is-unread {
  border-left-color: var(--accent);
}

.topbar-alert-row__icon {
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: color-mix(in srgb, var(--surface-1) 86%, transparent);
}

.topbar-alert-row__icon.severity-critical {
  color: #f04040;
}

.topbar-alert-row__icon.severity-warn {
  color: #f5a623;
}

.topbar-alert-row__icon.severity-info {
  color: var(--accent);
}

.topbar-alert-row__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.55rem;
}

.topbar-alert-row__summary {
  min-width: 0;
}

.topbar-alert-row__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1.25;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.topbar-tag,
.topbar-count-pill {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 0.45rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--surface-1) 86%, transparent);
  font-size: 0.7rem;
  color: var(--text-secondary);
}

.topbar-alert-row__subtitle {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-popover__server-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-top: 0.45rem;
}

.topbar-unread-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--accent);
}

.topbar-row-menu {
  position: relative;
}

.topbar-row-menu summary {
  list-style: none;
}

.topbar-row-menu summary::-webkit-details-marker {
  display: none;
}

.topbar-row-menu__trigger {
  width: 30px;
  height: 30px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
}

.topbar-row-menu__body {
  position: absolute;
  top: calc(100% + 0.35rem);
  right: 0;
  z-index: 4;
  min-width: 170px;
  padding: 0.35rem;
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  background: var(--surface-1);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.22);
}

.topbar-alert-expansion {
  display: grid;
  gap: 0.45rem;
  margin-top: 0.55rem;
}

.topbar-alert-expansion__item {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.45rem;
  border-top: 1px dashed var(--border-subtle);
  font-size: 0.76rem;
  color: var(--text-secondary);
}

.topbar-state {
  min-height: 180px;
  display: grid;
  place-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
}

.user-popover {
  display: grid;
  gap: 8px;
}

.user-popover__scroll-area {
  max-height: 50vh;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 2px;
}

.user-popover__header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0 2px 8px;
  border-bottom: 1px solid var(--border-subtle);
  background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 18%, transparent), color-mix(in srgb, var(--surface-2) 80%, transparent));
}

.user-popover__hero {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  min-width: 0;
}

.user-popover__meta {
  display: grid;
  gap: 1px;
  min-width: 0;
  flex: 1;
}

.user-popover__meta strong {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-popover__meta-line {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.user-popover__section {
  display: grid;
  gap: 4px;
}

.user-popover__section + .user-popover__section {
  margin-top: 2px;
  padding-top: 8px;
  border-top: 1px solid var(--border-subtle);
}

.user-popover__section-label {
  margin: 0 2px 2px;
  font-size: 10px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}

.user-popover__item,
.user-popover__setting-row,
.user-popover__server-item,
.user-popover__signout,
.user-popover__add-account {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 36px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  text-align: left;
}

.user-popover__item:focus-visible,
.user-popover__server-item:focus-visible,
.user-popover__signout:focus-visible,
.user-popover__add-account:focus-visible {
  background: var(--accent-muted);
}

.user-popover__setting-row {
  min-height: 0;
  padding: 0;
  align-items: center;
  gap: 8px;
}

.user-popover__setting-row--stacked {
  padding-top: 4px;
}

.user-popover__item-main {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 10px;
}

.user-popover__item-main i,
.user-popover__item-meta-icon {
  font-size: 16px;
  color: var(--text-secondary);
}

.user-popover__item-label,
.user-popover__setting-label {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 400;
}

.user-popover__item-meta {
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.user-popover__signout {
  position: sticky;
  bottom: 0;
  z-index: 2;
  margin-top: 4px;
  min-height: 36px;
  color: #f04040;
  background: var(--surface-1);
  border-top: 1px solid var(--border-subtle);
  border-radius: 0 0 12px 12px;
  padding: 8px 12px;
  justify-content: flex-start;
}

.user-popover__signout:hover {
  background: color-mix(in srgb, #f04040 9%, var(--surface-1));
}

.user-popover__server-search {
  position: relative;
}

.user-popover__server-search i {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
}

.user-popover__server-search input,
.user-popover__select {
  width: 100%;
  min-height: 36px;
  padding: 0.45rem 0.8rem 0.45rem 2rem;
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  background: var(--surface-1);
  color: var(--text-primary);
}

.user-popover__select {
  padding-left: 0.8rem;
}

.user-popover__server-list {
  display: grid;
  gap: 0.65rem;
}

.user-popover__server-head,
.user-popover__server-health {
  display: flex;
  align-items: center;
  gap: 0.45rem;
}

.user-popover__server-item.active {
  background: var(--accent-muted);
}

.user-popover__last-login {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-popover__setting-row :deep(.segmented-control) {
  padding: 2px;
  border-radius: 999px;
}

.user-popover__setting-row :deep(.segmented-control__item) {
  min-width: 0;
  min-height: 24px;
  padding: 0 8px;
  font-size: 11px;
  line-height: 24px;
}

.quick-mount-panel {
  border: 1px solid var(--sc-border, #1e2d4a);
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0,0,0,0.35);
}

.quick-mount-row {
  cursor: default;
  transition: background 0.12s;
}

.quick-mount-row:hover {
  background: rgba(74, 158, 255, 0.04);
}

.quick-mount-help-backdrop,
.pwa-install-backdrop {
  position: fixed;
  inset: 0;
  z-index: 1400;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(5, 10, 18, 0.74);
}

.quick-mount-help-modal,
.pwa-install-modal {
  width: min(560px, 100%);
  background: var(--surface-1);
  border: 1px solid var(--border-subtle);
  border-radius: 18px;
  box-shadow: 0 28px 72px rgba(0, 0, 0, 0.35);
  overflow: hidden;
}

.quick-mount-help-header,
.pwa-install-header,
.pwa-install-footer {
  padding: 1rem 1.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.quick-mount-help-body,
.pwa-install-body {
  padding: 0 1.25rem 1.25rem;
}

.quick-mount-help-note {
  margin-top: 10px;
  display: flex;
  gap: 8px;
  align-items: flex-start;
  font-size: 0.75rem;
  color: #8aa4c8;
  background: rgba(34, 214, 124, 0.08);
  border: 1px solid rgba(34, 214, 124, 0.2);
  border-radius: 8px;
  padding: 8px 10px;
}

.quick-mount-help-example {
  margin-top: 10px;
  border: 1px solid rgba(74, 158, 255, 0.18);
  border-radius: 8px;
  background: rgba(74, 158, 255, 0.05);
  padding: 8px 10px;
}

.quick-mount-help-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  padding: 0 14px 12px;
}

@keyframes bellPulse {
  0% { opacity: 0.85; transform: scale(0.9); }
  100% { opacity: 0; transform: scale(1.12); }
}

@media (max-width: 1024px) {
  .user-name-wrap,
  .user-menu .mdi-chevron-down {
    display: none !important;
  }
}

@media (max-width: 768px) {
  .topbar-search {
    display: none !important;
  }

  .topbar-right {
    gap: 0.3rem;
  }
}

@media (max-width: 640px) {
  .topbar-tabs {
    padding-bottom: 0.35rem;
  }

  .topbar-alerts__list {
    max-height: min(64vh, 520px);
  }

  .topbar-alert-row,
  .user-popover__item,
  .user-popover__setting-row,
  .user-popover__server-item,
  .user-popover__signout,
  .user-popover__add-account {
    padding-inline: 10px;
  }
}

@media (pointer: coarse) {
  .topbar-btn,
  .topbar-tab,
  .user-popover__item,
  .user-popover__server-item,
  .user-popover__signout,
  .user-popover__add-account {
    min-height: 44px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .has-critical-pulse::after,
  .search-input,
  .quick-mount-row {
    animation: none !important;
    transition: none !important;
  }
}
</style>
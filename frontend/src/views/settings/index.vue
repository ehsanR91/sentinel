<template>
  <div class="settings-page">
    <PageHeader :title="currentSection.label" icon="mdi mdi-cog-outline" :items="pageHeaderItems">
      <template #actions>
        <AppButton
          variant="secondary"
          size="sm"
          icon="mdi mdi-file-document-edit-outline"
          label="Review Changes"
          :disabled="!dirtyFields.length"
          @click="reviewDrawerOpen = true"
        />
      </template>
    </PageHeader>

    <div class="settings-layout">
      <aside class="settings-rail sc-surface" aria-label="Settings navigation">
        <div class="settings-rail__header">
          <div class="settings-rail__title">Settings</div>
          <p class="settings-rail__summary">Searchable, sectioned configuration with one save model.</p>
        </div>
        <nav class="settings-nav">
          <router-link
            v-for="section in settingsSections"
            :key="section.id"
            class="settings-nav__item sc-focus-ring"
            :class="{ active: currentSectionId === section.id }"
            :to="sectionRoute(section.id)"
          >
            <i :class="section.icon" aria-hidden="true"></i>
            <span>{{ section.label }}</span>
          </router-link>
        </nav>
      </aside>

      <div class="settings-main">
        <div class="settings-mobile-nav sc-surface">
          <label for="settings-mobile-select" class="visually-hidden">Select settings section</label>
          <select id="settings-mobile-select" class="settings-mobile-nav__select sc-focus-ring" :value="currentSectionId" @change="goToSection($event.target.value)">
            <option v-for="section in settingsSections" :key="section.id" :value="section.id">{{ section.label }}</option>
          </select>
        </div>

        <header class="settings-toolbar sc-surface">
          <div>
            <div class="settings-breadcrumbs">
              <router-link to="/settings/general">Settings</router-link>
              <span>/</span>
              <span>{{ currentSection.label }}</span>
              <template v-if="activeSubsectionLabel">
                <span>/</span>
                <span>{{ activeSubsectionLabel }}</span>
              </template>
            </div>
            <p class="settings-toolbar__description">{{ currentSection.description }}</p>
          </div>
          <div class="settings-search">
            <label for="settings-search" class="visually-hidden">Search settings</label>
            <div class="settings-search__field">
              <i class="mdi mdi-magnify" aria-hidden="true"></i>
              <input
                id="settings-search"
                ref="searchInput"
                v-model.trim="searchQuery"
                class="sc-focus-ring"
                type="search"
                placeholder="Search settings (Ctrl/Cmd+K)"
                @focus="searchResultsOpen = true"
                @keydown.down.prevent="moveSearchSelection(1)"
                @keydown.up.prevent="moveSearchSelection(-1)"
                @keydown.enter.prevent="activateSelectedSearchResult"
                @keydown.esc.prevent="closeSearchResults"
              >
            </div>
            <div v-if="searchResultsOpen && searchQuery" class="settings-search__results">
              <button
                v-for="(result, index) in filteredSearchResults"
                :key="result.id"
                type="button"
                class="settings-search__result"
                :class="{ active: searchSelectionIndex === index }"
                @mouseenter="searchSelectionIndex = index"
                @click="jumpToSearchResult(result)"
              >
                <div>
                  <strong>{{ result.label }}</strong>
                  <p>{{ result.description }}</p>
                </div>
                <span>{{ result.breadcrumb }}</span>
              </button>
              <div v-if="!filteredSearchResults.length" class="settings-search__empty">
                No settings match "{{ searchQuery }}".
              </div>
            </div>
          </div>
        </header>

        <div ref="contentRoot" class="settings-content">
          <template v-if="currentSectionId === 'general'">
            <div id="general-runtime" class="settings-anchor" data-subsection="Runtime">
              <SettingsSection title="Runtime" description="Detected server identity and agent-owned runtime values." heading-id="settings-general-runtime" eyebrow="General" sticky>
                <SettingRow
                  row-id="general-admin-email"
                  label="Admin email"
                  label-for="settings-admin-email"
                  description="Primary operator email used for notifications and escalation paths."
                  help-text="Stored on the agent and reused by the notification pipeline."
                  status-label="Required"
                  status-state="warn"
                  :error="errorFor('alert_email')"
                  :highlighted="isHighlighted('general-admin-email')"
                  footnote="Changes are persisted on the agent immediately after save."
                >
                  <TextField input-id="settings-admin-email" v-model="form.alert_email" type="email" placeholder="admin@yourdomain.com" @blur="touchField('alert_email')" />
                </SettingRow>

                <SettingRow
                  row-id="general-runtime-hostname"
                  label="Detected hostname"
                  description="Reported by the live metrics stream. This is currently read-only in SentinelCore."
                  status-label="Runtime"
                  status-state="info"
                  footnote="A writable hostname endpoint is not exposed by the current agent API."
                >
                  <TextField input-id="settings-hostname" :model-value="runtimeHostname" readonly />
                </SettingRow>

                <SettingRow
                  row-id="general-runtime-listen"
                  label="Listen address"
                  description="Current runtime binding for the web process."
                  status-label="Agent config"
                  status-state="muted"
                  footnote="Changing bind addresses requires backend support and a controlled restart." 
                >
                  <TextField input-id="settings-listen-addr" :model-value="runtimeListenAddress" readonly />
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="general-preferences" class="settings-anchor" data-subsection="Preferences">
              <SettingsSection title="Preferences" description="Local operator defaults stored in this browser until synced settings arrive." heading-id="settings-general-preferences" eyebrow="Interface">
                <SettingRow
                  row-id="general-theme"
                  label="Default theme"
                  description="Preferred initial theme when the UI loads in this browser."
                  status-label="Local only"
                  status-state="muted"
                  :highlighted="isHighlighted('general-theme')"
                  footnote="Stored locally because the current agent does not expose account-level theme defaults."
                >
                  <SegmentedControl v-model="form.ui_theme_default" label="Theme default" :options="themeOptions" />
                </SettingRow>

                <SettingRow
                  row-id="general-language"
                  label="Language"
                  label-for="settings-language"
                  description="Reserved for upcoming localized copy."
                  status-label="Preview"
                  status-state="info"
                  :highlighted="isHighlighted('general-language')"
                  footnote="Stored locally until language preferences are part of the signed-in profile."
                >
                  <SelectField input-id="settings-language" v-model="form.ui_language" :options="languageOptions" />
                </SettingRow>

                <SettingRow
                  row-id="general-timezone"
                  label="Timezone"
                  label-for="settings-timezone"
                  description="Override local time rendering for settings and audit references."
                  status-label="Local only"
                  status-state="muted"
                  :highlighted="isHighlighted('general-timezone')"
                >
                  <TextField input-id="settings-timezone" v-model="form.ui_timezone" placeholder="UTC" @blur="touchField('ui_timezone')" />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'security'">
            <div id="security-login-protection" class="settings-anchor" data-subsection="Login Protection">
              <SettingsSection title="Login Protection" description="Throttle brute-force activity and prepare the allowlist model." heading-id="settings-security-login" eyebrow="Security" sticky>
                <SettingRow
                  row-id="security-login-attempts"
                  label="Max login attempts"
                  label-for="settings-brute-force"
                  description="After this many failed attempts in 10 minutes, the IP is auto-banned."
                  help-text="Set between 1 and 100. Higher values relax the temporary ban threshold."
                  status-label="Active"
                  status-state="ok"
                  :error="errorFor('brute_force_threshold')"
                  :highlighted="isHighlighted('security-login-attempts')"
                >
                  <NumberField input-id="settings-brute-force" v-model="form.brute_force_threshold" placeholder="5" @blur="touchField('brute_force_threshold')" />
                </SettingRow>

                <SettingRow
                  row-id="security-ip-allowlist"
                  label="IP allowlist"
                  description="Model CIDR entries ahead of agent-side enforcement."
                  status-label="Awaiting agent support"
                  status-state="warn"
                  :highlighted="isHighlighted('security-ip-allowlist')"
                  footnote="The current backend does not expose an enforceable allowlist key yet, so this remains a design-preview row."
                  disabled
                >
                  <TextAreaField
                    input-id="settings-allowlist-preview"
                    :model-value="allowlistPreview"
                    readonly
                    :rows="4"
                  />
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="security-tls" class="settings-anchor" data-subsection="TLS">
              <SettingsSection title="TLS / HTTPS" description="Current transport posture and the missing fields required for agent-side certificate management." heading-id="settings-security-tls" eyebrow="Transport">
                <SettingRow
                  row-id="security-tls"
                  label="Transport security"
                  description="TLS configuration is not currently writable through the settings API."
                  status-label="Runtime only"
                  status-state="muted"
                  :highlighted="isHighlighted('security-tls')"
                  footnote="Expose certificate path, key path, and renewal controls in the agent before turning this into a writable toggle."
                >
                  <ToggleSwitch input-id="settings-tls" :model-value="false" label="Agent-side TLS controls not yet exposed" disabled />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'access-control'">
            <div id="access-two-factor" class="settings-anchor" data-subsection="Two-Factor Authentication">
              <SettingsSection title="Two-Factor Authentication" description="TOTP enrollment, recovery posture, and safe disable flows." heading-id="settings-access-2fa" eyebrow="Access Control" sticky>
                <SettingRow
                  row-id="access-2fa"
                  label="2FA status"
                  description="Use an authenticator app to protect privileged access."
                  :status-label="me.totp_enabled ? 'Enabled' : 'Disabled'"
                  :status-state="me.totp_enabled ? 'ok' : 'warn'"
                  :highlighted="isHighlighted('access-2fa')"
                  footnote="Backup codes and recovery-email workflows require additional backend support."
                >
                  <div class="settings-inline-actions">
                    <AppButton
                      v-if="!me.totp_enabled && !totpSetup.secret"
                      variant="primary"
                      size="sm"
                      icon="mdi mdi-qrcode"
                      label="Set Up 2FA"
                      :loading="totpSetup.loading"
                      @click="initSetup2FA"
                    />
                    <div v-else-if="totpSetup.secret" class="settings-setup-block">
                      <canvas ref="qrCanvas" class="settings-qr"></canvas>
                      <TextField input-id="settings-totp-secret" :model-value="totpSetup.secret" readonly />
                      <div class="settings-inline-actions">
                        <TextField input-id="settings-totp-code" v-model="totpSetup.verifyCode" placeholder="Enter 6 digit code" />
                        <AppButton variant="primary" size="sm" label="Activate" @click="enable2FA" />
                      </div>
                    </div>
                    <AppButton v-else variant="ghost" size="sm" icon="mdi mdi-check-decagram-outline" label="2FA is active" disabled />
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="access-secret-gate" class="settings-anchor" data-subsection="Secret Link Gate">
              <SettingsSection title="Secret Link Gate" description="Hidden login entrypoint and unlock duration." heading-id="settings-access-secret" eyebrow="Entry Gate">
                <SettingRow
                  row-id="access-secret-gate"
                  label="Secret URL"
                  label-for="settings-secret-path"
                  description="Visitors must enter through this path before the login screen becomes visible."
                  help-text="Changing the path invalidates the old URL immediately after save."
                  :error="errorFor('secret_path')"
                  :highlighted="isHighlighted('access-secret-gate')"
                  footnote="Stored on the agent and enforced by the gate middleware."
                >
                  <SecretField input-id="settings-secret-path" v-model="form.secret_path" placeholder="sentinel-core" @blur="touchField('secret_path')">
                    <template #actions>
                      <AppButton variant="ghost" size="sm" icon="mdi mdi-auto-fix" aria-label="Generate a new secret path" @click="generateSecretPath" />
                    </template>
                  </SecretField>
                  <div class="settings-inline-note">{{ secretUrlPreview }}</div>
                </SettingRow>

                <SettingRow
                  row-id="access-secret-gate-expiry"
                  label="Gate unlock duration"
                  description="How long the gate stays unlocked once a visitor passes through."
                  footnote="Persisted as days on the current agent. Minutes and hours are rounded up when saved."
                >
                  <DurationPicker input-id="settings-gate-expiry" v-model="form.gate_expiry" />
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="access-lock-screen" class="settings-anchor" data-subsection="Lock Screen">
              <SettingsSection title="Lock Screen" description="Segmented PIN workflow and local access controls." heading-id="settings-access-lock" eyebrow="Session Lock">
                <SettingRow
                  row-id="access-lock-screen"
                  label="Lock screen"
                  description="Require a six digit PIN before the workstation unlocks."
                  :status-label="lockEnabled ? (lockPinSet ? 'Enabled' : 'Pending PIN') : 'Disabled'"
                  :status-state="lockEnabled ? 'ok' : 'muted'"
                  :highlighted="isHighlighted('access-lock-screen')"
                >
                  <div class="settings-stack-md">
                    <ToggleSwitch input-id="settings-lock-enabled" :model-value="lockEnabled" label="Enable lock screen" @update:modelValue="setLockEnabled" />
                    <div v-if="lockEnabled" class="settings-stack-sm">
                      <PinField v-model="lockForm.pin" label="New lock PIN" />
                      <PinField v-model="lockForm.confirmPin" label="Confirm lock PIN" />
                      <div class="settings-inline-actions">
                        <AppButton variant="primary" size="sm" :loading="settingPin" :label="lockPinSet ? 'Change PIN' : 'Set PIN'" @click="saveLockPin" />
                        <ToggleSwitch input-id="settings-lock-on-blur" v-model="lockForm.lockOnBlur" label="Lock on tab switch" />
                      </div>
                      <div class="settings-inline-note">Shortcut: press Space anywhere outside a form control to lock the screen.</div>
                    </div>
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'notifications'">
            <div id="notifications-email" class="settings-anchor" data-subsection="Email Delivery">
              <SettingsSection title="Email Delivery" description="Configure SMTP and email-only routing controls supported by the current agent." heading-id="settings-notifications-email" eyebrow="Notifications" sticky>
                <SettingRow
                  row-id="notifications-email"
                  label="Email notifications"
                  description="Expand email transport fields only when the channel is enabled."
                  :status-label="form.email_alerts_enabled ? 'Enabled' : 'Disabled'"
                  :status-state="form.email_alerts_enabled ? 'ok' : 'muted'"
                  :highlighted="isHighlighted('notifications-email')"
                >
                  <div class="settings-stack-md">
                    <ToggleSwitch input-id="settings-email-enabled" v-model="form.email_alerts_enabled" label="Enable email delivery" />
                    <div v-if="form.email_alerts_enabled" class="settings-grid-two">
                      <TextField input-id="settings-smtp-host" v-model="form.smtp_host" placeholder="smtp.example.com" @blur="touchField('smtp_host')" />
                      <TextField input-id="settings-smtp-port" v-model="form.smtp_port" placeholder="587" @blur="touchField('smtp_port')" />
                      <TextField input-id="settings-smtp-user" v-model="form.smtp_user" placeholder="SMTP username" @blur="touchField('smtp_user')" />
                      <TextField input-id="settings-alert-email" v-model="form.alert_email" type="email" placeholder="alerts@example.com" @blur="touchField('alert_email')" />
                      <SecretField input-id="settings-smtp-pass" v-model="form.smtp_pass" placeholder="SMTP password" @blur="touchField('smtp_pass')" />
                      <SelectField input-id="settings-email-severity" v-model="form.email_severity" :options="severityOptions" />
                    </div>
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="notifications-routing" class="settings-anchor" data-subsection="Routing Rules">
              <SettingsSection title="Routing Rules" description="The current backend only supports email delivery. Telegram, webhook, and Slack routing are held as explicit future gaps." heading-id="settings-notifications-routing" eyebrow="Expansion">
                <SettingRow
                  row-id="notifications-routing"
                  label="Channel routing"
                  description="Map alert classes to delivery channels."
                  status-label="Unsupported today"
                  status-state="warn"
                  :highlighted="isHighlighted('notifications-routing')"
                  footnote="The current agent exposes SMTP only. Use this row as the future routing handoff."
                >
                  <TextAreaField input-id="settings-routing-preview" :model-value="routingPreview" readonly :rows="4" />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'integrations'">
            <div id="integrations-captcha" class="settings-anchor" data-subsection="reCAPTCHA">
              <SettingsSection title="reCAPTCHA" description="Bot protection for the login form." heading-id="settings-integrations-captcha" eyebrow="Integrations" sticky>
                <SettingRow
                  row-id="integrations-recaptcha"
                  label="Require reCAPTCHA"
                  description="When enabled, the login screen requires reCAPTCHA verification."
                  :status-label="form.recaptcha_enabled ? 'Enabled' : 'Disabled'"
                  :status-state="form.recaptcha_enabled ? 'ok' : 'muted'"
                  :highlighted="isHighlighted('integrations-recaptcha')"
                >
                  <div class="settings-stack-md">
                    <ToggleSwitch input-id="settings-recaptcha-enabled" v-model="form.recaptcha_enabled" label="Enable reCAPTCHA on login" />
                    <div v-if="form.recaptcha_enabled" class="settings-grid-two">
                      <TextField input-id="settings-recaptcha-site-key" v-model="form.recaptcha_site_key" placeholder="Site key" @blur="touchField('recaptcha_site_key')" />
                      <SecretField input-id="settings-recaptcha-secret-key" v-model="form.recaptcha_secret_key" placeholder="Secret key" @blur="touchField('recaptcha_secret_key')" />
                    </div>
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="integrations-ip-intelligence" class="settings-anchor" data-subsection="IP Intelligence">
              <SettingsSection title="IP Intelligence" description="Pick the reverse lookup provider used by the audit and alert surfaces." heading-id="settings-integrations-ip" eyebrow="Lookup Provider">
                <SettingRow
                  row-id="integrations-ip-provider"
                  label="IP lookup provider"
                  label-for="settings-ip-provider"
                  description="Choose the IP enrichment provider used in the UI."
                  :highlighted="isHighlighted('integrations-ip-provider')"
                >
                  <div class="settings-stack-md">
                    <SelectField input-id="settings-ip-provider" v-model="form.ip_lookup_provider" :options="ipProviderOptions" />
                    <SecretField
                      v-if="form.ip_lookup_provider === 'ipify'"
                      input-id="settings-ipify-key"
                      v-model="form.ipify_api_key"
                      placeholder="Optional ipify API key"
                      @blur="touchField('ipify_api_key')"
                    />
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'data-storage'">
            <div id="data-database" class="settings-anchor" data-subsection="Database">
              <SettingsSection title="Database" description="Administrative actions for the SentinelCore database." heading-id="settings-data-database" eyebrow="Data & Storage" sticky>
                <SettingRow
                  row-id="data-db-export"
                  label="Database transport"
                  description="Export or replace the current database snapshot."
                  :highlighted="isHighlighted('data-db-export')"
                >
                  <div class="settings-stack-md">
                    <div class="settings-stat-strip">
                      <div class="settings-stat-card">
                        <span>Login attempts</span>
                        <strong>{{ dbStats?.login_attempts ?? '-' }}</strong>
                      </div>
                      <div class="settings-stat-card">
                        <span>Alerts</span>
                        <strong>{{ dbStats?.alerts ?? '-' }}</strong>
                      </div>
                      <div class="settings-stat-card">
                        <span>Manual bans</span>
                        <strong>{{ dbStats?.manual_bans ?? '-' }}</strong>
                      </div>
                    </div>
                    <div class="settings-inline-actions">
                      <AppButton variant="ghost" size="sm" icon="mdi mdi-refresh" label="Refresh Stats" :loading="loadingDbStats" @click="loadDbStats" />
                      <AppButton variant="secondary" size="sm" icon="mdi mdi-database-export" label="Export DB" @click="downloadDb" />
                      <label class="settings-upload-button">
                        <AppButton variant="secondary" size="sm" icon="mdi mdi-database-import" label="Import DB" :loading="importing" />
                        <input ref="dbImportInput" type="file" accept=".db" class="d-none" @change="importDb">
                      </label>
                    </div>
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>

            <div id="data-prune" class="settings-anchor" data-subsection="Retention & Prune">
              <SettingsSection title="Retention & Prune" description="Preview record deletion requests before submitting them." heading-id="settings-data-prune" eyebrow="Retention">
                <SettingRow
                  row-id="data-prune"
                  label="Prune old records"
                  description="Select a table, choose a retention window, then queue the prune request."
                  :highlighted="isHighlighted('data-prune')"
                  footnote="Exact affected-count previews are not exposed by the current agent; table totals are shown instead."
                >
                  <div class="settings-grid-two">
                    <SelectField input-id="settings-prune-table" v-model="pruneWizard.table" :options="pruneTableOptions" />
                    <DurationPicker input-id="settings-prune-retention" v-model="pruneWizard.retention" />
                  </div>
                  <div class="settings-inline-note">Current table total: {{ pruneTableTotal }}</div>
                  <div class="settings-inline-actions">
                    <AppButton variant="danger" size="sm" icon="mdi mdi-delete-sweep" label="Queue prune" :loading="pruning" @click="doPrune" />
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'master-key'">
            <div id="master-key-status" class="settings-anchor" data-subsection="Key Status">
              <SettingsSection title="Master Key" description="Current encryption key posture and recovery guidance." heading-id="settings-master-key" eyebrow="Master Key" sticky>
                <SettingRow
                  row-id="master-key-status"
                  label="Configured key path"
                  description="SentinelCore encrypts secret settings with the path shown below."
                  status-label="Enabled"
                  status-state="ok"
                  :highlighted="isHighlighted('master-key-status')"
                  footnote="Key rotation and restore flows require dedicated backend endpoints before they can be executed safely from this page."
                >
                  <div class="settings-stack-md">
                    <TextField input-id="settings-master-key-path" :model-value="form.secrets_key_path || 'Not configured'" readonly />
                    <button type="button" class="settings-inline-link" @click="openHelp('Master key rotation', `Last rotated: ${formatRotationTime(form.last_master_key_rotation)}. The agent currently exposes the timestamp but not a rotate endpoint.`)">
                      Last rotated: {{ formatRotationTime(form.last_master_key_rotation) }}
                    </button>
                    <div class="settings-inline-actions">
                      <AppButton variant="secondary" size="sm" icon="mdi mdi-database-export" label="Backup database" @click="downloadDb" />
                      <AppButton variant="ghost" size="sm" icon="mdi mdi-timer-sand" label="Rotate endpoint not exposed" disabled />
                    </div>
                  </div>
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'audit-logs'">
            <div id="audit-overview" class="settings-anchor" data-subsection="Audit Overview">
              <SettingsSection title="Audit & Logs" description="Configuration for retention and export is not exposed yet, but the dedicated audit surface is available." heading-id="settings-audit" eyebrow="Audit & Logs" sticky>
                <SettingRow
                  row-id="audit-overview"
                  label="Audit controls"
                  description="Open the audit log page to review live history while backend retention controls are added."
                  status-label="Linked page"
                  status-state="info"
                  :highlighted="isHighlighted('audit-overview')"
                >
                  <AppButton variant="secondary" size="sm" icon="mdi mdi-open-in-new" label="Open Audit Logs" @click="$router.push('/audit-logs')" />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'updates'">
            <div id="updates-policy" class="settings-anchor" data-subsection="Update Policy">
              <SettingsSection title="Updates" description="The update workflow lives in the dedicated Updates page today." heading-id="settings-updates" eyebrow="Updates" sticky>
                <SettingRow
                  row-id="updates-policy"
                  label="Update workflow"
                  description="Review package updates and install logs in the current Updates page."
                  status-label="Linked page"
                  status-state="info"
                  :highlighted="isHighlighted('updates-policy')"
                >
                  <AppButton variant="secondary" size="sm" icon="mdi mdi-open-in-new" label="Open Updates" @click="$router.push('/updates')" />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'advanced'">
            <div id="advanced-raw-config" class="settings-anchor" data-subsection="Raw Config">
              <SettingsSection title="Advanced" description="Raw config editing and feature flags are still backend gaps. This section shows the currently exposed payload instead." heading-id="settings-advanced" eyebrow="Advanced" sticky>
                <SettingRow
                  row-id="advanced-raw-config"
                  label="Current payload preview"
                  description="Review the live settings payload being edited by this page."
                  status-label="Read only"
                  status-state="muted"
                  :highlighted="isHighlighted('advanced-raw-config')"
                >
                  <TextAreaField input-id="settings-raw-preview" :model-value="rawSettingsPreview" readonly :rows="12" />
                </SettingRow>
              </SettingsSection>
            </div>
          </template>

          <template v-else-if="currentSectionId === 'danger-zone'">
            <div id="danger-destructive-actions" class="settings-anchor" data-subsection="Destructive Actions">
              <SettingsSection title="Danger Zone" description="Isolated, high-impact actions with confirmation before execution." heading-id="settings-danger" eyebrow="Danger Zone" sticky>
                <DangerZone
                  heading-id="settings-danger-actions"
                  title="High-impact actions"
                  description="Each action below requires an explicit confirm flow."
                  :items="dangerItems"
                  @action="runDangerAction"
                />
              </SettingsSection>
            </div>
          </template>
        </div>

        <SaveBar
          :visible="hasDirtyChanges || saveState === 'error'"
          :changes="dirtyFields"
          :saving="isSaving"
          :disabled="!canSave"
          :state-label="saveStateLabel"
          @save="saveAll"
          @discard="discardChanges"
          @review="reviewDrawerOpen = true"
        />
      </div>
    </div>

    <DetailDrawer v-model="reviewDrawerOpen" title="Review changes" subtitle="Current draft vs saved state">
      <div class="settings-drawer-list">
        <div v-for="change in dirtyFields" :key="change.id" class="settings-drawer-row">
          <div>
            <strong>{{ change.label }}</strong>
            <p>{{ change.breadcrumb }}</p>
          </div>
          <div class="settings-drawer-diff">
            <code>{{ formatDiffValue(change.oldValue) }}</code>
            <i class="mdi mdi-arrow-right"></i>
            <code>{{ formatDiffValue(change.newValue) }}</code>
          </div>
        </div>
      </div>
    </DetailDrawer>

    <DetailDrawer v-model="conflictDrawerOpen" title="Concurrent edits detected" subtitle="Another tab changed one or more settings before this save.">
      <div class="settings-drawer-list">
        <div v-for="change in conflictDiffs" :key="change.id" class="settings-drawer-row">
          <div>
            <strong>{{ change.label }}</strong>
            <p>{{ change.breadcrumb }}</p>
          </div>
          <div class="settings-drawer-diff settings-drawer-diff--stacked">
            <code>saved: {{ formatDiffValue(change.savedValue) }}</code>
            <code>remote: {{ formatDiffValue(change.remoteValue) }}</code>
            <code>draft: {{ formatDiffValue(change.draftValue) }}</code>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="settings-inline-actions w-100 justify-content-between">
          <AppButton variant="ghost" size="sm" label="Reload remote" @click="reloadRemoteState" />
          <AppButton variant="primary" size="sm" label="Overwrite remote" @click="forceSaveAfterConflict" />
        </div>
      </template>
    </DetailDrawer>

    <DetailDrawer v-model="helpDrawerOpen" :title="helpDrawer.title || 'Help'" subtitle="Settings guidance">
      <div class="settings-help-copy">{{ helpDrawer.body }}</div>
    </DetailDrawer>
  </div>
</template>

<script>
import QRCode from 'qrcode'
import PageHeader from '@/components/page-header.vue'
import AppButton from '@/components/ui/app-button.vue'
import DetailDrawer from '@/components/ui/detail-drawer.vue'
import SettingsSection from '@/components/settings/settings-section.vue'
import SettingRow from '@/components/settings/setting-row.vue'
import SaveBar from '@/components/settings/save-bar.vue'
import DangerZone from '@/components/settings/danger-zone.vue'
import TextField from '@/components/settings/fields/text-field.vue'
import NumberField from '@/components/settings/fields/number-field.vue'
import TextAreaField from '@/components/settings/fields/textarea-field.vue'
import ToggleSwitch from '@/components/settings/fields/toggle-switch.vue'
import SelectField from '@/components/settings/fields/select-field.vue'
import SegmentedControl from '@/components/settings/fields/segmented-control.vue'
import KeyValueList from '@/components/settings/fields/key-value-list.vue'
import PinField from '@/components/settings/fields/pin-field.vue'
import SecretField from '@/components/settings/fields/secret-field.vue'
import DurationPicker from '@/components/settings/fields/duration-picker.vue'
import api from '@/services/api'
import { SETTINGS_SECTIONS, SETTINGS_SEARCH_ENTRIES, getSection } from './schema'

const LOCAL_PREFS_KEY = 'sc_settings_local_prefs'

function safeParse(value, fallback) {
  try {
    return JSON.parse(value ?? '')
  } catch {
    return fallback
  }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function normalizeBoolean(value) {
  if (typeof value === 'boolean') return value
  return value === 'true' || value === '1'
}

function detectTimezone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
}

function defaultLocalPrefs() {
  return {
    ui_theme_default: localStorage.getItem('sc_theme') || 'system',
    ui_language: 'en',
    ui_timezone: detectTimezone(),
    email_severity: 'critical',
    email_quiet_start: '22:00',
    email_quiet_end: '07:00'
  }
}

function normalizeGateDuration(daysValue) {
  const days = Number(daysValue)
  if (!Number.isFinite(days) || days <= 0) {
    return { value: 0, unit: 'days' }
  }
  return { value: days, unit: 'days' }
}

function serializeGateDuration(duration) {
  const value = Number(duration?.value || 0)
  const unit = duration?.unit || 'days'
  if (value <= 0) return '0'
  if (unit === 'minutes') return String(Math.max(1, Math.ceil(value / (60 * 24))))
  if (unit === 'hours') return String(Math.max(1, Math.ceil(value / 24)))
  return String(Math.max(1, Math.ceil(value)))
}

function normalizeRemoteSettings(data = {}) {
  return {
    alert_email: data.alert_email || '',
    brute_force_threshold: Number(data.brute_force_threshold || 5),
    email_alerts_enabled: normalizeBoolean(data.email_alerts_enabled),
    smtp_host: data.smtp_host || '',
    smtp_port: data.smtp_port || '587',
    smtp_user: data.smtp_user || '',
    smtp_pass: data.smtp_pass || '',
    smtp_pass_configured: data.smtp_pass_configured || '0',
    secret_path: data.secret_path || 'sentinel-core',
    gate_expiry: normalizeGateDuration(data.gate_expiry_days || 0),
    recaptcha_enabled: normalizeBoolean(data.recaptcha_enabled),
    recaptcha_site_key: data.recaptcha_site_key || '',
    recaptcha_secret_key: data.recaptcha_secret_key || '',
    recaptcha_secret_key_configured: data.recaptcha_secret_key_configured || '0',
    ip_lookup_provider: data.ip_lookup_provider || 'none',
    ipify_api_key: data.ipify_api_key || '',
    ipify_api_key_configured: data.ipify_api_key_configured || '0',
    secrets_key_path: data.secrets_key_path || '',
    last_master_key_rotation: data.last_master_key_rotation || ''
  }
}

function localPrefsFromStorage() {
  return {
    ...defaultLocalPrefs(),
    ...safeParse(localStorage.getItem(LOCAL_PREFS_KEY), {})
  }
}

function validateHostname(value) {
  return /^[a-zA-Z0-9.-]+$/.test(value || '')
}

export default {
  name: 'SettingsPage',
  components: {
    PageHeader,
    AppButton,
    DetailDrawer,
    SettingsSection,
    SettingRow,
    SaveBar,
    DangerZone,
    TextField,
    NumberField,
    TextAreaField,
    ToggleSwitch,
    SelectField,
    SegmentedControl,
    KeyValueList,
    PinField,
    SecretField,
    DurationPicker
  },
  data() {
    return {
      loading: true,
      searchQuery: '',
      searchResultsOpen: false,
      searchSelectionIndex: 0,
      activeSubsectionLabel: '',
      highlightedRowId: '',
      form: {
        ...normalizeRemoteSettings(),
        ...defaultLocalPrefs()
      },
      remoteBaseline: normalizeRemoteSettings(),
      localBaseline: defaultLocalPrefs(),
      me: { totp_enabled: false },
      totpSetup: { secret: '', otpauthUrl: '', verifyCode: '', loading: false },
      dbStats: null,
      loadingDbStats: false,
      lockEnabled: false,
      lockPinSet: false,
      lockForm: {
        pin: '',
        confirmPin: '',
        lockOnBlur: false
      },
      settingPin: false,
      pruneWizard: {
        table: 'login_attempts',
        retention: { value: 30, unit: 'days' }
      },
      pruning: false,
      importing: false,
      isSaving: false,
      saveState: 'idle',
      reviewDrawerOpen: false,
      conflictDrawerOpen: false,
      helpDrawerOpen: false,
      helpDrawer: { title: '', body: '' },
      conflictDiffs: [],
      pendingRemoteState: null,
      observer: null,
      pendingJumpTarget: '',
      themeOptions: [
        { label: 'System', value: 'system' },
        { label: 'Light', value: 'light' },
        { label: 'Dark', value: 'dark' }
      ],
      languageOptions: [
        { label: 'English', value: 'en' },
        { label: 'Deutsch', value: 'de' },
        { label: 'Arabic', value: 'ar' }
      ],
      severityOptions: [
        { label: 'Info and above', value: 'info' },
        { label: 'Warnings and above', value: 'warn' },
        { label: 'Errors and above', value: 'error' },
        { label: 'Critical only', value: 'critical' }
      ],
      ipProviderOptions: [
        { label: 'Disabled', value: 'none' },
        { label: 'ipify', value: 'ipify' },
        { label: 'ip-api', value: 'ip-api' }
      ],
      pruneTableOptions: [
        { label: 'Login attempts', value: 'login_attempts' },
        { label: 'Alerts', value: 'alerts' }
      ]
    }
  },
  computed: {
    settingsSections() {
      return SETTINGS_SECTIONS
    },
    currentSectionId() {
      const raw = String(this.$route.params.section || 'general')
      return SETTINGS_SECTIONS.some(section => section.id === raw) ? raw : 'general'
    },
    currentSection() {
      return getSection(this.currentSectionId)
    },
    pageHeaderItems() {
      return [
        { text: 'Settings', href: '/settings/general', icon: 'mdi mdi-cog-outline' },
        { text: this.currentSection.label, active: true, icon: this.currentSection.icon }
      ]
    },
    runtimeHostname() {
      return this.$store.getters['metrics/snap']?.hostname || window.location.hostname || 'Unknown'
    },
    runtimeListenAddress() {
      return 'Managed by agent configuration'
    },
    secretUrlPreview() {
      return `${window.location.origin}/${this.form.secret_path || 'sentinel-core'}/`
    },
    filteredSearchResults() {
      const query = this.searchQuery.trim().toLowerCase()
      if (!query) return []
      const tokens = query.split(/\s+/).filter(Boolean)
      return SETTINGS_SEARCH_ENTRIES
        .map(entry => ({
          ...entry,
          breadcrumb: `Settings > ${getSection(entry.section).label} > ${getSection(entry.section).groups.find(group => group.id === entry.anchor.split('-').slice(1).join('-'))?.label || entry.label}`
        }))
        .filter(entry => {
          const haystack = `${entry.label} ${entry.description} ${entry.key} ${entry.keywords.join(' ')}`.toLowerCase()
          return tokens.every(token => haystack.includes(token))
        })
        .slice(0, 8)
    },
    remotePayload() {
      return {
        alert_email: this.form.alert_email,
        brute_force_threshold: this.form.brute_force_threshold,
        email_alerts_enabled: this.form.email_alerts_enabled,
        smtp_host: this.form.smtp_host,
        smtp_port: this.form.smtp_port,
        smtp_user: this.form.smtp_user,
        smtp_pass: this.form.smtp_pass,
        secret_path: this.form.secret_path,
        gate_expiry_days: serializeGateDuration(this.form.gate_expiry),
        recaptcha_enabled: this.form.recaptcha_enabled,
        recaptcha_site_key: this.form.recaptcha_site_key,
        recaptcha_secret_key: this.form.recaptcha_secret_key,
        ip_lookup_provider: this.form.ip_lookup_provider,
        ipify_api_key: this.form.ipify_api_key
      }
    },
    localPayload() {
      return {
        ui_theme_default: this.form.ui_theme_default,
        ui_language: this.form.ui_language,
        ui_timezone: this.form.ui_timezone,
        email_severity: this.form.email_severity,
        email_quiet_start: this.form.email_quiet_start,
        email_quiet_end: this.form.email_quiet_end
      }
    },
    validationErrors() {
      const errors = {}
      if (this.form.alert_email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(this.form.alert_email)) {
        errors.alert_email = 'Enter a valid email address.'
      }
      if (!Number.isFinite(Number(this.form.brute_force_threshold)) || Number(this.form.brute_force_threshold) < 1 || Number(this.form.brute_force_threshold) > 100) {
        errors.brute_force_threshold = 'Use a value between 1 and 100.'
      }
      if (this.form.recaptcha_enabled && !this.form.recaptcha_site_key.trim()) {
        errors.recaptcha_site_key = 'Provide the site key when reCAPTCHA is enabled.'
      }
      if (this.form.email_alerts_enabled) {
        if (!this.form.smtp_host.trim()) errors.smtp_host = 'SMTP host is required when email delivery is enabled.'
        if (!this.form.smtp_port.trim()) errors.smtp_port = 'SMTP port is required when email delivery is enabled.'
      }
      if (!/^[a-z0-9][a-z0-9-]*$/i.test(this.form.secret_path || '')) {
        errors.secret_path = 'Use letters, numbers, and dashes only.'
      }
      if (!this.form.ui_timezone.trim()) {
        errors.ui_timezone = 'Timezone cannot be empty.'
      }
      if (this.lockEnabled && this.lockForm.pin && this.lockForm.pin.length !== 6) {
        errors.lock_pin = 'PIN must be exactly six digits.'
      }
      return errors
    },
    dirtyFields() {
      const changes = []
      const remoteEntries = [
        ['general-admin-email', 'alert_email', 'Admin email', 'Settings > General > Runtime'],
        ['security-login-attempts', 'brute_force_threshold', 'Max login attempts', 'Settings > Security > Login Protection'],
        ['access-secret-gate', 'secret_path', 'Secret URL', 'Settings > Access Control > Secret Link Gate'],
        ['access-secret-gate-expiry', 'gate_expiry', 'Gate unlock duration', 'Settings > Access Control > Secret Link Gate'],
        ['notifications-email', 'email_alerts_enabled', 'Email notifications', 'Settings > Notifications > Email Delivery'],
        ['notifications-email-host', 'smtp_host', 'SMTP host', 'Settings > Notifications > Email Delivery'],
        ['notifications-email-port', 'smtp_port', 'SMTP port', 'Settings > Notifications > Email Delivery'],
        ['notifications-email-user', 'smtp_user', 'SMTP user', 'Settings > Notifications > Email Delivery'],
        ['notifications-email-pass', 'smtp_pass', 'SMTP password', 'Settings > Notifications > Email Delivery'],
        ['integrations-recaptcha', 'recaptcha_enabled', 'Require reCAPTCHA', 'Settings > Integrations > reCAPTCHA'],
        ['integrations-recaptcha-site', 'recaptcha_site_key', 'reCAPTCHA site key', 'Settings > Integrations > reCAPTCHA'],
        ['integrations-recaptcha-secret', 'recaptcha_secret_key', 'reCAPTCHA secret key', 'Settings > Integrations > reCAPTCHA'],
        ['integrations-ip-provider', 'ip_lookup_provider', 'IP lookup provider', 'Settings > Integrations > IP Intelligence'],
        ['integrations-ipify-key', 'ipify_api_key', 'ipify API key', 'Settings > Integrations > IP Intelligence']
      ]

      remoteEntries.forEach(([id, key, label, breadcrumb]) => {
        const current = key === 'gate_expiry' ? serializeGateDuration(this.form.gate_expiry) : this.form[key]
        const baseline = key === 'gate_expiry' ? serializeGateDuration(this.remoteBaseline.gate_expiry) : this.remoteBaseline[key]
        if (JSON.stringify(current) !== JSON.stringify(baseline)) {
          changes.push({ id, key, label, breadcrumb, oldValue: baseline, newValue: current })
        }
      })

      const localEntries = [
        ['general-theme', 'ui_theme_default', 'Default theme', 'Settings > General > Preferences'],
        ['general-language', 'ui_language', 'Language', 'Settings > General > Preferences'],
        ['general-timezone', 'ui_timezone', 'Timezone', 'Settings > General > Preferences']
      ]
      localEntries.forEach(([id, key, label, breadcrumb]) => {
        if (JSON.stringify(this.form[key]) !== JSON.stringify(this.localBaseline[key])) {
          changes.push({ id, key, label, breadcrumb, oldValue: this.localBaseline[key], newValue: this.form[key] })
        }
      })

      return changes
    },
    hasDirtyChanges() {
      return this.dirtyFields.length > 0
    },
    canSave() {
      return this.hasDirtyChanges && !this.isSaving && !this.dirtyFields.some(change => this.validationErrors[change.key])
    },
    saveStateLabel() {
      if (this.isSaving) return 'Saving changes...'
      if (this.saveState === 'error') return 'Save failed. Review the highlighted fields.'
      return `Unsaved changes (${this.dirtyFields.length})`
    },
    pruneTableTotal() {
      if (!this.dbStats) return 'Unavailable'
      return this.pruneWizard.table === 'alerts' ? this.dbStats.alerts : this.dbStats.login_attempts
    },
    rawSettingsPreview() {
      return JSON.stringify({ ...this.remotePayload, ...this.localPayload }, null, 2)
    },
    routingPreview() {
      return [
        'Brute-force activity -> Email only',
        `Delivery threshold -> ${this.form.email_severity}`,
        `Quiet hours -> ${this.form.email_quiet_start} to ${this.form.email_quiet_end}`,
        'Telegram / Webhook / Slack routes require backend support.'
      ].join('\n')
    },
    allowlistPreview() {
      return [
        '# Example only',
        '10.0.0.0/24 -> office LAN',
        '203.0.113.18/32 -> bastion host',
        '',
        'Agent-side enforcement is not exposed by the current backend.'
      ].join('\n')
    },
    dangerItems() {
      return [
        {
          id: 'disable-2fa',
          label: 'Disable 2FA',
          description: 'Remove the current TOTP requirement for this operator account.',
          actionLabel: this.me.totp_enabled ? 'Disable 2FA' : '2FA already off',
          variant: 'danger',
          badge: this.me.totp_enabled ? { label: 'Medium risk', state: 'warn' } : { label: 'Inactive', state: 'muted' }
        },
        {
          id: 'remove-lock-pin',
          label: 'Remove lock screen PIN',
          description: 'Delete the local unlock PIN for this account.',
          actionLabel: this.lockPinSet ? 'Remove PIN' : 'No PIN set',
          variant: 'danger',
          badge: this.lockPinSet ? { label: 'Low risk', state: 'warn' } : { label: 'Inactive', state: 'muted' }
        },
        {
          id: 'regenerate-secret-path',
          label: 'Regenerate secret link gate',
          description: 'Generate a new hidden entry path and invalidate the previous one.',
          actionLabel: 'Regenerate',
          variant: 'danger',
          badge: { label: 'High risk', state: 'critical' }
        },
        {
          id: 'restart-daemon',
          label: 'Restart daemon',
          description: 'A restart endpoint is not exposed yet; this remains a guarded placeholder.',
          actionLabel: 'Unavailable',
          variant: 'secondary',
          badge: { label: 'Agent gap', state: 'muted' }
        }
      ]
    }
  },
  watch: {
    '$route.params.section': {
      immediate: true,
      handler(value) {
        if (!SETTINGS_SECTIONS.some(section => section.id === value)) {
          this.$router.replace('/settings/general')
          return
        }
        this.$nextTick(() => {
          this.observeAnchors()
          if (this.pendingJumpTarget) {
            this.focusRow(this.pendingJumpTarget)
            this.pendingJumpTarget = ''
          }
        })
      }
    }
  },
  async mounted() {
    await this.bootstrap()
    window.addEventListener('beforeunload', this.handleBeforeUnload)
    document.addEventListener('keydown', this.handleKeyDown, true)
    document.addEventListener('click', this.handleOutsideSearch, true)
  },
  beforeUnmount() {
    window.removeEventListener('beforeunload', this.handleBeforeUnload)
    document.removeEventListener('keydown', this.handleKeyDown, true)
    document.removeEventListener('click', this.handleOutsideSearch, true)
    this.observer?.disconnect()
  },
  beforeRouteLeave(to, from, next) {
    this.confirmNavigation(next)
  },
  beforeRouteUpdate(to, from, next) {
    if (to.params.section !== from.params.section) {
      this.confirmNavigation(next)
      return
    }
    next()
  },
  methods: {
    sectionRoute(sectionId) {
      return `/settings/${sectionId}`
    },
    goToSection(sectionId) {
      this.$router.push(this.sectionRoute(sectionId))
    },
    errorFor(key) {
      return this.validationErrors[key] || ''
    },
    isHighlighted(rowId) {
      return this.highlightedRowId === rowId
    },
    touchField() {
      this.saveState = 'idle'
    },
    async bootstrap() {
      this.loading = true
      try {
        const [settingsResponse, meResponse, dbStatsResponse, lockResponse] = await Promise.allSettled([
          api.getSettings(),
          api.getMe(),
          api.getDbStats(),
          api.getLockSettings()
        ])
        const remote = normalizeRemoteSettings(settingsResponse.status === 'fulfilled' ? settingsResponse.value.data : {})
        const local = localPrefsFromStorage()
        this.form = { ...remote, ...local }
        this.remoteBaseline = clone(remote)
        this.localBaseline = clone(local)
        this.me = meResponse.status === 'fulfilled' ? (meResponse.value.data || { totp_enabled: false }) : { totp_enabled: false }
        this.dbStats = dbStatsResponse.status === 'fulfilled' ? (dbStatsResponse.value.data || null) : null
        this.lockEnabled = lockResponse.status === 'fulfilled' ? !!lockResponse.value.data?.enabled : false
        this.lockPinSet = lockResponse.status === 'fulfilled' ? !!lockResponse.value.data?.pinSet : false
      } finally {
        this.loading = false
      }
    },
    observeAnchors() {
      this.observer?.disconnect()
      const anchors = this.$refs.contentRoot?.querySelectorAll('.settings-anchor') || []
      if (!anchors.length) return
      this.observer = new IntersectionObserver(entries => {
        const visible = entries.filter(entry => entry.isIntersecting)
        if (!visible.length) return
        this.activeSubsectionLabel = visible[0].target.dataset.subsection || ''
      }, { rootMargin: '-110px 0px -60% 0px', threshold: [0.2, 0.5] })
      anchors.forEach(anchor => this.observer.observe(anchor))
      this.activeSubsectionLabel = anchors[0].dataset.subsection || ''
    },
    moveSearchSelection(direction) {
      if (!this.filteredSearchResults.length) return
      this.searchSelectionIndex = (this.searchSelectionIndex + direction + this.filteredSearchResults.length) % this.filteredSearchResults.length
    },
    activateSelectedSearchResult() {
      if (!this.filteredSearchResults.length) return
      this.jumpToSearchResult(this.filteredSearchResults[this.searchSelectionIndex])
    },
    closeSearchResults() {
      this.searchResultsOpen = false
    },
    handleOutsideSearch(event) {
      if (!this.$el?.contains(event.target)) return
      const search = this.$el.querySelector('.settings-search')
      if (search && !search.contains(event.target)) {
        this.searchResultsOpen = false
      }
    },
    jumpToSearchResult(result) {
      this.searchResultsOpen = false
      this.searchSelectionIndex = 0
      this.pendingJumpTarget = result.id
      this.$router.push(this.sectionRoute(result.section)).then(() => {
        this.$nextTick(() => {
          this.focusRow(result.id)
        })
      })
    },
    focusRow(rowId) {
      const element = document.getElementById(rowId)
      if (!element) return
      element.scrollIntoView({ block: 'center', behavior: 'smooth' })
      this.highlightedRowId = rowId
      window.setTimeout(() => {
        if (this.highlightedRowId === rowId) {
          this.highlightedRowId = ''
        }
      }, 2000)
    },
    handleKeyDown(event) {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        event.stopPropagation()
        this.searchResultsOpen = true
        this.$nextTick(() => this.$refs.searchInput?.focus())
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
        if (!this.canSave) return
        event.preventDefault()
        this.saveAll()
      }
      if (event.key === 'Escape' && this.hasDirtyChanges && !this.reviewDrawerOpen && !this.conflictDrawerOpen) {
        this.discardChanges()
      }
    },
    handleBeforeUnload(event) {
      if (!this.hasDirtyChanges) return
      event.preventDefault()
      event.returnValue = ''
    },
    confirmNavigation(next) {
      if (!this.hasDirtyChanges) {
        next()
        return
      }
      const shouldLeave = window.confirm('You have unsaved settings changes. Press OK to discard them or Cancel to stay on this page.')
      if (shouldLeave) {
        next()
      } else {
        next(false)
      }
    },
    async saveAll(force = false) {
      this.isSaving = true
      this.saveState = 'saving'
      try {
        if (!force) {
          const { data } = await api.getSettings()
          const latest = normalizeRemoteSettings(data)
          const remoteConflicts = Object.keys(this.remotePayload).filter(key => {
            const baseline = key === 'gate_expiry_days' ? serializeGateDuration(this.remoteBaseline.gate_expiry) : (this.remoteBaseline[key] ?? '')
            const latestValue = key === 'gate_expiry_days' ? serializeGateDuration(latest.gate_expiry) : (latest[key] ?? '')
            const draftValue = this.remotePayload[key]
            return JSON.stringify(baseline) !== JSON.stringify(latestValue) && JSON.stringify(baseline) !== JSON.stringify(draftValue)
          })

          if (remoteConflicts.length) {
            this.pendingRemoteState = latest
            this.conflictDiffs = remoteConflicts.map(key => ({
              id: key,
              label: key.replace(/_/g, ' '),
              breadcrumb: 'Agent-backed setting',
              savedValue: this.remoteBaseline[key] ?? '',
              remoteValue: latest[key] ?? '',
              draftValue: this.remotePayload[key] ?? ''
            }))
            this.conflictDrawerOpen = true
            this.saveState = 'error'
            return
          }
        }

        const dirtyRemoteKeys = this.dirtyFields.map(change => change.key).filter(key => key in this.remotePayload)
        const payload = {}
        dirtyRemoteKeys.forEach(key => {
          if (key === 'gate_expiry') {
            payload.gate_expiry_days = serializeGateDuration(this.form.gate_expiry)
          } else {
            payload[key] = this.remotePayload[key]
          }
        })
        if (Object.keys(payload).length) {
          await api.updateSettings(payload)
        }
        localStorage.setItem(LOCAL_PREFS_KEY, JSON.stringify(this.localPayload))
        this.remoteBaseline = clone(normalizeRemoteSettings({ ...this.remoteBaseline, ...payload, gate_expiry_days: payload.gate_expiry_days || serializeGateDuration(this.form.gate_expiry) }))
        this.localBaseline = clone(this.localPayload)
        this.form = { ...this.form, ...this.localPayload, ...this.remoteBaseline }
        this.saveState = 'saved'
      } catch (error) {
        console.error('Settings save failed:', error)
        this.saveState = 'error'
      } finally {
        this.isSaving = false
      }
    },
    discardChanges() {
      this.form = {
        ...clone(this.remoteBaseline),
        ...clone(this.localBaseline)
      }
      this.saveState = 'idle'
    },
    reloadRemoteState() {
      if (!this.pendingRemoteState) return
      this.remoteBaseline = clone(this.pendingRemoteState)
      this.form = { ...this.form, ...clone(this.pendingRemoteState) }
      this.conflictDrawerOpen = false
      this.pendingRemoteState = null
      this.saveState = 'idle'
    },
    forceSaveAfterConflict() {
      this.conflictDrawerOpen = false
      this.saveAll(true)
    },
    formatDiffValue(value) {
      if (value === '' || value == null) return '(empty)'
      if (typeof value === 'object') return JSON.stringify(value)
      return String(value)
    },
    formatRotationTime(ts) {
      if (!ts) return 'Never rotated'
      const asNumber = Number(ts)
      if (!Number.isFinite(asNumber) || asNumber <= 0) return 'Unknown'
      return new Date(asNumber * 1000).toLocaleString()
    },
    openHelp(title, body) {
      this.helpDrawer = { title, body }
      this.helpDrawerOpen = true
    },
    generateSecretPath() {
      const random = Math.random().toString(36).slice(2, 10)
      this.form.secret_path = `sentinel-${random}`
    },
    async initSetup2FA() {
      this.totpSetup.loading = true
      try {
        const { data } = await api.setup2fa()
        this.totpSetup.secret = data.secret
        this.totpSetup.otpauthUrl = data.otpauth_url
        await this.$nextTick()
        QRCode.toCanvas(this.$refs.qrCanvas, data.otpauth_url, {
          width: 220,
          margin: 1,
          color: { dark: '#000000', light: '#ffffff' }
        })
      } catch (error) {
        console.error('Could not generate QR code:', error)
      } finally {
        this.totpSetup.loading = false
      }
    },
    async enable2FA() {
      if (!this.totpSetup.verifyCode) return
      await api.enable2fa(this.totpSetup.secret, this.totpSetup.verifyCode)
      this.me.totp_enabled = true
      this.totpSetup = { secret: '', otpauthUrl: '', verifyCode: '', loading: false }
    },
    setLockEnabled(value) {
      this.lockEnabled = value
    },
    async saveLockPin() {
      if (!/^\d{6}$/.test(this.lockForm.pin)) {
        this.openHelp('PIN validation', 'PIN must be exactly 6 digits.')
        return
      }
      if (this.lockForm.pin !== this.lockForm.confirmPin) {
        this.openHelp('PIN validation', 'PIN values do not match.')
        return
      }
      this.settingPin = true
      try {
        await api.saveLockPin(this.lockForm.pin, this.lockEnabled)
        this.lockPinSet = true
        this.lockForm.pin = ''
        this.lockForm.confirmPin = ''
      } finally {
        this.settingPin = false
      }
    },
    async loadDbStats() {
      this.loadingDbStats = true
      try {
        const { data } = await api.getDbStats()
        this.dbStats = data
      } finally {
        this.loadingDbStats = false
      }
    },
    async doPrune() {
      const confirmed = window.confirm(`Prune ${this.pruneWizard.table} records older than ${this.pruneWizard.retention.value} ${this.pruneWizard.retention.unit}?`)
      if (!confirmed) return
      this.pruning = true
      try {
        const days = serializeGateDuration(this.pruneWizard.retention)
        await api.pruneDb(this.pruneWizard.table, Number(days))
        await this.loadDbStats()
      } finally {
        this.pruning = false
      }
    },
    async downloadDb() {
      const { data } = await api.exportDb()
      const url = URL.createObjectURL(new Blob([data], { type: 'application/octet-stream' }))
      const link = document.createElement('a')
      link.href = url
      link.download = 'sentinelcore.db'
      link.click()
      URL.revokeObjectURL(url)
    },
    async importDb(event) {
      const file = event.target.files?.[0]
      if (!file) return
      const confirmed = window.confirm('Importing a database replaces the current one. Continue?')
      if (!confirmed) {
        this.$refs.dbImportInput.value = ''
        return
      }
      this.importing = true
      try {
        const formData = new FormData()
        formData.append('db', file)
        await api.importDb(formData)
        await this.loadDbStats()
      } finally {
        this.importing = false
        this.$refs.dbImportInput.value = ''
      }
    },
    async runDangerAction(item) {
      if (item.id === 'disable-2fa') {
        if (!this.me.totp_enabled) return
        const code = window.prompt('Enter your current 2FA code to disable 2FA')
        if (!code) return
        await api.disable2fa(code)
        this.me.totp_enabled = false
        return
      }
      if (item.id === 'remove-lock-pin') {
        if (!this.lockPinSet) return
        const confirmed = window.confirm('Remove the lock screen PIN?')
        if (!confirmed) return
        await api.clearLockPin()
        this.lockPinSet = false
        this.lockEnabled = false
        return
      }
      if (item.id === 'regenerate-secret-path') {
        const phrase = window.prompt(`Type ${this.runtimeHostname} to regenerate the secret path`) || ''
        if (phrase !== this.runtimeHostname) return
        this.generateSecretPath()
        await this.saveAll(true)
        return
      }
      this.openHelp('Unavailable action', 'The current agent does not expose a safe restart or factory-reset endpoint yet.')
    }
  }
}
</script>

<style scoped>
.settings-page {
  display: grid;
  gap: var(--space-20);
}

.settings-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: var(--space-20);
  align-items: start;
}

.settings-rail {
  position: sticky;
  top: calc(72px + env(safe-area-inset-top));
  padding: var(--space-16);
}

.settings-rail__title {
  font-size: var(--font-size-16);
  font-weight: 600;
  color: var(--text-primary);
}

.settings-rail__summary,
.settings-toolbar__description,
.settings-inline-note,
.settings-help-copy,
.settings-drawer-row p,
.settings-stat-card span,
.settings-upload-button,
.settings-inline-link {
  color: var(--text-secondary);
  font-size: var(--font-size-13);
}

.settings-nav {
  display: grid;
  gap: var(--space-4);
  margin-top: var(--space-16);
}

.settings-nav__item {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: 0.75rem 0.85rem;
  border-radius: var(--radius-md);
  text-decoration: none;
  color: var(--text-secondary);
}

.settings-nav__item.active,
.settings-nav__item:hover {
  background: color-mix(in srgb, var(--accent) 12%, var(--surface-2));
  color: var(--text-primary);
}

.settings-main {
  display: grid;
  gap: var(--space-16);
}

.settings-mobile-nav {
  display: none;
  padding: var(--space-12);
}

.settings-mobile-nav__select {
  width: 100%;
  min-height: 44px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-2);
  color: var(--text-primary);
  padding: 0.75rem 0.9rem;
}

.settings-toolbar {
  position: sticky;
  top: calc(72px + env(safe-area-inset-top));
  z-index: 8;
  padding: var(--space-16);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-16);
}

.settings-breadcrumbs {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
  font-size: var(--font-size-13);
  color: var(--text-secondary);
}

.settings-breadcrumbs a {
  color: var(--accent);
  text-decoration: none;
}

.settings-search {
  position: relative;
  width: min(380px, 100%);
}

.settings-search__field {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  min-height: 44px;
  padding: 0 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-2);
}

.settings-search__field input {
  flex: 1;
  border: 0;
  background: transparent;
  color: var(--text-primary);
}

.settings-search__field input:focus {
  outline: none;
}

.settings-search__results {
  position: absolute;
  top: calc(100% + var(--space-8));
  right: 0;
  width: 100%;
  max-height: 360px;
  overflow: auto;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--surface-1);
  box-shadow: 0 20px 44px color-mix(in srgb, #000 20%, transparent);
}

.settings-search__result {
  width: 100%;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--space-12) var(--space-14);
  border: 0;
  background: transparent;
  text-align: left;
}

.settings-search__result.active,
.settings-search__result:hover {
  background: color-mix(in srgb, var(--accent) 12%, var(--surface-1));
}

.settings-search__result strong,
.settings-drawer-row strong,
.settings-stat-card strong,
.settings-inline-link {
  color: var(--text-primary);
}

.settings-search__result p,
.settings-search__empty {
  margin: var(--space-4) 0 0;
  color: var(--text-secondary);
  font-size: var(--font-size-12);
}

.settings-search__result span {
  color: var(--text-tertiary);
  font-size: var(--font-size-11);
  text-align: right;
}

.settings-content {
  display: grid;
  gap: var(--space-20);
}

.settings-anchor {
  display: grid;
  gap: var(--space-16);
}

.settings-grid-two {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-10);
}

.settings-stack-md {
  display: grid;
  gap: var(--space-14);
}

.settings-stack-sm {
  display: grid;
  gap: var(--space-10);
}

.settings-inline-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-8);
}

.settings-setup-block {
  display: grid;
  gap: var(--space-12);
}

.settings-qr {
  width: 220px;
  max-width: 100%;
  border-radius: var(--radius-md);
  background: #fff;
  padding: 0.5rem;
}

.settings-stat-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-10);
}

.settings-stat-card {
  padding: var(--space-14);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-1);
}

.settings-stat-card strong {
  display: block;
  margin-top: var(--space-6);
  font-size: var(--font-size-24);
}

.settings-upload-button {
  position: relative;
}

.settings-upload-button input {
  position: absolute;
  inset: 0;
  opacity: 0;
}

.settings-inline-link {
  border: 0;
  background: transparent;
  padding: 0;
  text-align: left;
}

.settings-drawer-list {
  display: grid;
  gap: var(--space-12);
}

.settings-drawer-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-16);
  padding: var(--space-12);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  background: var(--surface-2);
}

.settings-drawer-diff {
  display: inline-flex;
  align-items: center;
  gap: var(--space-8);
  flex-wrap: wrap;
}

.settings-drawer-diff code {
  padding: 0.25rem 0.4rem;
  border-radius: var(--radius-sm);
}

.settings-drawer-diff--stacked {
  display: grid;
}

@media (max-width: 1024px) {
  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-rail {
    display: none;
  }

  .settings-mobile-nav {
    display: block;
  }
}

@media (max-width: 768px) {
  .settings-toolbar {
    top: calc(64px + env(safe-area-inset-top));
    flex-direction: column;
  }

  .settings-search {
    width: 100%;
  }

  .settings-grid-two,
  .settings-stat-strip {
    grid-template-columns: 1fr;
  }

  .settings-drawer-row {
    flex-direction: column;
  }
}

@media (prefers-reduced-motion: reduce) {
  .settings-search__results,
  .settings-nav__item,
  .settings-content,
  .settings-toolbar {
    scroll-behavior: auto;
    transition: none !important;
  }
}
</style>
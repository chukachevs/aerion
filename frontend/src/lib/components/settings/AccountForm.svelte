<script lang="ts">
  import Icon from '@iconify/svelte'
  import { Button } from '$lib/components/ui/button'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import * as Select from '$lib/components/ui/select'
  import { ColorPicker } from '$lib/components/ui/color-picker'
  import {
    providers,
    detectProvider,
    getProvider,
    getCustomProvider,
    securityOptions,
    syncPeriodOptions,
    syncIntervalOptions,
    isOAuthProvider,
    allowsPasswordFallback,
    getOAuthProviderType,
    type EmailProvider,
    type SecurityType,
    type OAuthProvider,
  } from '$lib/config/providers'
  import { oauthStore } from '$lib/stores/oauth.svelte'
  // @ts-ignore - wailsjs path
  import { account } from '../../../../wailsjs/go/models'
  // @ts-ignore - wailsjs path
  import { GetAccountFoldersForMapping, GetAutoDetectedFolders, GetIdentities } from '../../../../wailsjs/go/app/App'

  // OAuth credentials to pass to parent
  export interface OAuthCredentials {
    provider: string
    accessToken: string
    refreshToken: string
    expiresIn: number
  }

  interface Props {
    /** Account to edit (null for new account) */
    editAccount?: account.Account | null
    /** Callback when form is submitted successfully */
    onSubmit?: (config: account.AccountConfig, oauthCredentials?: OAuthCredentials) => Promise<void>
    /** Callback when form is cancelled */
    onCancel?: () => void
    /** Callback for testing connection */
    onTestConnection?: (config: account.AccountConfig) => Promise<void>
  }

  let {
    editAccount = null,
    onSubmit,
    onCancel,
    onTestConnection,
  }: Props = $props()

  // Form state
  let step = $state<'provider' | 'details'>('provider')
  let selectedProvider = $state<EmailProvider | null>(null)
  let showAdvanced = $state(false)

  // OAuth state
  let authMethod = $state<'password' | 'oauth2'>('password')
  let oauthConfigured = $state<Record<OAuthProvider, boolean>>({
    google: false,
    microsoft: false,
  })
  let oauthInitialized = $state(false)

  // Form fields
  let name = $state('')
  let displayName = $state('')
  let color = $state('')
  let email = $state('')
  let username = $state('')
  let password = $state('')
  let imapHost = $state('')
  let imapPort = $state(993)
  let imapSecurity = $state<string>('tls')
  let smtpHost = $state('')
  let smtpPort = $state(587)
  let smtpSecurity = $state<string>('starttls')
  let syncPeriodDays = $state<string>('180')
  let syncInterval = $state<string>('30') // Default: 30 minutes
  let readReceiptRequestPolicy = $state<string>('never')

  // Read receipt request policy options
  const readReceiptRequestOptions = [
    { value: 'never', label: 'Never request' },
    { value: 'ask', label: 'Ask each time' },
    { value: 'always', label: 'Always request' },
  ]

  // Helper functions to get labels
  function getSecurityLabel(value: string): string {
    return securityOptions.find(opt => opt.value === value)?.label || value
  }

  function getSyncPeriodLabel(value: string): string {
    const numValue = Number(value)
    return syncPeriodOptions.find(opt => opt.value === numValue)?.label || `${value} days`
  }

  function getSyncIntervalLabel(value: string): string {
    const numValue = Number(value)
    return syncIntervalOptions.find(opt => opt.value === numValue)?.label || `${value} min`
  }

  function getReadReceiptLabel(value: string): string {
    return readReceiptRequestOptions.find(opt => opt.value === value)?.label || value
  }

  // UI state
  let testing = $state(false)
  let testResult = $state<{ success: boolean; message: string } | null>(null)
  let submitting = $state(false)
  let errors = $state<Record<string, string>>({})
  let initialized = $state(false)

  // Folder mapping state
  let showFolderMapping = $state(false)
  let loadingFolders = $state(false)
  let availableFolders = $state<any[]>([])
  let autoDetectedFolders = $state<Record<string, string>>({})

  // Folder mapping values
  let sentFolderPath = $state('')
  let draftsFolderPath = $state('')
  let trashFolderPath = $state('')
  let spamFolderPath = $state('')
  let archiveFolderPath = $state('')
  let allMailFolderPath = $state('')
  let starredFolderPath = $state('')

  // Folder mapping types configuration
  const folderMappingTypes = [
    { key: 'sent', label: 'Sent', getValue: () => sentFolderPath, setValue: (v: string) => sentFolderPath = v },
    { key: 'drafts', label: 'Drafts', getValue: () => draftsFolderPath, setValue: (v: string) => draftsFolderPath = v },
    { key: 'trash', label: 'Trash', getValue: () => trashFolderPath, setValue: (v: string) => trashFolderPath = v },
    { key: 'spam', label: 'Spam/Junk', getValue: () => spamFolderPath, setValue: (v: string) => spamFolderPath = v },
    { key: 'archive', label: 'Archive', getValue: () => archiveFolderPath, setValue: (v: string) => archiveFolderPath = v },
    { key: 'all', label: 'All Mail', getValue: () => allMailFolderPath, setValue: (v: string) => allMailFolderPath = v },
    { key: 'starred', label: 'Starred', getValue: () => starredFolderPath, setValue: (v: string) => starredFolderPath = v },
  ]

  // Load folders for mapping UI
  async function loadFoldersForMapping() {
    if (!editAccount || availableFolders.length > 0) return

    loadingFolders = true
    try {
      availableFolders = await GetAccountFoldersForMapping(editAccount.id)
      autoDetectedFolders = await GetAutoDetectedFolders(editAccount.id)

      // Pre-select: use saved value if exists, otherwise auto-detected
      // @ts-ignore - wailsjs binding will have these fields after regeneration
      sentFolderPath = editAccount.sentFolderPath || autoDetectedFolders.sent || ''
      // @ts-ignore
      draftsFolderPath = editAccount.draftsFolderPath || autoDetectedFolders.drafts || ''
      // @ts-ignore
      trashFolderPath = editAccount.trashFolderPath || autoDetectedFolders.trash || ''
      // @ts-ignore
      spamFolderPath = editAccount.spamFolderPath || autoDetectedFolders.spam || ''
      // @ts-ignore
      archiveFolderPath = editAccount.archiveFolderPath || autoDetectedFolders.archive || ''
      // @ts-ignore
      allMailFolderPath = editAccount.allMailFolderPath || autoDetectedFolders.all || ''
      // @ts-ignore
      starredFolderPath = editAccount.starredFolderPath || autoDetectedFolders.starred || ''
    } catch (err) {
      console.error('Failed to load folders for mapping:', err)
    } finally {
      loadingFolders = false
    }
  }

  // Initialize form when editing (only once)
  $effect(() => {
    if (editAccount && !initialized) {
      initialized = true
      step = 'details'
      name = editAccount.name
      email = editAccount.email
      username = editAccount.username
      imapHost = editAccount.imapHost
      imapPort = editAccount.imapPort
      imapSecurity = editAccount.imapSecurity
      smtpHost = editAccount.smtpHost
      smtpPort = editAccount.smtpPort
      smtpSecurity = editAccount.smtpSecurity
      syncPeriodDays = String(editAccount.syncPeriodDays)
      // @ts-ignore - syncInterval from backend
      syncInterval = String(editAccount.syncInterval ?? 30)
      readReceiptRequestPolicy = editAccount.readReceiptRequestPolicy || 'never'
      // @ts-ignore - authType from backend
      authMethod = editAccount.authType === 'oauth2' ? 'oauth2' : 'password'
      // @ts-ignore - color from backend
      color = editAccount.color || ''

      // Initialize folder mappings (will be populated when section is expanded)
      // @ts-ignore - wailsjs binding will have these fields after regeneration
      sentFolderPath = editAccount.sentFolderPath || ''
      // @ts-ignore
      draftsFolderPath = editAccount.draftsFolderPath || ''
      // @ts-ignore
      trashFolderPath = editAccount.trashFolderPath || ''
      // @ts-ignore
      spamFolderPath = editAccount.spamFolderPath || ''
      // @ts-ignore
      archiveFolderPath = editAccount.archiveFolderPath || ''
      // @ts-ignore
      allMailFolderPath = editAccount.allMailFolderPath || ''
      // @ts-ignore
      starredFolderPath = editAccount.starredFolderPath || ''

      // Try to detect provider
      selectedProvider = detectProvider(email) ?? getCustomProvider()
      showAdvanced = selectedProvider.id === 'custom'

      // Load display name from the default identity
      loadDisplayName(editAccount.id)
    }
  })

  // Load display name from the account's default identity
  async function loadDisplayName(accountId: string) {
    try {
      const identities = await GetIdentities(accountId)
      const defaultIdentity = identities?.find((id: any) => id.isDefault) || identities?.[0]
      if (defaultIdentity) {
        displayName = defaultIdentity.name || ''
      }
    } catch (err) {
      console.error('Failed to load display name:', err)
    }
  }

  // Initialize OAuth configuration check
  $effect(() => {
    if (!oauthInitialized) {
      oauthInitialized = true
      checkOAuthConfiguration()
      // Initialize OAuth event listeners
      oauthStore.initEvents()
    }
  })

  // Update authMethod when OAuth configuration is loaded (fixes race condition)
  // This ensures that if a user selects a provider before OAuth config is loaded,
  // the auth method will be corrected once we know OAuth is available
  $effect(() => {
    // Only update if we're on a new account (not editing) and using password auth
    if (!editAccount && authMethod === 'password' && selectedProvider) {
      // Check if OAuth is now available for this provider
      if (canUseOAuth(selectedProvider)) {
        authMethod = 'oauth2'
      }
    }
  })

  // Check which OAuth providers are configured
  async function checkOAuthConfiguration() {
    try {
      const [googleConfigured, microsoftConfigured] = await Promise.all([
        oauthStore.isProviderConfigured('google'),
        oauthStore.isProviderConfigured('microsoft'),
      ])
      oauthConfigured = {
        google: googleConfigured,
        microsoft: microsoftConfigured,
      }
    } catch (err) {
      console.error('Failed to check OAuth configuration:', err)
    }
  }

  // Check if the selected provider supports OAuth and it's configured
  function canUseOAuth(provider: EmailProvider | null): boolean {
    if (!provider) return false
    if (!isOAuthProvider(provider)) return false
    const oauthType = getOAuthProviderType(provider)
    if (!oauthType) return false
    return oauthConfigured[oauthType] ?? false
  }

  // Start OAuth flow for the selected provider
  async function startOAuthFlow() {
    if (!selectedProvider) return
    const oauthType = getOAuthProviderType(selectedProvider)
    if (!oauthType) return

    try {
      await oauthStore.startFlow(oauthType)
    } catch (err) {
      console.error('Failed to start OAuth flow:', err)
    }
  }

  // Cancel OAuth flow
  function cancelOAuthFlow() {
    oauthStore.cancelFlow()
  }

  // Get OAuth button text based on provider
  function getOAuthButtonText(provider: EmailProvider | null): string {
    if (!provider) return 'Sign in'
    const oauthType = getOAuthProviderType(provider)
    if (oauthType === 'google') return 'Sign in with Google'
    if (oauthType === 'microsoft') return 'Sign in with Microsoft'
    return 'Sign in'
  }

  // Get OAuth button icon based on provider
  function getOAuthButtonIcon(provider: EmailProvider | null): string {
    if (!provider) return 'mdi:login'
    const oauthType = getOAuthProviderType(provider)
    if (oauthType === 'google') return 'logos:google-icon'
    if (oauthType === 'microsoft') return 'logos:microsoft-icon'
    return 'mdi:login'
  }

  // Auto-fill settings when provider is selected
  function selectProvider(provider: EmailProvider) {
    selectedProvider = provider
    imapHost = provider.imap.host
    imapPort = provider.imap.port
    imapSecurity = provider.imap.security
    smtpHost = provider.smtp.host
    smtpPort = provider.smtp.port
    smtpSecurity = provider.smtp.security

    // Set auth method based on provider and configuration
    if (canUseOAuth(provider)) {
      authMethod = 'oauth2'
    } else {
      authMethod = 'password'
    }

    // Show advanced for custom provider
    showAdvanced = provider.id === 'custom'
    step = 'details'
  }

  // Auto-detect provider when email changes
  function handleEmailChange(e: Event) {
    const target = e.target as HTMLInputElement
    email = target.value

    // Auto-fill username
    if (!username || username === email.split('@')[0] + '@' + (selectedProvider?.domains[0] ?? '')) {
      username = email
    }

    // Try to detect provider
    const detected = detectProvider(email)
    if (detected && detected.id !== selectedProvider?.id) {
      selectProvider(detected)
    }

    // Auto-fill name from email if empty
    if (!name) {
      const localPart = email.split('@')[0]
      if (localPart) {
        name = localPart.charAt(0).toUpperCase() + localPart.slice(1)
      }
    }
  }

  // Build config from form fields
  function buildConfig(): account.AccountConfig {
    return new account.AccountConfig({
      name,
      displayName,
      color,
      email,
      username: username || email,
      password: authMethod === 'oauth2' ? '' : password,
      imapHost,
      imapPort,
      imapSecurity,
      smtpHost,
      smtpPort,
      smtpSecurity,
      authType: authMethod,
      syncPeriodDays: Number(syncPeriodDays),
      syncInterval: Number(syncInterval),
      readReceiptRequestPolicy,
      // Folder mappings
      sentFolderPath,
      draftsFolderPath,
      trashFolderPath,
      spamFolderPath,
      archiveFolderPath,
      allMailFolderPath,
      starredFolderPath,
    })
  }

  // Validate form
  function validate(): boolean {
    errors = {}

    if (!name.trim()) errors.name = 'Account name is required'
    if (!displayName.trim()) errors.displayName = 'Display name is required'
    if (!email.trim()) errors.email = 'Email is required'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.email = 'Invalid email format'
    
    // Password is only required for password auth on new accounts
    if (authMethod === 'password' && !password && !editAccount) {
      errors.password = 'Password is required'
    }
    
    // For OAuth, check that the flow completed successfully
    if (authMethod === 'oauth2' && !editAccount && !oauthStore.isFlowSuccess) {
      errors.oauth = 'Please complete the sign-in process'
    }
    
    if (!imapHost.trim()) errors.imapHost = 'IMAP host is required'
    if (!smtpHost.trim()) errors.smtpHost = 'SMTP host is required'
    if (imapPort < 1 || imapPort > 65535) errors.imapPort = 'Invalid port'
    if (smtpPort < 1 || smtpPort > 65535) errors.smtpPort = 'Invalid port'

    return Object.keys(errors).length === 0
  }

  // Test connection
  async function handleTestConnection() {
    if (!validate()) return

    testing = true
    testResult = null

    try {
      await onTestConnection?.(buildConfig())
      testResult = { success: true, message: 'Connection successful!' }
    } catch (err) {
      testResult = {
        success: false,
        message: err instanceof Error ? err.message : String(err),
      }
    } finally {
      testing = false
    }
  }

  // Submit form
  async function handleSubmit(e: Event) {
    e.preventDefault()
    if (!validate()) return

    submitting = true
    testResult = null

    try {
      // Build OAuth credentials if using OAuth
      let oauthCredentials: OAuthCredentials | undefined
      if (authMethod === 'oauth2' && oauthStore.isFlowSuccess && oauthStore.flowResult) {
        // Note: The actual tokens are stored in the backend during the OAuth flow
        // We just pass the metadata so the parent can complete the account setup
        oauthCredentials = {
          provider: oauthStore.flowResult.provider,
          accessToken: '', // Tokens are handled by backend
          refreshToken: '', // Tokens are handled by backend
          expiresIn: oauthStore.flowResult.expiresIn,
        }
      }
      
      await onSubmit?.(buildConfig(), oauthCredentials)
      
      // Reset OAuth state on success
      if (authMethod === 'oauth2') {
        oauthStore.reset()
      }
    } catch (err) {
      testResult = {
        success: false,
        message: err instanceof Error ? err.message : String(err),
      }
    } finally {
      submitting = false
    }
  }

  // Go back to provider selection
  function goBackToProviders() {
    step = 'provider'
    testResult = null
  }
</script>

<form onsubmit={handleSubmit} class="space-y-6">
  {#if step === 'provider' && !editAccount}
    <!-- Step 1: Provider Selection -->
    <div class="space-y-4">
      <div class="text-center">
        <h3 class="text-lg font-medium">Choose your email provider</h3>
        <p class="text-sm text-muted-foreground mt-1">
          Select your provider for automatic configuration
        </p>
      </div>

      <div class="grid grid-cols-3 gap-3">
        {#each providers as provider (provider.id)}
          <button
            type="button"
            class="flex flex-col items-center gap-2 p-4 rounded-lg border border-input bg-background hover:bg-accent hover:text-accent-foreground transition-colors"
            onclick={() => selectProvider(provider)}
          >
            <Icon icon={provider.icon} class="w-8 h-8" />
            <span class="text-sm font-medium text-center">{provider.name}</span>
          </button>
        {/each}
      </div>
    </div>
  {:else}
    <!-- Step 2: Account Details -->
    <div class="space-y-4">
      {#if !editAccount}
        <button
          type="button"
          class="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors"
          onclick={goBackToProviders}
        >
          <Icon icon="mdi:arrow-left" class="w-4 h-4" />
          Change provider
        </button>
      {/if}

      {#if selectedProvider?.notes}
        <div class="flex items-start gap-2 p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
          <Icon icon="mdi:information-outline" class="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
          <p class="text-sm text-amber-600 dark:text-amber-400">
            {selectedProvider.notes}
          </p>
        </div>
      {/if}

      <!-- Basic Fields -->
      <div class="grid gap-4">
        <div class="space-y-2">
          <Label for="name">Account Name</Label>
          <div class="flex items-center gap-3">
            <ColorPicker value={color} onchange={(c) => color = c} />
            <Input
              id="name"
              type="text"
              placeholder="e.g., Personal, Work"
              bind:value={name}
              class={errors.name ? 'border-destructive' : ''}
            />
          </div>
          <p class="text-xs text-muted-foreground">
            Color is used to identify this account in unified inbox
          </p>
          {#if errors.name}
            <p class="text-sm text-destructive">{errors.name}</p>
          {/if}
        </div>

        <div class="space-y-2">
          <Label for="displayName">Display Name</Label>
          <Input
            id="displayName"
            type="text"
            placeholder="e.g., John Smith"
            bind:value={displayName}
            class={errors.displayName ? 'border-destructive' : ''}
          />
          <p class="text-xs text-muted-foreground">
            Name shown to email recipients
          </p>
          {#if errors.displayName}
            <p class="text-sm text-destructive">{errors.displayName}</p>
          {/if}
        </div>

        <div class="space-y-2">
          <Label for="email">Email Address</Label>
          <Input
            id="email"
            type="email"
            placeholder="you@example.com"
            value={email}
            oninput={handleEmailChange}
            class={errors.email ? 'border-destructive' : ''}
          />
          {#if errors.email}
            <p class="text-sm text-destructive">{errors.email}</p>
          {/if}
        </div>

        <div class="space-y-2">
          <Label for="username">Username</Label>
          <Input
            id="username"
            type="text"
            placeholder="Usually your email address"
            bind:value={username}
          />
          <p class="text-xs text-muted-foreground">
            Leave empty to use email address
          </p>
        </div>

        <!-- Authentication Section -->
        <div class="space-y-3">
          {#if canUseOAuth(selectedProvider) && !editAccount}
            <!-- OAuth Provider - Show Sign In Button -->
            <div class="space-y-3">
              <Label>Authentication</Label>
              
              {#if allowsPasswordFallback(selectedProvider!)}
                <!-- Provider allows both OAuth and password -->
                <div class="flex gap-2">
                  <Button
                    type="button"
                    variant={authMethod === 'oauth2' ? 'default' : 'outline'}
                    size="sm"
                    onclick={() => authMethod = 'oauth2'}
                    class="flex-1"
                  >
                    <Icon icon={getOAuthButtonIcon(selectedProvider)} class="w-4 h-4 mr-2" />
                    OAuth
                  </Button>
                  <Button
                    type="button"
                    variant={authMethod === 'password' ? 'default' : 'outline'}
                    size="sm"
                    onclick={() => authMethod = 'password'}
                    class="flex-1"
                  >
                    <Icon icon="mdi:key" class="w-4 h-4 mr-2" />
                    App Password
                  </Button>
                </div>
              {/if}

              {#if authMethod === 'oauth2'}
                <!-- OAuth Flow UI -->
                <div class="rounded-lg border border-border p-4 space-y-3">
                  {#if oauthStore.flowState === 'idle' || oauthStore.flowState === 'cancelled'}
                    <!-- Initial state - show sign in button -->
                    <Button
                      type="button"
                      variant="outline"
                      class="w-full h-12"
                      onclick={startOAuthFlow}
                    >
                      <Icon icon={getOAuthButtonIcon(selectedProvider)} class="w-5 h-5 mr-3" />
                      {getOAuthButtonText(selectedProvider)}
                    </Button>
                    <p class="text-xs text-muted-foreground text-center">
                      You'll be redirected to sign in securely
                    </p>
                  {:else if oauthStore.flowState === 'pending'}
                    <!-- Waiting for OAuth callback -->
                    <div class="flex flex-col items-center gap-3 py-2">
                      <Icon icon="mdi:loading" class="w-8 h-8 animate-spin text-primary" />
                      <div class="text-center">
                        <p class="text-sm font-medium">Waiting for authentication...</p>
                        <p class="text-xs text-muted-foreground mt-1">
                          Complete sign-in in your browser
                        </p>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onclick={cancelOAuthFlow}
                      >
                        Cancel
                      </Button>
                    </div>
                  {:else if oauthStore.flowState === 'success'}
                    <!-- OAuth completed successfully -->
                    <div class="flex items-center gap-3 py-2">
                      <div class="flex-shrink-0 w-10 h-10 rounded-full bg-green-500/10 flex items-center justify-center">
                        <Icon icon="mdi:check" class="w-5 h-5 text-green-500" />
                      </div>
                      <div class="flex-1 min-w-0">
                        <p class="text-sm font-medium text-green-600 dark:text-green-400">
                          Connected successfully
                        </p>
                        <p class="text-xs text-muted-foreground truncate">
                          {oauthStore.flowResult?.email}
                        </p>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onclick={() => {
                          oauthStore.reset()
                        }}
                      >
                        <Icon icon="mdi:refresh" class="w-4 h-4" />
                      </Button>
                    </div>
                  {:else if oauthStore.flowState === 'error'}
                    <!-- OAuth failed -->
                    <div class="space-y-3">
                      <div class="flex items-start gap-3">
                        <div class="flex-shrink-0 w-10 h-10 rounded-full bg-destructive/10 flex items-center justify-center">
                          <Icon icon="mdi:alert" class="w-5 h-5 text-destructive" />
                        </div>
                        <div class="flex-1">
                          <p class="text-sm font-medium text-destructive">
                            Authentication failed
                          </p>
                          <p class="text-xs text-muted-foreground mt-1">
                            {oauthStore.flowError}
                          </p>
                        </div>
                      </div>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        class="w-full"
                        onclick={startOAuthFlow}
                      >
                        Try again
                      </Button>
                    </div>
                  {/if}
                </div>
                {#if errors.oauth}
                  <p class="text-sm text-destructive">{errors.oauth}</p>
                {/if}
              {:else}
                <!-- Password field for app password -->
                <div class="space-y-2">
                  <Label for="password">App Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Enter app password"
                    bind:value={password}
                    class={errors.password ? 'border-destructive' : ''}
                  />
                  {#if errors.password}
                    <p class="text-sm text-destructive">{errors.password}</p>
                  {/if}
                </div>
              {/if}
            </div>
          {:else if editAccount && editAccount.authType === 'oauth2'}
            <!-- Editing an OAuth account -->
            <div class="space-y-2">
              <Label>Authentication</Label>
              <div class="rounded-lg border border-border p-4">
                <div class="flex items-center gap-3">
                  <div class="flex-shrink-0 w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                    <Icon icon={getOAuthButtonIcon(selectedProvider)} class="w-5 h-5" />
                  </div>
                  <div class="flex-1">
                    <p class="text-sm font-medium">Connected via OAuth</p>
                    <p class="text-xs text-muted-foreground">
                      Sign in again to refresh credentials if needed
                    </p>
                  </div>
                </div>
              </div>
            </div>
          {:else}
            <!-- Standard password field -->
            <div class="space-y-2">
              <Label for="password">
                {selectedProvider?.notes?.includes('App Password') ? 'App Password' : 'Password'}
              </Label>
              <Input
                id="password"
                type="password"
                placeholder={editAccount ? 'Leave empty to keep current' : 'Enter password'}
                bind:value={password}
                class={errors.password ? 'border-destructive' : ''}
              />
              {#if errors.password}
                <p class="text-sm text-destructive">{errors.password}</p>
              {/if}
            </div>
          {/if}
        </div>
      </div>

      <!-- Advanced Settings Toggle -->
      <button
        type="button"
        class="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
        onclick={() => (showAdvanced = !showAdvanced)}
      >
        <Icon
          icon={showAdvanced ? 'mdi:chevron-down' : 'mdi:chevron-right'}
          class="w-4 h-4"
        />
        Advanced Settings
      </button>

      {#if showAdvanced}
        <div class="space-y-4 pt-2 border-t border-border">
          <!-- IMAP Settings -->
          <div class="space-y-3">
            <h4 class="text-sm font-medium">Incoming Mail (IMAP)</h4>
            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-2">
                <Label for="imapHost">Server</Label>
                <Input
                  id="imapHost"
                  type="text"
                  placeholder="imap.example.com"
                  bind:value={imapHost}
                  class={errors.imapHost ? 'border-destructive' : ''}
                />
                {#if errors.imapHost}
                  <p class="text-sm text-destructive">{errors.imapHost}</p>
                {/if}
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div class="space-y-2">
                  <Label for="imapPort">Port</Label>
                  <Input
                    id="imapPort"
                    type="number"
                    bind:value={imapPort}
                    class={errors.imapPort ? 'border-destructive' : ''}
                  />
                </div>
                <div class="space-y-2">
                  <Label>Security</Label>
                  <Select.Root bind:value={imapSecurity}>
                    <Select.Trigger class="h-10">
                      <Select.Value placeholder="Select">
                        {getSecurityLabel(imapSecurity)}
                      </Select.Value>
                    </Select.Trigger>
                    <Select.Content>
                      {#each securityOptions as opt (opt.value)}
                        <Select.Item value={opt.value} label={opt.label} />
                      {/each}
                    </Select.Content>
                  </Select.Root>
                </div>
              </div>
            </div>
          </div>

          <!-- SMTP Settings -->
          <div class="space-y-3">
            <h4 class="text-sm font-medium">Outgoing Mail (SMTP)</h4>
            <div class="grid grid-cols-2 gap-3">
              <div class="space-y-2">
                <Label for="smtpHost">Server</Label>
                <Input
                  id="smtpHost"
                  type="text"
                  placeholder="smtp.example.com"
                  bind:value={smtpHost}
                  class={errors.smtpHost ? 'border-destructive' : ''}
                />
                {#if errors.smtpHost}
                  <p class="text-sm text-destructive">{errors.smtpHost}</p>
                {/if}
              </div>
              <div class="grid grid-cols-2 gap-2">
                <div class="space-y-2">
                  <Label for="smtpPort">Port</Label>
                  <Input
                    id="smtpPort"
                    type="number"
                    bind:value={smtpPort}
                    class={errors.smtpPort ? 'border-destructive' : ''}
                  />
                </div>
                <div class="space-y-2">
                  <Label>Security</Label>
                  <Select.Root bind:value={smtpSecurity}>
                    <Select.Trigger class="h-10">
                      <Select.Value placeholder="Select">
                        {getSecurityLabel(smtpSecurity)}
                      </Select.Value>
                    </Select.Trigger>
                    <Select.Content>
                      {#each securityOptions as opt (opt.value)}
                        <Select.Item value={opt.value} label={opt.label} />
                      {/each}
                    </Select.Content>
                  </Select.Root>
                </div>
              </div>
            </div>
          </div>

          <!-- Sync Settings -->
          <div class="space-y-2">
            <Label>Sync Period</Label>
            <Select.Root bind:value={syncPeriodDays}>
              <Select.Trigger>
                <Select.Value placeholder="Select">
                  {getSyncPeriodLabel(syncPeriodDays)}
                </Select.Value>
              </Select.Trigger>
              <Select.Content>
                {#each syncPeriodOptions as opt (opt.value)}
                  <Select.Item value={String(opt.value)} label={opt.label} />
                {/each}
              </Select.Content>
            </Select.Root>
            <p class="text-xs text-muted-foreground">
              How far back to sync messages
            </p>
          </div>

          <!-- Check Interval Settings -->
          <div class="space-y-2">
            <Label>Check for New Mail</Label>
            <Select.Root bind:value={syncInterval}>
              <Select.Trigger>
                <Select.Value placeholder="Select">
                  {getSyncIntervalLabel(syncInterval)}
                </Select.Value>
              </Select.Trigger>
              <Select.Content>
                {#each syncIntervalOptions as opt (opt.value)}
                  <Select.Item value={String(opt.value)} label={opt.label} />
                {/each}
              </Select.Content>
            </Select.Root>
            <p class="text-xs text-muted-foreground">
              How often to check for new messages (IDLE push is also used when available)
            </p>
          </div>

          <!-- Read Receipt Settings -->
          <div class="space-y-2">
            <Label>Request Read Receipts</Label>
            <Select.Root bind:value={readReceiptRequestPolicy}>
              <Select.Trigger>
                <Select.Value placeholder="Select">
                  {getReadReceiptLabel(readReceiptRequestPolicy)}
                </Select.Value>
              </Select.Trigger>
              <Select.Content>
                {#each readReceiptRequestOptions as opt (opt.value)}
                  <Select.Item value={opt.value} label={opt.label} />
                {/each}
              </Select.Content>
            </Select.Root>
            <p class="text-xs text-muted-foreground">
              When to request read receipts for outgoing messages
            </p>
          </div>

          <!-- Folder Mapping -->
          <div class="space-y-2">
            <button
              type="button"
              class="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
              onclick={() => {
                showFolderMapping = !showFolderMapping
                if (showFolderMapping) loadFoldersForMapping()
              }}
              disabled={!editAccount}
            >
              <Icon
                icon={showFolderMapping ? 'mdi:chevron-down' : 'mdi:chevron-right'}
                class="w-4 h-4"
              />
              Folder Mapping
              {#if !editAccount}
                <span class="text-xs text-muted-foreground">(save account first)</span>
              {/if}
            </button>

            {#if showFolderMapping}
              <div class="space-y-3 pl-6 pt-2 border-l border-border ml-2">
                <p class="text-xs text-muted-foreground">
                  Map folder types to specific IMAP folders on your server.
                </p>

                {#if loadingFolders}
                  <div class="flex items-center gap-2 text-sm text-muted-foreground">
                    <Icon icon="mdi:loading" class="w-4 h-4 animate-spin" />
                    Loading folders...
                  </div>
                {:else if availableFolders.length === 0}
                  <p class="text-sm text-muted-foreground">No folders available.</p>
                {:else}
                  <div class="grid gap-3">
                    <!-- Sent -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Sent:</Label>
                      <Select.Root bind:value={sentFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {sentFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.sent === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- Drafts -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Drafts:</Label>
                      <Select.Root bind:value={draftsFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {draftsFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.drafts === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- Trash -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Trash:</Label>
                      <Select.Root bind:value={trashFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {trashFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.trash === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- Spam/Junk -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Spam/Junk:</Label>
                      <Select.Root bind:value={spamFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {spamFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.spam === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- Archive -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Archive:</Label>
                      <Select.Root bind:value={archiveFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {archiveFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.archive === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- All Mail -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">All Mail:</Label>
                      <Select.Root bind:value={allMailFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {allMailFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.all === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>

                    <!-- Starred -->
                    <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                      <Label class="text-sm">Starred:</Label>
                      <Select.Root bind:value={starredFolderPath}>
                        <Select.Trigger class="h-9">
                          <Select.Value placeholder="None">
                            {starredFolderPath || 'None'}
                          </Select.Value>
                        </Select.Trigger>
                        <Select.Content>
                          <Select.Item value="" label="None" />
                          {#each availableFolders as f (f.path)}
                            <Select.Item value={f.path} label={f.path + (autoDetectedFolders.starred === f.path ? ' (detected)' : '')} />
                          {/each}
                        </Select.Content>
                      </Select.Root>
                    </div>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/if}

      <!-- Test Result -->
      {#if testResult}
        <div
          class="flex items-start gap-2 p-3 rounded-lg {testResult.success
            ? 'bg-green-500/10 border border-green-500/20'
            : 'bg-destructive/10 border border-destructive/20'}"
        >
          <Icon
            icon={testResult.success ? 'mdi:check-circle' : 'mdi:alert-circle'}
            class="w-5 h-5 flex-shrink-0 mt-0.5 {testResult.success
              ? 'text-green-500'
              : 'text-destructive'}"
          />
          <p
            class="text-sm {testResult.success
              ? 'text-green-600 dark:text-green-400'
              : 'text-destructive'}"
          >
            {testResult.message}
          </p>
        </div>
      {/if}

      <!-- Actions -->
      <div class="flex items-center justify-between pt-4 border-t border-border">
        <Button
          type="button"
          variant="outline"
          onclick={handleTestConnection}
          disabled={testing || submitting}
        >
          {#if testing}
            <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
          {:else}
            <Icon icon="mdi:connection" class="w-4 h-4 mr-2" />
          {/if}
          Test Connection
        </Button>

        <div class="flex gap-2">
          <Button type="button" variant="ghost" onclick={onCancel} disabled={submitting}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting || testing}>
            {#if submitting}
              <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
            {/if}
            {editAccount ? 'Save Changes' : 'Add Account'}
          </Button>
        </div>
      </div>
    </div>
  {/if}
</form>

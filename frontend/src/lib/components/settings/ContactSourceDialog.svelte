<script lang="ts">
  import Icon from '@iconify/svelte'
  import * as Dialog from '$lib/components/ui/dialog'
  import * as Select from '$lib/components/ui/select'
  import * as Tabs from '$lib/components/ui/tabs'
  import { Label } from '$lib/components/ui/label'
  import { Input } from '$lib/components/ui/input'
  import { Button } from '$lib/components/ui/button'
  import { addToast } from '$lib/stores/toast'
  import { contactSourcesStore, type LinkedAccountInfo } from '$lib/stores/contactSources.svelte'
  // @ts-ignore - wailsjs runtime
  import { EventsOn, EventsOff } from '../../../../wailsjs/runtime/runtime'
  // @ts-ignore - wailsjs path
  import {
    DiscoverCardDAVAddressbooks,
    AddContactSource,
    UpdateContactSource,
    GetSourceAddressbooks,
  } from '../../../../wailsjs/go/app/App.js'
  // @ts-ignore - wailsjs path
  import type { carddav } from '../../../../wailsjs/go/models'

  interface Props {
    open?: boolean
    editSource?: carddav.Source | null
    onClose?: () => void
  }

  let {
    open = $bindable(false),
    editSource = null,
    onClose,
  }: Props = $props()

  // Source type selection
  type SourceType = 'carddav' | 'google' | 'microsoft'
  let sourceType = $state<SourceType>('carddav')

  // CardDAV form state
  let name = $state('')
  let url = $state('')
  let username = $state('')
  let password = $state('')
  let syncInterval = $state(60)

  // Discovery state
  let discovering = $state(false)
  let discoveredAddressbooks = $state<carddav.AddressbookInfo[]>([])
  let selectedAddressbooks = $state<Set<string>>(new Set())
  let discoveryError = $state<string | null>(null)
  let hasDiscovered = $state(false)

  // OAuth state
  let linkedAccounts = $state<LinkedAccountInfo[]>([])
  let selectedAccountId = $state<string>('')
  let loadingAccounts = $state(false)
  let oauthInProgress = $state(false)
  let oauthEmail = $state<string>('')

  // Save state
  let saving = $state(false)

  // Sync interval options (value is string for Select component)
  const syncIntervalOptions = [
    { value: '0', label: 'Manual only' },
    { value: '15', label: 'Every 15 minutes' },
    { value: '30', label: 'Every 30 minutes' },
    { value: '60', label: 'Every hour' },
    { value: '120', label: 'Every 2 hours' },
    { value: '360', label: 'Every 6 hours' },
    { value: '1440', label: 'Daily' },
  ]

  // Convert between number state and string Select value
  let syncIntervalStr = $derived(String(syncInterval))

  function getSyncIntervalLabel(value: number): string {
    return syncIntervalOptions.find(opt => opt.value === String(value))?.label || `${value} minutes`
  }

  function handleSyncIntervalChange(value: string) {
    syncInterval = parseInt(value, 10)
  }

  // Computed: is editing an OAuth source
  let isEditingOAuthSource = $derived(
    editSource && (editSource.type === 'google' || editSource.type === 'microsoft')
  )

  // Load linked accounts when switching to OAuth tabs
  async function loadLinkedAccounts() {
    loadingAccounts = true
    try {
      linkedAccounts = await contactSourcesStore.getLinkedAccounts()
    } finally {
      loadingAccounts = false
    }
  }

  // Get available accounts for the selected provider (not already linked)
  let availableAccounts = $derived(
    linkedAccounts.filter(acc => acc.provider === sourceType && !acc.isLinked)
  )

  // Load existing source data when editing
  $effect(() => {
    if (open && editSource) {
      name = editSource.name || ''
      url = editSource.url || ''
      username = editSource.username || ''
      password = '' // Don't load password
      syncInterval = editSource.sync_interval || 60
      hasDiscovered = false
      discoveredAddressbooks = []
      selectedAddressbooks = new Set()
      discoveryError = null
      sourceType = editSource.type as SourceType || 'carddav'

      // Load existing addressbooks for CardDAV
      if (editSource.type === 'carddav') {
        loadExistingAddressbooks()
      }
    } else if (open && !editSource) {
      // Reset for new source
      name = ''
      url = ''
      username = ''
      password = ''
      syncInterval = 60
      hasDiscovered = false
      discoveredAddressbooks = []
      selectedAddressbooks = new Set()
      discoveryError = null
      sourceType = 'carddav'
      selectedAccountId = ''
      oauthInProgress = false
      oauthEmail = ''

      // Load linked accounts for OAuth sources
      loadLinkedAccounts()
    }
  })

  // Set up OAuth event listeners
  $effect(() => {
    if (open) {
      // Listen for OAuth success
      EventsOn('contact-source-oauth:success', (data: { provider: string; email: string }) => {
        oauthInProgress = false
        oauthEmail = data.email
        if (!name) {
          name = `${data.provider === 'google' ? 'Google' : 'Microsoft'} Contacts (${data.email})`
        }
      })

      // Listen for OAuth error
      EventsOn('contact-source-oauth:error', (data: { error: string }) => {
        oauthInProgress = false
        addToast({ type: 'error', message: `OAuth failed: ${data.error}` })
      })

      // Listen for OAuth cancelled
      EventsOn('contact-source-oauth:cancelled', () => {
        oauthInProgress = false
      })

      return () => {
        EventsOff('contact-source-oauth:success')
        EventsOff('contact-source-oauth:error')
        EventsOff('contact-source-oauth:cancelled')
      }
    }
  })

  async function loadExistingAddressbooks() {
    if (!editSource) return
    try {
      const addressbooks = await GetSourceAddressbooks(editSource.id)
      if (addressbooks) {
        // Convert to AddressbookInfo format
        discoveredAddressbooks = addressbooks.map((ab: carddav.Addressbook) => ({
          path: ab.path,
          name: ab.name,
          description: '',
        }))
        // Select all enabled ones
        selectedAddressbooks = new Set(
          addressbooks
            .filter((ab: carddav.Addressbook) => ab.enabled)
            .map((ab: carddav.Addressbook) => ab.path)
        )
        hasDiscovered = true
      }
    } catch (err) {
      console.error('Failed to load addressbooks:', err)
    }
  }

  async function handleDiscover() {
    if (!url || !username || !password) {
      discoveryError = 'Please fill in URL, username, and password'
      return
    }

    discovering = true
    discoveryError = null
    discoveredAddressbooks = []
    selectedAddressbooks = new Set()

    try {
      const addressbooks = await DiscoverCardDAVAddressbooks(url, username, password)
      if (addressbooks && addressbooks.length > 0) {
        discoveredAddressbooks = addressbooks
        // Select all by default
        selectedAddressbooks = new Set(addressbooks.map((ab: carddav.AddressbookInfo) => ab.path))
        hasDiscovered = true
      } else {
        discoveryError = 'No addressbooks found at this URL'
      }
    } catch (err) {
      console.error('Discovery failed:', err)
      discoveryError = `Discovery failed: ${err}`
    } finally {
      discovering = false
    }
  }

  function toggleAddressbook(path: string) {
    const newSet = new Set(selectedAddressbooks)
    if (newSet.has(path)) {
      newSet.delete(path)
    } else {
      newSet.add(path)
    }
    selectedAddressbooks = newSet
  }

  async function handleStartOAuth() {
    oauthInProgress = true
    try {
      await contactSourcesStore.startOAuthFlow(sourceType)
    } catch (err) {
      oauthInProgress = false
      addToast({ type: 'error', message: `Failed to start OAuth: ${err}` })
    }
  }

  async function handleSave() {
    saving = true

    try {
      if (sourceType === 'carddav') {
        // CardDAV source
        if (!name || !url || !username) {
          addToast({ type: 'error', message: 'Please fill in all required fields' })
          return
        }

        if (!editSource && !password) {
          addToast({ type: 'error', message: 'Password is required for new sources' })
          return
        }

        if (selectedAddressbooks.size === 0) {
          addToast({ type: 'error', message: 'Please select at least one addressbook' })
          return
        }

        const config = {
          name,
          type: 'carddav' as const,
          url,
          username,
          password,
          enabled: true,
          sync_interval: syncInterval,
          enabled_addressbooks: Array.from(selectedAddressbooks),
        }

        if (editSource) {
          await UpdateContactSource(editSource.id, config)
          addToast({ type: 'success', message: 'Contact source updated' })
        } else {
          await AddContactSource(config)
          addToast({ type: 'success', message: 'Contact source added' })
        }
      } else {
        // Google or Microsoft source
        if (editSource && isEditingOAuthSource) {
          // Editing existing OAuth source - just update name and sync interval
          const config = {
            name,
            type: editSource.type as 'google' | 'microsoft',
            url: '',
            username: '',
            password: '',
            enabled: true,
            sync_interval: syncInterval,
            enabled_addressbooks: [],
          }
          await UpdateContactSource(editSource.id, config)
          addToast({ type: 'success', message: 'Contact source updated' })
        } else if (selectedAccountId) {
          // Link to existing email account
          await contactSourcesStore.linkAccount(selectedAccountId, name, syncInterval)
          addToast({ type: 'success', message: 'Contact source linked to account' })
        } else if (oauthEmail) {
          // Complete standalone OAuth flow
          await contactSourcesStore.completeOAuthSetup(name, syncInterval)
          addToast({ type: 'success', message: 'Contact source created' })
        } else {
          addToast({ type: 'error', message: 'Please link an account or sign in' })
          return
        }
      }

      open = false
      onClose?.()
    } catch (err) {
      console.error('Failed to save:', err)
      addToast({ type: 'error', message: `Failed to save: ${err}` })
    } finally {
      saving = false
    }
  }

  function handleCancel() {
    if (oauthInProgress) {
      contactSourcesStore.cancelOAuthFlow()
    }
    open = false
    onClose?.()
  }

  function handleOpenChange(isOpen: boolean) {
    open = isOpen
    if (!isOpen) {
      if (oauthInProgress) {
        contactSourcesStore.cancelOAuthFlow()
      }
      onClose?.()
    }
  }

  function handleTabChange(value: string) {
    sourceType = value as SourceType
    // Reset OAuth state when switching tabs
    selectedAccountId = ''
    oauthEmail = ''
    oauthInProgress = false
  }

  // Check if save is enabled
  let canSave = $derived(() => {
    if (sourceType === 'carddav') {
      return hasDiscovered && selectedAddressbooks.size > 0 && name
    }
    if (editSource && isEditingOAuthSource) {
      return !!name
    }
    return (selectedAccountId || oauthEmail) && name
  })
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
  <Dialog.Content class="max-w-lg">
    <Dialog.Header>
      <Dialog.Title>{editSource ? 'Edit Contact Source' : 'Add Contact Source'}</Dialog.Title>
      <Dialog.Description>
        Sync contacts for email autocomplete
      </Dialog.Description>
    </Dialog.Header>

    {#if !editSource}
      <!-- Source type tabs (only for new sources) -->
      <Tabs.Root value={sourceType} onValueChange={handleTabChange} class="mt-4">
        <Tabs.List class="grid w-full grid-cols-3">
          <Tabs.Trigger value="carddav" class="flex items-center gap-1.5">
            <Icon icon="mdi:card-account-details" class="w-4 h-4" />
            CardDAV
          </Tabs.Trigger>
          <Tabs.Trigger value="google" class="flex items-center gap-1.5">
            <Icon icon="mdi:google" class="w-4 h-4" />
            Google
          </Tabs.Trigger>
          <Tabs.Trigger value="microsoft" class="flex items-center gap-1.5">
            <Icon icon="mdi:microsoft" class="w-4 h-4" />
            Microsoft
          </Tabs.Trigger>
        </Tabs.List>
      </Tabs.Root>
    {/if}

    <div class="space-y-4 py-4">
      {#if sourceType === 'carddav'}
        <!-- CardDAV Form -->
        <div class="space-y-2">
          <Label for="name">Name</Label>
          <Input
            id="name"
            bind:value={name}
            placeholder="e.g., Nextcloud, Work, Personal"
          />
        </div>

        <div class="space-y-2">
          <Label for="url">Server URL</Label>
          <Input
            id="url"
            bind:value={url}
            placeholder="https://cloud.example.com"
            disabled={!!editSource}
          />
          <p class="text-xs text-muted-foreground">
            Enter the server URL or full CardDAV addressbook URL
          </p>
        </div>

        <div class="space-y-2">
          <Label for="username">Username</Label>
          <Input
            id="username"
            bind:value={username}
            placeholder="your@email.com"
            disabled={!!editSource}
          />
        </div>

        <div class="space-y-2">
          <Label for="password">Password {editSource ? '(leave blank to keep current)' : ''}</Label>
          <Input
            id="password"
            type="password"
            bind:value={password}
            placeholder={editSource ? '********' : 'Password'}
          />
        </div>

        <Button
          variant="outline"
          class="w-full"
          onclick={handleDiscover}
          disabled={discovering || !url || !username || !password}
        >
          {#if discovering}
            <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
            Discovering...
          {:else}
            <Icon icon="mdi:connection" class="w-4 h-4 mr-2" />
            {hasDiscovered ? 'Re-discover Addressbooks' : 'Connect & Discover'}
          {/if}
        </Button>

        {#if discoveryError}
          <div class="p-3 bg-destructive/10 border border-destructive/30 rounded-md text-sm text-destructive">
            {discoveryError}
          </div>
        {/if}

        {#if hasDiscovered && discoveredAddressbooks.length > 0}
          <div class="space-y-2">
            <Label>Addressbooks to sync</Label>
            <div class="border border-border rounded-md divide-y divide-border max-h-40 overflow-y-auto">
              {#each discoveredAddressbooks as ab (ab.path)}
                <button
                  type="button"
                  class="w-full flex items-center gap-3 p-3 text-left hover:bg-muted/50 transition-colors"
                  onclick={() => toggleAddressbook(ab.path)}
                >
                  <div class="w-4 h-4 border border-border rounded flex items-center justify-center {selectedAddressbooks.has(ab.path) ? 'bg-primary border-primary' : ''}">
                    {#if selectedAddressbooks.has(ab.path)}
                      <Icon icon="mdi:check" class="w-3 h-3 text-primary-foreground" />
                    {/if}
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="font-medium text-sm truncate">{ab.name || ab.path}</div>
                    {#if ab.description}
                      <div class="text-xs text-muted-foreground truncate">{ab.description}</div>
                    {/if}
                  </div>
                </button>
              {/each}
            </div>
          </div>

          <div class="space-y-2">
            <Label>Sync interval</Label>
            <Select.Root value={syncIntervalStr} onValueChange={handleSyncIntervalChange}>
              <Select.Trigger>
                <Select.Value placeholder="Select interval">
                  {getSyncIntervalLabel(syncInterval)}
                </Select.Value>
              </Select.Trigger>
              <Select.Content>
                {#each syncIntervalOptions as opt (opt.value)}
                  <Select.Item value={opt.value} label={opt.label} />
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
        {/if}

      {:else}
        <!-- Google or Microsoft OAuth Form -->
        {#if editSource && isEditingOAuthSource}
          <!-- Editing existing OAuth source -->
          <div class="space-y-2">
            <Label for="name">Name</Label>
            <Input
              id="name"
              bind:value={name}
              placeholder="Contact source name"
            />
          </div>

          <div class="space-y-2">
            <Label>Sync interval</Label>
            <Select.Root value={syncIntervalStr} onValueChange={handleSyncIntervalChange}>
              <Select.Trigger>
                <Select.Value placeholder="Select interval">
                  {getSyncIntervalLabel(syncInterval)}
                </Select.Value>
              </Select.Trigger>
              <Select.Content>
                {#each syncIntervalOptions as opt (opt.value)}
                  <Select.Item value={opt.value} label={opt.label} />
                {/each}
              </Select.Content>
            </Select.Root>
          </div>

        {:else}
          <!-- New OAuth source -->
          {#if loadingAccounts}
            <div class="flex items-center justify-center py-8">
              <Icon icon="mdi:loading" class="w-6 h-6 animate-spin text-muted-foreground" />
            </div>
          {:else}
            <!-- Link existing account section -->
            {#if availableAccounts.length > 0}
              <div class="space-y-3">
                <Label>Link to existing {sourceType === 'google' ? 'Google' : 'Microsoft'} account</Label>
                <div class="border border-border rounded-md divide-y divide-border">
                  {#each availableAccounts as account (account.accountId)}
                    <button
                      type="button"
                      class="w-full flex items-center gap-3 p-3 text-left hover:bg-muted/50 transition-colors"
                      onclick={() => {
                        selectedAccountId = account.accountId
                        oauthEmail = ''
                        if (!name) name = `${account.name || account.email} Contacts`
                      }}
                    >
                      <div class="w-4 h-4 border border-border rounded flex items-center justify-center {selectedAccountId === account.accountId ? 'bg-primary border-primary' : ''}">
                        {#if selectedAccountId === account.accountId}
                          <Icon icon="mdi:check" class="w-3 h-3 text-primary-foreground" />
                        {/if}
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="font-medium text-sm truncate">{account.name || account.email}</div>
                        <div class="text-xs text-muted-foreground truncate">{account.email}</div>
                      </div>
                      {#if !account.hasContactScope}
                        <span class="text-xs text-amber-500 flex items-center gap-1">
                          <Icon icon="mdi:alert" class="w-3 h-3" />
                          Re-auth needed
                        </span>
                      {/if}
                    </button>
                  {/each}
                </div>
              </div>

              <div class="relative">
                <div class="absolute inset-0 flex items-center">
                  <span class="w-full border-t border-border"></span>
                </div>
                <div class="relative flex justify-center text-xs uppercase">
                  <span class="bg-background px-2 text-muted-foreground">Or</span>
                </div>
              </div>
            {/if}

            <!-- Sign in with OAuth -->
            <div class="space-y-3">
              <Label>
                {availableAccounts.length > 0 ? 'Sign in with a different account' : `Sign in with ${sourceType === 'google' ? 'Google' : 'Microsoft'}`}
              </Label>

              {#if oauthInProgress}
                <div class="p-4 border border-border rounded-lg text-center space-y-2">
                  <Icon icon="mdi:loading" class="w-8 h-8 animate-spin text-primary mx-auto" />
                  <p class="text-sm text-muted-foreground">
                    Waiting for sign-in... Check your browser
                  </p>
                  <Button variant="ghost" size="sm" onclick={() => {
                    contactSourcesStore.cancelOAuthFlow()
                    oauthInProgress = false
                  }}>
                    Cancel
                  </Button>
                </div>
              {:else if oauthEmail}
                <div class="p-3 border border-green-500/30 bg-green-500/10 rounded-lg flex items-center gap-3">
                  <Icon icon="mdi:check-circle" class="w-5 h-5 text-green-500" />
                  <div class="flex-1">
                    <div class="text-sm font-medium">Signed in as</div>
                    <div class="text-sm text-muted-foreground">{oauthEmail}</div>
                  </div>
                  <Button variant="ghost" size="sm" onclick={() => {
                    oauthEmail = ''
                    name = ''
                  }}>
                    Change
                  </Button>
                </div>
              {:else}
                <Button
                  variant="outline"
                  class="w-full"
                  onclick={handleStartOAuth}
                >
                  <Icon icon={sourceType === 'google' ? 'mdi:google' : 'mdi:microsoft'} class="w-4 h-4 mr-2" />
                  Sign in with {sourceType === 'google' ? 'Google' : 'Microsoft'}
                </Button>
              {/if}
            </div>

            <!-- Name and sync interval (shown when account is selected or OAuth completed) -->
            {#if selectedAccountId || oauthEmail}
              <div class="space-y-4 pt-2">
                <div class="space-y-2">
                  <Label for="oauth-name">Name</Label>
                  <Input
                    id="oauth-name"
                    bind:value={name}
                    placeholder="Contact source name"
                  />
                </div>

                <div class="space-y-2">
                  <Label>Sync interval</Label>
                  <Select.Root value={syncIntervalStr} onValueChange={handleSyncIntervalChange}>
                    <Select.Trigger>
                      <Select.Value placeholder="Select interval">
                        {getSyncIntervalLabel(syncInterval)}
                      </Select.Value>
                    </Select.Trigger>
                    <Select.Content>
                      {#each syncIntervalOptions as opt (opt.value)}
                        <Select.Item value={opt.value} label={opt.label} />
                      {/each}
                    </Select.Content>
                  </Select.Root>
                </div>
              </div>
            {/if}
          {/if}
        {/if}
      {/if}
    </div>

    <!-- Actions -->
    <div class="flex items-center justify-end gap-2 pt-4 border-t border-border">
      <Button variant="ghost" onclick={handleCancel} disabled={saving}>
        Cancel
      </Button>
      <Button
        onclick={handleSave}
        disabled={saving || !canSave()}
      >
        {#if saving}
          <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
        {/if}
        {editSource ? 'Update' : 'Add'}
      </Button>
    </div>
  </Dialog.Content>
</Dialog.Root>

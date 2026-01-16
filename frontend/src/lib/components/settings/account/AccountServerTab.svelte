<script lang="ts">
  import Icon from '@iconify/svelte'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import * as Select from '$lib/components/ui/select'
  import {
    securityOptions,
    syncIntervalOptions,
  } from '$lib/config/providers'
  // @ts-ignore - wailsjs path
  import { account } from '../../../../../wailsjs/go/models'
  // @ts-ignore - wailsjs path
  import { GetAccountFoldersForMapping, GetAutoDetectedFolders } from '../../../../../wailsjs/go/app/App'

  interface Props {
    /** The account being edited */
    editAccount: account.Account
    /** Bound form values */
    imapHost: string
    imapPort: number
    imapSecurity: string
    smtpHost: string
    smtpPort: number
    smtpSecurity: string
    syncInterval: string
    readReceiptRequestPolicy: string
    /** Folder mappings */
    sentFolderPath: string
    draftsFolderPath: string
    trashFolderPath: string
    spamFolderPath: string
    archiveFolderPath: string
    allMailFolderPath: string
    starredFolderPath: string
    /** Validation errors */
    errors: Record<string, string>
    /** Callbacks */
    onImapHostChange: (value: string) => void
    onImapPortChange: (value: number) => void
    onImapSecurityChange: (value: string) => void
    onSmtpHostChange: (value: string) => void
    onSmtpPortChange: (value: number) => void
    onSmtpSecurityChange: (value: string) => void
    onSyncIntervalChange: (value: string) => void
    onReadReceiptPolicyChange: (value: string) => void
    onFolderMappingChange: (type: string, value: string) => void
  }

  let {
    editAccount,
    imapHost = $bindable(),
    imapPort = $bindable(),
    imapSecurity = $bindable(),
    smtpHost = $bindable(),
    smtpPort = $bindable(),
    smtpSecurity = $bindable(),
    syncInterval = $bindable(),
    readReceiptRequestPolicy = $bindable(),
    sentFolderPath = $bindable(),
    draftsFolderPath = $bindable(),
    trashFolderPath = $bindable(),
    spamFolderPath = $bindable(),
    archiveFolderPath = $bindable(),
    allMailFolderPath = $bindable(),
    starredFolderPath = $bindable(),
    errors,
    onImapHostChange,
    onImapPortChange,
    onImapSecurityChange,
    onSmtpHostChange,
    onSmtpPortChange,
    onSmtpSecurityChange,
    onSyncIntervalChange,
    onReadReceiptPolicyChange,
    onFolderMappingChange,
  }: Props = $props()

  // Folder mapping state
  let showFolderMapping = $state(false)
  let loadingFolders = $state(false)
  let availableFolders = $state<any[]>([])
  let autoDetectedFolders = $state<Record<string, string>>({})

  // Read receipt request policy options
  const readReceiptRequestOptions = [
    { value: 'never', label: 'Never request' },
    { value: 'ask', label: 'Ask each time' },
    { value: 'always', label: 'Always request' },
  ]

  // Helper functions
  function getSecurityLabel(value: string): string {
    return securityOptions.find(opt => opt.value === value)?.label || value
  }

  function getSyncIntervalLabel(value: string): string {
    const numValue = Number(value)
    return syncIntervalOptions.find(opt => opt.value === numValue)?.label || `${value} min`
  }

  function getReadReceiptLabel(value: string): string {
    return readReceiptRequestOptions.find(opt => opt.value === value)?.label || value
  }

  // Load folders for mapping UI
  async function loadFoldersForMapping() {
    if (availableFolders.length > 0) return

    loadingFolders = true
    try {
      availableFolders = await GetAccountFoldersForMapping(editAccount.id)
      autoDetectedFolders = await GetAutoDetectedFolders(editAccount.id)
    } catch (err) {
      console.error('Failed to load folders for mapping:', err)
    } finally {
      loadingFolders = false
    }
  }

  function handleFolderMappingToggle() {
    showFolderMapping = !showFolderMapping
    if (showFolderMapping) {
      loadFoldersForMapping()
    }
  }

  // Folder mapping types configuration
  // get() returns saved mapping or falls back to auto-detected folder
  const folderMappingTypes = [
    { key: 'sent', label: 'Sent', get: () => sentFolderPath || autoDetectedFolders['sent'] || '', set: (v: string) => { sentFolderPath = v; onFolderMappingChange('sent', v) }},
    { key: 'drafts', label: 'Drafts', get: () => draftsFolderPath || autoDetectedFolders['drafts'] || '', set: (v: string) => { draftsFolderPath = v; onFolderMappingChange('drafts', v) }},
    { key: 'trash', label: 'Trash', get: () => trashFolderPath || autoDetectedFolders['trash'] || '', set: (v: string) => { trashFolderPath = v; onFolderMappingChange('trash', v) }},
    { key: 'spam', label: 'Spam/Junk', get: () => spamFolderPath || autoDetectedFolders['spam'] || '', set: (v: string) => { spamFolderPath = v; onFolderMappingChange('spam', v) }},
    { key: 'archive', label: 'Archive', get: () => archiveFolderPath || autoDetectedFolders['archive'] || '', set: (v: string) => { archiveFolderPath = v; onFolderMappingChange('archive', v) }},
    { key: 'all', label: 'All Mail', get: () => allMailFolderPath || autoDetectedFolders['all'] || '', set: (v: string) => { allMailFolderPath = v; onFolderMappingChange('all', v) }},
    { key: 'starred', label: 'Starred', get: () => starredFolderPath || autoDetectedFolders['starred'] || '', set: (v: string) => { starredFolderPath = v; onFolderMappingChange('starred', v) }},
  ]
</script>

<div class="space-y-6">
  <!-- Incoming Mail (IMAP) -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:email-receive-outline" class="w-4 h-4" />
      Incoming Mail (IMAP)
    </h3>

    <div class="grid grid-cols-2 gap-3">
      <div class="space-y-2">
        <Label for="imapHost">Server</Label>
        <Input
          id="imapHost"
          type="text"
          placeholder="imap.example.com"
          bind:value={imapHost}
          oninput={(e) => onImapHostChange((e.target as HTMLInputElement).value)}
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
            oninput={(e) => onImapPortChange(Number((e.target as HTMLInputElement).value))}
            class={errors.imapPort ? 'border-destructive' : ''}
          />
        </div>
        <div class="space-y-2">
          <Label>Security</Label>
          <Select.Root 
            value={imapSecurity} 
            onValueChange={(v) => { imapSecurity = v; onImapSecurityChange(v) }}
          >
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

  <!-- Divider -->
  <div class="border-t border-border"></div>

  <!-- Outgoing Mail (SMTP) -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:email-send-outline" class="w-4 h-4" />
      Outgoing Mail (SMTP)
    </h3>

    <div class="grid grid-cols-2 gap-3">
      <div class="space-y-2">
        <Label for="smtpHost">Server</Label>
        <Input
          id="smtpHost"
          type="text"
          placeholder="smtp.example.com"
          bind:value={smtpHost}
          oninput={(e) => onSmtpHostChange((e.target as HTMLInputElement).value)}
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
            oninput={(e) => onSmtpPortChange(Number((e.target as HTMLInputElement).value))}
            class={errors.smtpPort ? 'border-destructive' : ''}
          />
        </div>
        <div class="space-y-2">
          <Label>Security</Label>
          <Select.Root 
            value={smtpSecurity} 
            onValueChange={(v) => { smtpSecurity = v; onSmtpSecurityChange(v) }}
          >
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

  <!-- Divider -->
  <div class="border-t border-border"></div>

  <!-- Check for New Mail -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:refresh" class="w-4 h-4" />
      Sync Options
    </h3>

    <div class="space-y-2">
      <Label>Check for New Mail</Label>
      <Select.Root 
        value={syncInterval} 
        onValueChange={(v) => { syncInterval = v; onSyncIntervalChange(v) }}
      >
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

    <div class="space-y-2">
      <Label>Request Read Receipts</Label>
      <Select.Root 
        value={readReceiptRequestPolicy} 
        onValueChange={(v) => { readReceiptRequestPolicy = v; onReadReceiptPolicyChange(v) }}
      >
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
  </div>

  <!-- Divider -->
  <div class="border-t border-border"></div>

  <!-- Folder Mapping -->
  <div class="space-y-2">
    <button
      type="button"
      class="flex items-center gap-2 text-sm font-medium hover:text-primary transition-colors"
      onclick={handleFolderMappingToggle}
    >
      <Icon
        icon={showFolderMapping ? 'mdi:chevron-down' : 'mdi:chevron-right'}
        class="w-4 h-4"
      />
      <Icon icon="mdi:folder-cog-outline" class="w-4 h-4" />
      Folder Mapping
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
            {#each folderMappingTypes as mapping (mapping.key)}
              <div class="grid grid-cols-[100px_1fr] items-center gap-2">
                <Label class="text-sm">{mapping.label}:</Label>
                <Select.Root value={mapping.get()} onValueChange={mapping.set}>
                  <Select.Trigger class="h-9">
                    <Select.Value placeholder="None">
                      {mapping.get() || 'None'}
                    </Select.Value>
                  </Select.Trigger>
                  <Select.Content>
                    <Select.Item value="" label="None" />
                    {#each availableFolders as f (f.path)}
                      <Select.Item 
                        value={f.path} 
                        label={f.path + (autoDetectedFolders[mapping.key] === f.path ? ' (detected)' : '')} 
                      />
                    {/each}
                  </Select.Content>
                </Select.Root>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>

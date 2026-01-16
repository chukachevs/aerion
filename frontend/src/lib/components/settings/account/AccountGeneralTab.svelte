<script lang="ts">
  import Icon from '@iconify/svelte'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import * as Select from '$lib/components/ui/select'
  import { ColorPicker } from '$lib/components/ui/color-picker'
  import { Button } from '$lib/components/ui/button'
  import {
    syncPeriodOptions,
  } from '$lib/config/providers'
  // @ts-ignore - wailsjs path
  import { account } from '../../../../../wailsjs/go/models'

  interface Props {
    /** The account being edited */
    editAccount: account.Account
    /** Bound form values */
    name: string
    displayName: string
    color: string
    email: string
    username: string
    password: string
    syncPeriodDays: string
    /** Auth type from account */
    authType: string
    /** Validation errors */
    errors: Record<string, string>
    /** Whether re-authorization is in progress */
    reauthorizing?: boolean
    /** Whether re-authorization succeeded */
    reauthorizeSuccess?: boolean
    /** Callbacks */
    onNameChange: (value: string) => void
    onDisplayNameChange: (value: string) => void
    onColorChange: (value: string) => void
    onUsernameChange: (value: string) => void
    onPasswordChange: (value: string) => void
    onSyncPeriodChange: (value: string) => void
    onReauthorize?: () => void
  }

  let {
    editAccount,
    name = $bindable(),
    displayName = $bindable(),
    color = $bindable(),
    email = $bindable(),
    username = $bindable(),
    password = $bindable(),
    syncPeriodDays = $bindable(),
    authType,
    errors,
    reauthorizing = false,
    reauthorizeSuccess = false,
    onNameChange,
    onDisplayNameChange,
    onColorChange,
    onUsernameChange,
    onPasswordChange,
    onSyncPeriodChange,
    onReauthorize,
  }: Props = $props()

  function getSyncPeriodLabel(value: string): string {
    const numValue = Number(value)
    return syncPeriodOptions.find(opt => opt.value === numValue)?.label || `${value} days`
  }
</script>

<div class="space-y-6">
  <!-- Account Identification -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:account-circle-outline" class="w-4 h-4" />
      Account Identification
    </h3>

    <div class="space-y-2">
      <Label for="name">Account Name</Label>
      <div class="flex items-center gap-3">
        <ColorPicker value={color} onchange={(c) => { color = c; onColorChange(c) }} />
        <Input
          id="name"
          type="text"
          placeholder="e.g., Personal, Work"
          bind:value={name}
          oninput={(e) => onNameChange((e.target as HTMLInputElement).value)}
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
      <Label for="displayName">Default Display Name</Label>
      <Input
        id="displayName"
        type="text"
        placeholder="e.g., John Smith"
        bind:value={displayName}
        oninput={(e) => onDisplayNameChange((e.target as HTMLInputElement).value)}
        class={errors.displayName ? 'border-destructive' : ''}
      />
      <p class="text-xs text-muted-foreground">
        Name shown to email recipients (can be customized per email address in Identity tab)
      </p>
      {#if errors.displayName}
        <p class="text-sm text-destructive">{errors.displayName}</p>
      {/if}
    </div>
  </div>

  <!-- Divider -->
  <div class="border-t border-border"></div>

  <!-- Credentials -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:key-outline" class="w-4 h-4" />
      Credentials
    </h3>

    <div class="space-y-2">
      <Label for="email">Email Address</Label>
      <Input
        id="email"
        type="email"
        value={email}
        disabled
        class="bg-muted"
      />
      <p class="text-xs text-muted-foreground">
        Primary email address (cannot be changed)
      </p>
    </div>

    <div class="space-y-2">
      <Label for="username">Username</Label>
      <Input
        id="username"
        type="text"
        placeholder="Usually your email address"
        bind:value={username}
        oninput={(e) => onUsernameChange((e.target as HTMLInputElement).value)}
      />
      <p class="text-xs text-muted-foreground">
        Leave empty to use email address
      </p>
    </div>

    {#if authType === 'oauth2'}
      <!-- OAuth account -->
      <div class="space-y-2">
        <Label>Authentication</Label>
        <div class="rounded-lg border {reauthorizeSuccess ? 'border-green-500 bg-green-500/5' : 'border-border'} p-4 transition-colors">
          <div class="flex items-center gap-3">
            <div class="flex-shrink-0 w-10 h-10 rounded-full {reauthorizeSuccess ? 'bg-green-500/20' : 'bg-primary/10'} flex items-center justify-center transition-colors">
              {#if reauthorizeSuccess}
                <Icon icon="mdi:check-circle" class="w-5 h-5 text-green-500" />
              {:else}
                <Icon icon="mdi:shield-check" class="w-5 h-5 text-primary" />
              {/if}
            </div>
            <div class="flex-1">
              {#if reauthorizeSuccess}
                <p class="text-sm font-medium text-green-600 dark:text-green-400">Re-authorized successfully!</p>
                <p class="text-xs text-muted-foreground">
                  Your account has a fresh OAuth token
                </p>
              {:else}
                <p class="text-sm font-medium">Connected via OAuth</p>
                <p class="text-xs text-muted-foreground">
                  Your account is securely connected
                </p>
              {/if}
            </div>
            {#if !reauthorizeSuccess}
              <Button
                variant="outline"
                size="sm"
                onclick={onReauthorize}
                disabled={reauthorizing}
              >
                {#if reauthorizing}
                  <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
                  Authorizing...
                {:else}
                  <Icon icon="mdi:refresh" class="w-4 h-4 mr-2" />
                  Re-authorize
                {/if}
              </Button>
            {/if}
          </div>
        </div>
        {#if !reauthorizeSuccess}
          <p class="text-xs text-muted-foreground">
            If you're having sync issues, try re-authorizing to get a fresh token
          </p>
        {/if}
      </div>
    {:else}
      <!-- Password account -->
      <div class="space-y-2">
        <Label for="password">Password</Label>
        <Input
          id="password"
          type="password"
          placeholder="Leave empty to keep current"
          bind:value={password}
          oninput={(e) => onPasswordChange((e.target as HTMLInputElement).value)}
          class={errors.password ? 'border-destructive' : ''}
        />
        {#if errors.password}
          <p class="text-sm text-destructive">{errors.password}</p>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Divider -->
  <div class="border-t border-border"></div>

  <!-- Sync Settings -->
  <div class="space-y-4">
    <h3 class="text-sm font-medium flex items-center gap-2">
      <Icon icon="mdi:sync" class="w-4 h-4" />
      Sync Settings
    </h3>

    <div class="space-y-2">
      <Label>Sync Period</Label>
      <Select.Root 
        value={syncPeriodDays} 
        onValueChange={(v) => { syncPeriodDays = v; onSyncPeriodChange(v) }}
      >
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
  </div>
</div>

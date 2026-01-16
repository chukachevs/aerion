<script lang="ts">
  import Icon from '@iconify/svelte'
  import * as Dialog from '$lib/components/ui/dialog'
  import { Button } from '$lib/components/ui/button'
  import { Input } from '$lib/components/ui/input'
  import { Label } from '$lib/components/ui/label'
  import SignatureEditor from './SignatureEditor.svelte'
  // @ts-ignore - wailsjs path
  import { account } from '../../../../../wailsjs/go/models'

  interface Props {
    /** Whether the dialog is open */
    open?: boolean
    /** Identity to edit (null for new identity) */
    identity?: account.Identity | null
    /** Account ID (required for creating new identity) */
    accountId: string
    /** Callback when dialog should close */
    onClose?: () => void
    /** Callback when identity is saved */
    onSave?: (config: account.IdentityConfig) => Promise<void>
  }

  let {
    open = $bindable(false),
    identity = null,
    accountId,
    onClose,
    onSave,
  }: Props = $props()

  // Form state
  let email = $state('')
  let name = $state('')
  let signatureHtml = $state('')
  let signatureText = $state('')
  let signatureEnabled = $state(true)
  let signatureForNew = $state(true)
  let signatureForReply = $state(true)
  let signatureForForward = $state(true)
  let signaturePlacement = $state<'above' | 'below'>('above')
  let signatureSeparator = $state(false)

  let saving = $state(false)
  let errors = $state<Record<string, string>>({})

  // Initialize form when identity changes
  $effect(() => {
    if (open) {
      if (identity) {
        // Editing existing identity
        email = identity.email || ''
        name = identity.name || ''
        signatureHtml = identity.signatureHtml || ''
        signatureText = identity.signatureText || ''
        signatureEnabled = identity.signatureEnabled ?? true
        signatureForNew = identity.signatureForNew ?? true
        signatureForReply = identity.signatureForReply ?? true
        signatureForForward = identity.signatureForForward ?? true
        signaturePlacement = (identity.signaturePlacement as 'above' | 'below') || 'above'
        signatureSeparator = identity.signatureSeparator ?? false
      } else {
        // New identity - reset to defaults
        email = ''
        name = ''
        signatureHtml = ''
        signatureText = ''
        signatureEnabled = true
        signatureForNew = true
        signatureForReply = true
        signatureForForward = true
        signaturePlacement = 'above'
        signatureSeparator = false
      }
      errors = {}
    }
  })

  function validate(): boolean {
    errors = {}
    
    if (!email.trim()) {
      errors.email = 'Email address is required'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      errors.email = 'Invalid email format'
    }
    
    if (!name.trim()) {
      errors.name = 'Display name is required'
    }
    
    return Object.keys(errors).length === 0
  }

  async function handleSave() {
    if (!validate()) return
    
    saving = true
    try {
      const config = new account.IdentityConfig({
        email: email.trim(),
        name: name.trim(),
        signatureHtml,
        signatureText,
        signatureEnabled,
        signatureForNew,
        signatureForReply,
        signatureForForward,
        signaturePlacement,
        signatureSeparator,
      })
      
      await onSave?.(config)
      open = false
      onClose?.()
    } catch (err) {
      console.error('Failed to save identity:', err)
      errors.general = err instanceof Error ? err.message : String(err)
    } finally {
      saving = false
    }
  }

  function handleCancel() {
    open = false
    onClose?.()
  }

  function handleOpenChange(isOpen: boolean) {
    open = isOpen
    if (!isOpen) {
      onClose?.()
    }
  }

  // Convert HTML to plain text for the "Generate from HTML" button
  function generatePlainTextFromHtml() {
    if (!signatureHtml) return
    
    const temp = document.createElement('div')
    temp.innerHTML = signatureHtml
    
    // Replace <br> and block elements with newlines
    const blockElements = temp.querySelectorAll('p, div, br, li')
    blockElements.forEach(el => {
      if (el.tagName === 'BR') {
        el.replaceWith('\n')
      } else if (el.tagName === 'LI') {
        el.prepend(document.createTextNode('- '))
        el.append(document.createTextNode('\n'))
      } else {
        el.append(document.createTextNode('\n'))
      }
    })
    
    let text = temp.textContent || ''
    text = text.replace(/\n{3,}/g, '\n\n')
    signatureText = text.trim()
  }
</script>

<Dialog.Root bind:open onOpenChange={handleOpenChange}>
  <Dialog.Content class="max-w-lg max-h-[90vh] overflow-y-auto">
    <Dialog.Header>
      <Dialog.Title>
        {identity ? 'Edit Email Address' : 'Add Email Address'}
      </Dialog.Title>
      <Dialog.Description>
        {identity 
          ? 'Update this email address and its signature settings.'
          : 'Add a new email address you can send from.'}
      </Dialog.Description>
    </Dialog.Header>

    <form onsubmit={(e) => { e.preventDefault(); handleSave() }} class="space-y-6">
      <!-- Email & Name -->
      <div class="space-y-4">
        <div class="space-y-2">
          <Label for="email">Email Address</Label>
          <Input
            id="email"
            type="email"
            placeholder="you@example.com"
            bind:value={email}
            class={errors.email ? 'border-destructive' : ''}
          />
          {#if errors.email}
            <p class="text-sm text-destructive">{errors.email}</p>
          {/if}
        </div>

        <div class="space-y-2">
          <Label for="name">Display Name</Label>
          <Input
            id="name"
            type="text"
            placeholder="John Smith"
            bind:value={name}
            class={errors.name ? 'border-destructive' : ''}
          />
          <p class="text-xs text-muted-foreground">
            Name shown to email recipients
          </p>
          {#if errors.name}
            <p class="text-sm text-destructive">{errors.name}</p>
          {/if}
        </div>
      </div>

      <!-- Divider -->
      <div class="border-t border-border"></div>

      <!-- Signature Section -->
      <div class="space-y-4">
        <div class="flex items-center gap-2">
          <input
            type="checkbox"
            id="signatureEnabled"
            bind:checked={signatureEnabled}
            class="w-4 h-4 rounded border-input accent-primary"
          />
          <Label for="signatureEnabled" class="cursor-pointer font-medium">
            Use signature
          </Label>
        </div>

        {#if signatureEnabled}
          <!-- HTML Signature Editor -->
          <div class="space-y-2">
            <Label>HTML Signature</Label>
            <SignatureEditor
              value={signatureHtml}
              placeholder="Enter your signature..."
              onchange={(html) => signatureHtml = html}
            />
            <p class="text-xs text-muted-foreground">
              You can paste HTML content or add images via URL or file upload
            </p>
          </div>

          <!-- Plain Text Signature -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <Label for="signatureText">Plain Text Signature (optional)</Label>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onclick={generatePlainTextFromHtml}
                disabled={!signatureHtml}
                class="text-xs h-7"
              >
                Generate from HTML
              </Button>
            </div>
            <textarea
              id="signatureText"
              bind:value={signatureText}
              placeholder="Plain text version for text-only emails..."
              class="w-full min-h-[80px] p-3 text-sm bg-background border border-input rounded-md resize-y focus:outline-none focus:ring-2 focus:ring-ring font-mono"
            ></textarea>
            <p class="text-xs text-muted-foreground">
              Used when sending plain text emails
            </p>
          </div>

          <!-- Divider -->
          <div class="border-t border-border"></div>

          <!-- Signature Behavior -->
          <div class="space-y-4">
            <Label class="font-medium">Append signature to:</Label>
            <div class="flex flex-wrap gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={signatureForNew}
                  class="w-4 h-4 rounded border-input accent-primary"
                />
                <span class="text-sm">New messages</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={signatureForReply}
                  class="w-4 h-4 rounded border-input accent-primary"
                />
                <span class="text-sm">Replies</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={signatureForForward}
                  class="w-4 h-4 rounded border-input accent-primary"
                />
                <span class="text-sm">Forwards</span>
              </label>
            </div>
          </div>

          <!-- Signature Placement -->
          <div class="space-y-3">
            <Label class="font-medium">Signature placement in replies/forwards:</Label>
            <div class="flex gap-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="placement"
                  value="above"
                  bind:group={signaturePlacement}
                  class="w-4 h-4 accent-primary"
                />
                <span class="text-sm">Above quoted text</span>
              </label>
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="placement"
                  value="below"
                  bind:group={signaturePlacement}
                  class="w-4 h-4 accent-primary"
                />
                <span class="text-sm">Below quoted text</span>
              </label>
            </div>
          </div>

          <!-- Separator Option -->
          <div class="flex items-center gap-2">
            <input
              type="checkbox"
              id="signatureSeparator"
              bind:checked={signatureSeparator}
              class="w-4 h-4 rounded border-input accent-primary"
            />
            <Label for="signatureSeparator" class="cursor-pointer text-sm">
              Add separator line before signature (<code class="text-xs bg-muted px-1 rounded">-- </code>)
            </Label>
          </div>
        {/if}
      </div>

      <!-- Error message -->
      {#if errors.general}
        <div class="flex items-start gap-2 p-3 rounded-lg bg-destructive/10 border border-destructive/20">
          <Icon icon="mdi:alert-circle" class="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
          <p class="text-sm text-destructive">{errors.general}</p>
        </div>
      {/if}

      <!-- Actions -->
      <div class="flex items-center justify-end gap-2 pt-4 border-t border-border">
        <Button type="button" variant="ghost" onclick={handleCancel} disabled={saving}>
          Cancel
        </Button>
        <Button type="submit" disabled={saving}>
          {#if saving}
            <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
          {/if}
          {identity ? 'Save Changes' : 'Add Email Address'}
        </Button>
      </div>
    </form>
  </Dialog.Content>
</Dialog.Root>

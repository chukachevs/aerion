<script lang="ts">
  import Icon from '@iconify/svelte'
  import * as AlertDialog from '$lib/components/ui/alert-dialog'
  import { accountStore } from '$lib/stores/accounts.svelte'
  // @ts-ignore - wailsjs path
  import { account } from '../../../../wailsjs/go/models'

  interface Props {
    /** Whether the dialog is open */
    open?: boolean
    /** Account to delete */
    account: account.Account | null
    /** Callback when dialog should close */
    onClose?: () => void
    /** Callback when account is successfully deleted */
    onSuccess?: () => void
  }

  let {
    open = $bindable(false),
    account: accountToDelete = null,
    onClose,
    onSuccess,
  }: Props = $props()

  let deleting = $state(false)
  let error = $state<string | null>(null)

  async function handleDelete() {
    if (!accountToDelete) return

    deleting = true
    error = null

    try {
      await accountStore.removeAccount(accountToDelete.id)
      onSuccess?.()
      open = false
      onClose?.()
    } catch (err) {
      error = err instanceof Error ? err.message : String(err)
    } finally {
      deleting = false
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
      error = null
    }
  }
</script>

<AlertDialog.Root bind:open onOpenChange={handleOpenChange}>
  <AlertDialog.Content preventCloseAutoFocus>
    <AlertDialog.Header>
      <AlertDialog.Title>Delete Account</AlertDialog.Title>
      <AlertDialog.Description>
        Are you sure you want to delete <strong>{accountToDelete?.name}</strong> ({accountToDelete?.email})?
        <br /><br />
        This will permanently remove all local messages, folders, and settings for this account.
        Your emails on the server will not be affected.
      </AlertDialog.Description>
    </AlertDialog.Header>

    {#if error}
      <div class="flex items-start gap-2 p-3 rounded-lg bg-destructive/10 border border-destructive/20">
        <Icon icon="mdi:alert-circle" class="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
        <p class="text-sm text-destructive">{error}</p>
      </div>
    {/if}

    <AlertDialog.Footer>
      <AlertDialog.Cancel onclick={handleCancel} disabled={deleting}>
        Cancel
      </AlertDialog.Cancel>
      <AlertDialog.Action
        onclick={handleDelete}
        disabled={deleting}
        class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
      >
        {#if deleting}
          <Icon icon="mdi:loading" class="w-4 h-4 mr-2 animate-spin" />
        {/if}
        Delete Account
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

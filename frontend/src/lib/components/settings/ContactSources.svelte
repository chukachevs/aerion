<script lang="ts">
  import Icon from '@iconify/svelte'
  import { onMount } from 'svelte'
  import { Button } from '$lib/components/ui/button'
  import { ConfirmDialog } from '$lib/components/ui/confirm-dialog'
  import { contactSourcesStore } from '$lib/stores/contactSources.svelte'
  import { addToast } from '$lib/stores/toast'
  import ContactSourceDialog from './ContactSourceDialog.svelte'
  import { formatDistanceToNow } from 'date-fns'
  // @ts-ignore - wailsjs path
  import type { carddav } from '../../../../wailsjs/go/models'

  // Dialog state
  let showAddDialog = $state(false)
  let editingSource = $state<carddav.Source | null>(null)
  let syncingSourceId = $state<string | null>(null)

  // Delete confirmation state
  let showDeleteConfirm = $state(false)
  let deletingSource = $state<carddav.Source | null>(null)
  let isDeleting = $state(false)

  onMount(() => {
    contactSourcesStore.load()
  })

  function formatLastSync(source: carddav.Source): string {
    if (!source.last_synced_at) return 'Never synced'
    try {
      return `Synced ${formatDistanceToNow(new Date(source.last_synced_at), { addSuffix: true })}`
    } catch {
      return 'Never synced'
    }
  }

  async function handleSync(sourceId: string) {
    syncingSourceId = sourceId
    try {
      await contactSourcesStore.syncSource(sourceId)
      addToast({ type: 'success', message: 'Contact source synced' })
    } catch (err) {
      addToast({ type: 'error', message: `Sync failed: ${err}` })
    } finally {
      syncingSourceId = null
    }
  }

  function handleDelete(source: carddav.Source) {
    deletingSource = source
    showDeleteConfirm = true
  }

  async function confirmDelete() {
    if (!deletingSource) return
    isDeleting = true
    try {
      await contactSourcesStore.deleteSource(deletingSource.id)
      addToast({ type: 'success', message: 'Contact source deleted' })
      showDeleteConfirm = false
      deletingSource = null
    } catch (err) {
      addToast({ type: 'error', message: `Failed to delete: ${err}` })
    } finally {
      isDeleting = false
    }
  }

  function cancelDelete() {
    showDeleteConfirm = false
    deletingSource = null
  }

  function openEdit(source: carddav.Source) {
    editingSource = source
    showAddDialog = true
  }

  function openAdd() {
    editingSource = null
    showAddDialog = true
  }

  function handleDialogClose() {
    showAddDialog = false
    editingSource = null
    contactSourcesStore.refresh()
  }
</script>

<div class="space-y-4">
  <h3 class="text-sm font-medium flex items-center gap-2">
    <Icon icon="mdi:contacts-outline" class="w-4 h-4" />
    Contact Sources
  </h3>

  {#if contactSourcesStore.loading}
    <div class="flex items-center justify-center py-4">
      <Icon icon="mdi:loading" class="w-5 h-5 animate-spin text-muted-foreground" />
    </div>
  {:else if contactSourcesStore.sources.length === 0}
    <div class="text-sm text-muted-foreground py-4 text-center">
      <p class="mb-3">No contact sources configured</p>
      <Button size="sm" onclick={openAdd}>
        <Icon icon="mdi:plus" class="w-4 h-4 mr-1" />
        Add Contact Source
      </Button>
    </div>
  {:else}
    <div class="space-y-2">
      {#each contactSourcesStore.sources as source (source.id)}
        <div class="p-3 border border-border rounded-lg space-y-2 {source.last_error ? 'border-destructive/50 bg-destructive/5' : ''}">
          <!-- Source header -->
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-2">
              <Icon
                icon={source.type === 'google' ? 'mdi:google' : source.type === 'microsoft' ? 'mdi:microsoft' : 'mdi:card-account-details'}
                class="w-5 h-5 {source.enabled ? 'text-primary' : 'text-muted-foreground'}"
              />
              <div>
                <div class="font-medium text-sm">{source.name}</div>
                <div class="text-xs text-muted-foreground flex items-center gap-1.5">
                  <span class="capitalize">{source.type}</span>
                  {#if source.account_id}
                    <span class="text-muted-foreground/50">·</span>
                    <span class="text-muted-foreground/80">linked</span>
                  {/if}
                </div>
              </div>
            </div>
            <div class="text-xs text-muted-foreground">
              {formatLastSync(source)}
            </div>
          </div>

          <!-- Error display -->
          {#if source.last_error}
            <div class="flex items-start gap-2 p-2 bg-destructive/10 rounded text-sm">
              <Icon icon="mdi:alert-circle" class="w-4 h-4 text-destructive shrink-0 mt-0.5" />
              <div class="flex-1">
                <div class="text-destructive font-medium">Sync failed</div>
                <div class="text-xs text-muted-foreground">{source.last_error}</div>
              </div>
            </div>
          {/if}

          <!-- Actions -->
          <div class="flex items-center gap-2 pt-1">
            <Button 
              size="sm" 
              variant="ghost" 
              onclick={() => handleSync(source.id)}
              disabled={syncingSourceId === source.id}
            >
              {#if syncingSourceId === source.id}
                <Icon icon="mdi:loading" class="w-4 h-4 mr-1 animate-spin" />
              {:else}
                <Icon icon="mdi:sync" class="w-4 h-4 mr-1" />
              {/if}
              {source.last_error ? 'Retry' : 'Sync'}
            </Button>
            <Button size="sm" variant="ghost" onclick={() => openEdit(source)}>
              <Icon icon="mdi:pencil" class="w-4 h-4 mr-1" />
              Edit
            </Button>
            <Button size="sm" variant="ghost" class="text-destructive hover:text-destructive" onclick={() => handleDelete(source)}>
              <Icon icon="mdi:delete" class="w-4 h-4 mr-1" />
              Delete
            </Button>
          </div>
        </div>
      {/each}

      <!-- Add button -->
      <Button size="sm" variant="outline" class="w-full" onclick={openAdd}>
        <Icon icon="mdi:plus" class="w-4 h-4 mr-1" />
        Add Contact Source
      </Button>
    </div>
  {/if}
</div>

<!-- Add/Edit Dialog -->
<ContactSourceDialog
  bind:open={showAddDialog}
  editSource={editingSource}
  onClose={handleDialogClose}
/>

<!-- Delete Confirmation Dialog -->
<ConfirmDialog
  bind:open={showDeleteConfirm}
  title="Delete Contact Source"
  description={`Delete "${deletingSource?.name}"? This will remove all synced contacts from this source.`}
  confirmLabel="Delete"
  cancelLabel="Cancel"
  variant="destructive"
  loading={isDeleting}
  onConfirm={confirmDelete}
  onCancel={cancelDelete}
/>

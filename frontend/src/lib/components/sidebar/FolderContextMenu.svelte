<script lang="ts">
  import Icon from '@iconify/svelte'
  import { ContextMenu as ContextMenuPrimitive } from 'bits-ui'
  import {
    ContextMenuContent,
    ContextMenuItem,
  } from '$lib/components/ui/context-menu'
  import {
    MarkAllFolderMessagesAsRead,
    MarkAllFolderMessagesAsUnread,
    Undo,
  } from '../../../../wailsjs/go/app/App'
  import { toasts } from '$lib/stores/toast'
  import type { Snippet } from 'svelte'

  interface Props {
    folderId: string
    children?: Snippet
  }

  let {
    folderId,
    children,
  }: Props = $props()

  async function handleUndo() {
    try {
      const description = await Undo()
      toasts.success(`Undone: ${description}`)
    } catch (err) {
      toasts.error(`Undo failed: ${err}`)
    }
  }

  async function handleMarkAllRead() {
    try {
      await MarkAllFolderMessagesAsRead(folderId)
      toasts.success('Marked all as read', [{ label: 'Undo', onClick: handleUndo }])
    } catch (err) {
      toasts.error(`Failed to mark all as read: ${err}`)
    }
  }

  async function handleMarkAllUnread() {
    try {
      await MarkAllFolderMessagesAsUnread(folderId)
      toasts.success('Marked all as unread', [{ label: 'Undo', onClick: handleUndo }])
    } catch (err) {
      toasts.error(`Failed to mark all as unread: ${err}`)
    }
  }
</script>

<ContextMenuPrimitive.Root>
  <ContextMenuPrimitive.Trigger>
    {#if children}
      {@render children()}
    {/if}
  </ContextMenuPrimitive.Trigger>

  <ContextMenuContent>
    <ContextMenuItem onSelect={handleMarkAllRead}>
      <Icon icon="mdi:email-check-outline" class="mr-2 h-4 w-4" />
      Mark All as Read
    </ContextMenuItem>
    <ContextMenuItem onSelect={handleMarkAllUnread}>
      <Icon icon="mdi:email-outline" class="mr-2 h-4 w-4" />
      Mark All as Unread
    </ContextMenuItem>
  </ContextMenuContent>
</ContextMenuPrimitive.Root>

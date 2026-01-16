<script lang="ts">
  import { AlertDialog as AlertDialogPrimitive } from 'bits-ui'
  import { cn } from '$lib/utils'
  import type { Snippet } from 'svelte'

  interface Props {
    class?: string
    children?: Snippet
    onOpenAutoFocus?: (e: Event) => void
    /** Prevent focus from returning to trigger element on close */
    preventCloseAutoFocus?: boolean
  }

  let { class: className, children, onOpenAutoFocus, preventCloseAutoFocus = false }: Props = $props()

  function handleCloseAutoFocus(e: Event) {
    if (preventCloseAutoFocus) {
      e.preventDefault()
    }
  }
</script>

<AlertDialogPrimitive.Portal>
  <AlertDialogPrimitive.Overlay
    class="fixed inset-0 z-50 bg-black/80 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
  />
  <AlertDialogPrimitive.Content
    class={cn(
      'fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg',
      className
    )}
    {onOpenAutoFocus}
    onCloseAutoFocus={handleCloseAutoFocus}
  >
    {#if children}
      {@render children()}
    {/if}
  </AlertDialogPrimitive.Content>
</AlertDialogPrimitive.Portal>

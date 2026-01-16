// Runes-based settings store
// Provides reactive state for application settings

// @ts-ignore - wailsjs path
import { GetMessageListDensity, GetMessageListSortOrder, GetThemeMode } from '../../../wailsjs/go/app/App'

export type MessageListDensity = 'micro' | 'compact' | 'standard' | 'large'
export type MessageListSortOrder = 'newest' | 'oldest'
export type ThemeMode = 'system' | 'light' | 'dark'

// Module-level reactive state
let messageListDensity = $state<MessageListDensity>('standard')
let messageListSortOrder = $state<MessageListSortOrder>('newest')
let themeMode = $state<ThemeMode>('system')

// Getter functions to access the state
export function getMessageListDensity(): MessageListDensity {
  return messageListDensity
}

export function getMessageListSortOrder(): MessageListSortOrder {
  return messageListSortOrder
}

export function getThemeMode(): ThemeMode {
  return themeMode
}

// Setter functions to update the state
export function setMessageListDensity(density: MessageListDensity) {
  messageListDensity = density
}

export function setMessageListSortOrder(sortOrder: MessageListSortOrder) {
  messageListSortOrder = sortOrder
}

export function setThemeMode(mode: ThemeMode) {
  themeMode = mode
}

// Load settings from backend (call on app startup)
export async function loadSettings(): Promise<ThemeMode> {
  try {
    const [density, sortOrder, theme] = await Promise.all([
      GetMessageListDensity(),
      GetMessageListSortOrder(),
      GetThemeMode(),
    ])
    messageListDensity = (density as MessageListDensity) || 'standard'
    messageListSortOrder = (sortOrder as MessageListSortOrder) || 'newest'
    themeMode = (theme as ThemeMode) || 'system'
    return themeMode
  } catch (err) {
    console.error('Failed to load settings:', err)
    return 'system'
  }
}

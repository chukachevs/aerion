import './app.css'
import ComposerApp from './ComposerApp.svelte'
import { mount } from 'svelte'

const app = mount(ComposerApp, {
  target: document.getElementById('app')!,
})

export default app

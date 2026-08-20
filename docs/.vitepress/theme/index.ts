import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import CustomHome from './components/CustomHome.vue'
import './style.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ app }) {
    app.component('CustomHome', CustomHome)
  }
} satisfies Theme

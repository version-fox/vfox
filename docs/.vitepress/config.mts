import { defineConfig } from 'vitepress'
import { en } from './en'
import { zh } from './zh'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "vfox",
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['link', { rel: 'icon', type: 'image/png', href: '/logo.png' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'vfox | The Multiple SDK Version Manager' }],
    ['meta', { property: 'og:site_name', content: 'VitePress' }],
  ],
  locales: {
    root: {
      label: 'English',
      ...en
    },
    'zh-hans': {
      label: '简体中文',
      ...zh
    }
  },
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    search: {
      provider: "local",
    },
    logo: "/logo.png",
    socialLinks: [
      { icon: 'github', link: 'https://github.com/version-fox/vfox' },
      { icon: 'discord', link: 'https://discord.com/invite/85c8ptYgb7' }
    ],
  },
})

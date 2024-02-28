import { defineConfig } from 'vitepress'
import { en } from './en'
import { zh } from './zh'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "vfox",
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

---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

title: vfox
titleTemplate: 跨平台、可拓展的版本管理器

hero:
  name: vfox
  text: 跨平台、可拓展的版本管理器
  tagline: 😉轻松管理你的工具和运行环境~
  image:
    src: /logo.png
    alt: vfox
  actions:
    - theme: brand
      text: 👋快速上手
      link: /zh-hans/guides/quick-start
    - theme: alt
      text: 为什么选择vfox?
      link: /zh-hans/guides/intro
    - theme: alt
      text: 查看GitHub
      link: https://github.com/version-fox/vfox

features:
  - title: 跨平台
    details: "支持Windows(非WSL)、Linux、macOS!"
    icon: 💻
  - title: 插件
    details: "简单的API, 添加新工具的支持变得轻而易举！"
    icon: 🔌
  - title: "Shells"
    details: "支持 Powershell、Bash、ZSH、Fish、Clink和Nushell，并提供补全功能。"
    icon: 🐚
  - title: 向后兼容
    details: "支持从现有配置文件.tool-versions、.nvmrc、.node-version、.sdkmanrc平滑迁移！"
    icon: ⏮
  - title: "一个配置文件"
    details: "一个可共享的 .vfox.toml/vfox.toml 配置文件管理所有工具、运行环境及其版本。"
    icon: 📄
---


<style>
:root {
  --vp-home-hero-name-color: transparent;
  --vp-home-hero-name-background: -webkit-linear-gradient(120deg, #fd9620 26%, #ab7c44);
  --vp-home-hero-image-background-image: linear-gradient(30deg, #fa9943, #eeecec);
  --vp-home-hero-image-filter: blur(44px);
}

@media (min-width: 640px) {
  :root {
    --vp-home-hero-image-filter: blur(56px);
  }
}

@media (min-width: 960px) {
  :root {
    --vp-home-hero-image-filter: blur(68px);
  }
}
</style>

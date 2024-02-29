---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: vfox
  text: The Multiple SDK Version Manager
  tagline: 😉Easily manage all your SDK versions~
  image:
    src: /logo.png
    alt: vfox
  actions:
    - theme: brand
      text: 👋Get Started
      link: /guides/quick-start
    - theme: alt
      text: Why use vfox?
      link: /guides/intro
    - theme: alt
      text: View on GitHub
      link: https://github.com/version-fox/vfox

features:
  - title: Cross-platform
    details: "Supports Windows (non-WSL), Linux, macOS!"
    icon: 💻
  - title: Plugins
    details: "Simple API, making it easy to add support for new tools!"
    icon: 🔌
  - title: "Shells"
    details: "Supports Powershell, Bash, ZSH and Fish, with autocomplete feature."
    icon: 🐚
  - title: "One Config File"
    details: ".tool-versions manages all tools, runtime environments and their versions."
    icon: 📄
---


<style>
:root {
  --vp-home-hero-name-color: transparent;
--vp-home-hero-name-background: -webkit-linear-gradient(120deg, #fd9620 26%, #ab7c44);
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
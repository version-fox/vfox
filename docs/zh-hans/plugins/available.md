---
title: vfox
titleTemplate: 可用插件列表
layout: doc
editLink: false
---

<script setup>
import { ref,computed } from 'vue'
import axios from 'axios'

const info = ref({})
const success = ref(false)

axios.get('https://vfox-plugins.lhan.me/index.json').then(res => {
    info.value = res.data
    success.value = true
})

const parseGitHubUrl = (url) => {
  const regex = /^https?:\/\/github\.com\/([^\/]+)\/([^\/]+)/;
  const match = url.match(regex);
  if (match) {
    return {
      isGitHub: true,
      url: `https://img.shields.io/github/downloads/${match[1]}/${match[2]}/total?style=social`,
    };
  } else {
    return {
      isGitHub: false
    };
  }
}

</script>

# 可用插件列表

> 当前列举的插件全部来自[索引仓库](https://github.com/version-fox/vfox-plugins)

::: tip 提醒
这些都是来自社区的 vfox 插件

你可以通过一下命令来快速安装!

```shell
vfox add <name>
```
:::


<div :class="$style.layout_plugins" v-if="success">
<div v-for="item in info">
    <div :class="$style.card">
        <p style="display:flex;align-items: center;">
            <h5>
                <a :href="item.homepage" style="font-weight:bold">{{item.name}}</a>
            </h5>
            <img v-if="parseGitHubUrl(item.homepage).isGitHub" style="display:inline; margin-left:5px" :src="parseGitHubUrl(item.homepage).url"/>
        </p>
        <p :class="$style.desc">{{item.desc}}</p>
    </div>
</div>
</div>
<div v-else>正在加载中, 请耐心等候...</div>

<style module>
.layout_plugins {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 1rem;
}
.card {
    position: relative;
    border-radius: .5rem;
    border-width: 1px;
    border-bottom-width: 2px;
    border-style: solid;
    border-color: rgba(215, 223, 233, .75);
    background-color: rgb(242 244 248 / var(1));
    padding-left: 1rem;
    padding-right: 1rem;
    padding-bottom: 1rem;
    padding-top: 1rem;
}
.desc {
    font-weight: 400;
    font-size: 0.8rem;
    line-height: 0.5rem;
}
</style>
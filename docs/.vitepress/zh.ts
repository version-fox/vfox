/*
 *    Copyright 2024 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

import { defineConfig } from 'vitepress'

export const zh = defineConfig({
    lang: 'zh-Hans',
    description: '由 Vite 和 Vue 驱动的静态站点生成器',
    themeConfig: {
        // https://vitepress.dev/reference/default-theme-config
        nav: [
            { text: '首页', link: '/zh-hans' },
            { text: '例子', link: '/zh-hans/markdown-examples' }
        ],

        sidebar: [
            {
                text: '测试例子',
                items: [
                    { text: 'markdown例子', link: '/markdown-examples' },
                    { text: 'Runtime API Examples', link: '/api-examples' }
                ]
            }
        ],

        socialLinks: [
            { icon: 'github', link: 'https://github.com/version-fox/vfox' },
            { icon: 'discord', link: 'https://discord.com/invite/85c8ptYgb7' }
        ]
    }
})

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

import {DefaultTheme, defineConfig} from 'vitepress'

export const zh = defineConfig({
    lang: 'zh-Hans',
    description: '跨平台、可扩展的版本管理器',
    themeConfig: {
        nav: nav(),
        sidebar: sidebar(),
    }
})

function nav(): DefaultTheme.NavItem[] {
    return [
        {text: '首页', link: '/zh-hans'},
        {text: '文档', link: '/zh-hans/getting-started/intro'},
        {text: '插件仓库', link: 'https://github.com/version-fox/version-fox-plugins'}
    ]
}

function sidebar(): DefaultTheme.Sidebar {
    return [
        {
            text: '入门',
            items: [
                {text: '什么是vfox?', link: '/zh-hans/getting-started/intro'},
                {text: '快速入门', link: '/zh-hans/getting-started/quick-start'},
                {text: '配置', link: '/zh-hans/getting-started/configuration'},
                {text: '常见问题', link: '/zh-hans/getting-started/faq'},
            ]
        },
        {
            text: '用法',
            items: [
                {text: 'markdown例子', link: '/markdown-examples'},
                {text: 'Runtime API Examples', link: '/api-examples'}
            ]
        },
        {
            text: '插件',
            items: [
                {text: 'markdown例子', link: '/markdown-examples'},
                {text: 'Runtime API Examples', link: '/api-examples'}
            ]
        },
        {
            text: '其他',
            items: [
                {text: 'markdown例子', link: '/markdown-examples'},
                {text: 'Runtime API Examples', link: '/api-examples'}
            ]
        },
    ]
}
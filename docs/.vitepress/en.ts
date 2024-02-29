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

export const en= defineConfig({
    lang: 'en',
    description: 'The Multiple SDK Version Manager',
    themeConfig: {
        nav: nav(),
        sidebar: sidebar(),
    }
})

function nav(): DefaultTheme.NavItem[] {
    return [
        {text: 'Home', link: '/'},
        {text: 'Documentation', link: '/getting-started/intro'},
        {text: 'Plugins', link: 'https://github.com/version-fox/version-fox-plugins'}
    ]
}

function sidebar(): DefaultTheme.Sidebar {
    return [
        {
            text: 'Guide',
            items: [
                {text: 'What is vfox?', link: '/guides/intro'},
                {text: 'Quick Start', link: '/guides/quick-start'},
                {text: 'Configuration', link: '/guides/configuration'},
                {text: 'FAQ', link: '/guides/faq'},
            ]
        },
        {
            text: 'Usage',
            items: [
                {text: 'Core', link: '/usage/core-commands'},
                {text: 'All Commands', link: '/usage/all-commands'},
            ]
        },
        {
            text: 'Plugins',
            items: [
                {
                    text: 'Authors',
                    items:[
                        {text: 'Create a Plugin', link: '/plugins/create/howto'},
                        {text: 'Plugin Template', link: 'https://github.com/version-fox/vfox/blob/main/template.lua'},
                    ]
                },
                {
                    text: 'Library',
                    items:[
                        {text: 'http', link: '/plugins/library/http'},
                        {text: 'html', link: '/plugins/library/html'},
                        {text: 'json', link: '/plugins/library/json'},
                    ]
                },

                {text: 'Available Plugins', link: 'https://github.com/version-fox/version-fox-plugins'},
            ]
        },
        {
            text: 'Misc',
            items: [
                {text: 'Comparison to asdf', link: '/misc/vs-asdf'},
            ]
        },
    ]
}
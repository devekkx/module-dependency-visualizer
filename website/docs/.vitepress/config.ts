import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'mdv',
  description: 'Language-agnostic module dependency visualizer for Go, Node, and Python projects.',
  head: [['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }]],

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'mdv',

    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'Commands', link: '/commands' },
      { text: 'Schema', link: '/schema' },
      {
        text: 'v0.5.0',
        items: [
          { text: 'Changelog', link: 'https://github.com/devekkx/module-dependency-visualizer/releases' },
          { text: 'Contributing', link: '/contributing' },
        ],
      },
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is mdv?', link: '/' },
          { text: 'Getting Started', link: '/getting-started' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Commands', link: '/commands' },
          { text: 'Output Formats', link: '/formats' },
          { text: 'JSON Schema', link: '/schema' },
          { text: 'GitHub Actions', link: '/github-actions' },
        ],
      },
      {
        text: 'Contributing',
        items: [
          { text: 'Contributing Guide', link: '/contributing' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/devekkx/module-dependency-visualizer' },
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 Emmanuel Kpendo',
    },

    search: {
      provider: 'local',
    },

    editLink: {
      pattern: 'https://github.com/devekkx/module-dependency-visualizer/edit/main/website/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
})

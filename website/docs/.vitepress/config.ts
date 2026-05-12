import { execSync } from 'child_process'
import { defineConfig } from 'vitepress'

function latestTag(): string {
  try {
    return execSync('git describe --tags --abbrev=0', { encoding: 'utf8' }).trim()
  } catch {
    return ''
  }
}

const version = latestTag()

export default defineConfig({
  title: 'mdv',
  description: 'Language-agnostic module dependency visualizer for Go, Node, and Python projects.',
  head: [['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }]],

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'mdv',

    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'Languages', link: '/languages' },
      { text: 'Commands', link: '/commands' },
      { text: 'Schema', link: '/schema' },
      ...(version
        ? [{
            text: version,
            items: [
              { text: 'Changelog', link: 'https://github.com/devekkx/module-dependency-visualizer/releases' },
              { text: 'Contributing', link: '/contributing' },
            ],
          }]
        : [
            { text: 'Changelog', link: 'https://github.com/devekkx/module-dependency-visualizer/releases' },
            { text: 'Contributing', link: '/contributing' },
          ]
      ),
    ],

    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'What is mdv?', link: '/' },
          { text: 'Getting Started', link: '/getting-started' },
          { text: 'Supported Languages', link: '/languages' },
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
      copyright: 'Copyright © 2026 Emmanuel Komla Kpendo',
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

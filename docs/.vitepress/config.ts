import { defineConfig } from 'vitepress'
import { useSidebar } from 'vitepress-openapi'

import spec from '../../openapi/external.openapi.json'

// One sidebar group per OpenAPI tag, one entry per operation, linking to /api/<operationId>.
const apiItems = useSidebar({ spec, linkPrefix: '/api/' }).generateSidebarGroups()

export default defineConfig({
  title: '1mail',
  description: 'Open-core marketing automation you can run yourself',
  base: '/1mail/',
  cleanUrls: true,
  lastUpdated: true,
  // Internal working notes (agent setup, backlog, ADRs, research) are not part of the public site.
  srcExclude: ['agents/**', 'grill-backlog.md', 'adr/**', 'research/**'],
  ignoreDeadLinks: [/^http:\/\/localhost/],
  // The docs quote Liquid templates (`{{ ... }}`); keep Vue from interpolating them.
  vue: { template: { compilerOptions: { delimiters: ['[[vue:', ']]'] } } },
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/introduction' },
      { text: 'API reference', link: '/api/' },
      { text: 'Self-hosting', link: '/self-hosting' },
      { text: 'Roadmap', link: '/ROADMAP' },
    ],
    sidebar: {
      '/api/': [{ text: 'API reference', link: '/api/' }, ...apiItems],
      '/': [
        {
          text: 'Guide',
          items: [
            { text: 'Introduction', link: '/guide/introduction' },
            { text: 'Quickstart', link: '/guide/quickstart' },
            { text: 'Tracking visitors and events', link: '/guide/tracking' },
            { text: 'Segments', link: '/guide/segments' },
            { text: 'Sending email', link: '/guide/sending' },
            { text: 'Deliverability and consent', link: '/guide/deliverability' },
            { text: 'API', link: '/guide/api' },
            { text: 'Webhooks', link: '/guide/webhooks' },
            { text: 'MCP for agents', link: '/guide/mcp' },
            { text: 'Self-hosting', link: '/self-hosting' },
          ],
        },
        {
          text: 'Project',
          items: [
            { text: 'Roadmap', link: '/ROADMAP' },
            { text: 'Library choices', link: '/TECH-CHOICES' },
            { text: 'Domain events', link: '/design/domain-events' },
          ],
        },
      ],
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/mokevnin/1mail' }],
    editLink: {
      pattern: 'https://github.com/mokevnin/1mail/edit/main/docs/:path',
    },
    search: { provider: 'local' },
  },
})

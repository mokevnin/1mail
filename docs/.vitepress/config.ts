import { readdirSync, readFileSync } from 'node:fs'

import { defineConfig } from 'vitepress'

// One sidebar entry per ADR: the number plus the title up to its first colon.
const adrDir = new URL('../adr/', import.meta.url)
const adrItems = readdirSync(adrDir)
  .filter((file) => file.endsWith('.md'))
  .toSorted()
  .map((file) => {
    const slug = file.replace(/\.md$/, '')
    const heading = readFileSync(new URL(file, adrDir), 'utf8').match(/^# (.+)$/m)?.[1] ?? slug
    return { text: `${slug.slice(0, 4)} · ${heading.split(':')[0]}`, link: `/adr/${slug}` }
  })

export default defineConfig({
  title: '1mail',
  description: 'Open-core marketing automation you can run yourself',
  base: '/1mail/',
  cleanUrls: true,
  lastUpdated: true,
  // Internal working notes (agent setup, backlog) are not part of the public site.
  srcExclude: ['agents/**', 'grill-backlog.md'],
  ignoreDeadLinks: [/^http:\/\/localhost/],
  // The docs quote Liquid templates (`{{ ... }}`); keep Vue from interpolating them.
  vue: { template: { compilerOptions: { delimiters: ['[[vue:', ']]'] } } },
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/introduction' },
      { text: 'Self-hosting', link: '/self-hosting' },
      { text: 'Architecture', link: '/adr/0001-send-eligibility-model' },
      { text: 'Roadmap', link: '/ROADMAP' },
    ],
    sidebar: [
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
          { text: 'Self-hosting', link: '/self-hosting' },
        ],
      },
      {
        text: 'Operations',
        items: [
          { text: 'Overview', link: '/operations/' },
          { text: 'Backup and restore', link: '/operations/backup' },
          { text: 'Upgrading', link: '/operations/upgrading' },
          { text: 'Scaling and tuning', link: '/operations/scaling' },
          { text: 'Security hardening', link: '/operations/security' },
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
      {
        text: 'Research',
        items: [
          { text: 'Drip feature analysis', link: '/research/drip-com-feature-analysis' },
          { text: 'Mautic pain points', link: '/research/mautic-pain-points-analysis' },
          { text: 'Keila feature analysis', link: '/research/keila-feature-analysis' },
          { text: 'Outbound send prior art', link: '/research/outbound-send-prior-art' },
          { text: 'Quality tooling options', link: '/research/quality-tooling-options' },
        ],
      },
      {
        text: 'Architecture decisions',
        collapsed: false,
        items: adrItems,
      },
    ],
    socialLinks: [{ icon: 'github', link: 'https://github.com/mokevnin/1mail' }],
    editLink: {
      pattern: 'https://github.com/mokevnin/1mail/edit/main/docs/:path',
    },
    search: { provider: 'local' },
  },
})

import { defineConfig } from 'vitepress'

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
      { text: 'Self-hosting', link: '/self-hosting' },
      { text: 'Architecture', link: '/adr/0001-send-eligibility-model' },
      { text: 'Roadmap', link: '/ROADMAP' },
    ],
    sidebar: [
      {
        text: 'Guide',
        items: [{ text: 'Self-hosting', link: '/self-hosting' }],
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
        items: [
          {
            text: '0001 \u00b7 Send-eligibility: (channel, destination)-keyed Suppression + scoped Un',
            link: '/adr/0001-send-eligibility-model',
          },
          {
            text: '0002 \u00b7 One Contact identity: absorb the tracking profile, attach events by st',
            link: '/adr/0002-unified-contact-identity',
          },
          {
            text: '0003 \u00b7 Templates are copied at author time, never referenced (marketing)',
            link: '/adr/0003-templates-copied-not-referenced',
          },
          {
            text: '0004 \u00b7 Workspaces are multi-user via Membership, not single-owner',
            link: '/adr/0004-multi-user-workspaces-via-membership',
          },
          {
            text: '0005 \u00b7 Transactional is a first-class send surface, binding templates by refe',
            link: '/adr/0005-transactional-send-surface',
          },
          {
            text: '0006 \u00b7 One attribute concept: typed Custom fields, auto-created \u2014 no schemale',
            link: '/adr/0006-custom-fields-not-traits',
          },
          {
            text: '0007 \u00b7 Workspace suspension: mechanism in core, policy and console in EE',
            link: '/adr/0007-workspace-suspension-in-core',
          },
          {
            text: '0008 \u00b7 Platform Operator is a separate identity, not a User',
            link: '/adr/0008-operator-separate-identity',
          },
          {
            text: '0009 \u00b7 Billing boundary: metering in core, money in an external plane',
            link: '/adr/0009-billing-boundary-metering-in-core-money-outside',
          },
          {
            text: '0010 \u00b7 Sending domains: 1mail-native DKIM, verified-domain required to send',
            link: '/adr/0010-sending-domains-native-dkim',
          },
          {
            text: '0011 \u00b7 Deliverability rate metrics: complaint & bounce rate in core, threshol',
            link: '/adr/0011-deliverability-rate-metrics',
          },
          {
            text: '0012 \u00b7 Bulk-sender compliance: RFC 8058 one-click unsubscribe & DMARC readine',
            link: '/adr/0012-bulk-sender-compliance-one-click-unsubscribe',
          },
          {
            text: '0013 \u00b7 Double opt-in: confirmation as a positive event, not a subscription st',
            link: '/adr/0013-double-opt-in-confirmation',
          },
          {
            text: '0014 \u00b7 Open-core boundary: gate on org-shape, not product value',
            link: '/adr/0014-open-core-boundary',
          },
          {
            text: '0015 \u00b7 Outbound send: one module, one message record, two outcome scopes',
            link: '/adr/0015-outbound-send-single-chokepoint',
          },
          {
            text: '0016 \u00b7 MCP surface: a projection of the external `/api`, not a second impleme',
            link: '/adr/0016-mcp-surface-projection-of-external-api',
          },
        ],
      },
    ],
    socialLinks: [{ icon: 'github', link: 'https://github.com/mokevnin/1mail' }],
    editLink: {
      pattern: 'https://github.com/mokevnin/1mail/edit/main/docs/:path',
    },
    search: { provider: 'local' },
  },
})

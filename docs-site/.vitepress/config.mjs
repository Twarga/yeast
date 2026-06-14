import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Yeast',
  description: 'Turn a folder into real VMs.',
  ignoreDeadLinks: true,
  
  head: [
    ['link', { rel: 'icon', href: '/yeast/favicon.ico' }],
  ],

  base: '/yeast/docs/',

  themeConfig: {
    logo: '/logo.svg',
    siteTitle: 'Yeast',
    
    nav: [
      { text: 'Docs', link: '/docs/intro' },
      { text: 'Tutorials', link: '/tutorials/' },
      { text: 'Reference', link: '/docs/configuration' },
      { 
        text: 'Releases', 
        link: 'https://github.com/Twarga/yeast/releases' 
      },
    ],

    sidebar: {
      '/docs/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Introduction', link: '/docs/intro' },
            { text: 'Installation', link: '/docs/installation' },
            { text: 'Quickstart', link: '/docs/quickstart' },
            { text: 'Release Smoke Test', link: '/docs/release-smoke-v1.1.0' },
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Architecture', link: '/docs/architecture' },
            { text: 'Configuration', link: '/docs/configuration' },
            { text: 'Networking', link: '/docs/networking' },
            { text: 'Snapshots', link: '/docs/snapshots' },
            { text: 'Provisioning', link: '/docs/provisioning' },
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'Commands', link: '/docs/commands' },
            { text: 'Known Limitations', link: '/docs/known-limitations' },
            { text: 'Troubleshooting', link: '/docs/troubleshooting' },
          ]
        }
      ],
      '/tutorials/': [
        {
          text: 'Tutorials',
          items: [
            { text: 'Overview', link: '/tutorials/' },
          ]
        },
        {
          text: 'Fundamentals',
          items: [
            { text: '01 - First VM', link: '/tutorials/01-first-vm' },
            { text: '02 - Provisioning', link: '/tutorials/02-provisioning' },
            { text: '03 - Snapshots', link: '/tutorials/03-snapshots' },
            { text: '04 - Multi-VM Lab', link: '/tutorials/04-multi-vm-lab' },
          ]
        },
        {
          text: 'Advanced',
          items: [
            { text: '05 - Guest Control', link: '/tutorials/05-guest-control' },
            { text: '06 - LabsBackery Lab', link: '/tutorials/06-labsbackery-lab' },
            { text: '07 - Templates', link: '/tutorials/07-templates' },
            { text: '08 - JSON Automation', link: '/tutorials/08-json-automation' },
          ]
        },
        {
          text: 'Project Labs',
          items: [
            { text: '09 - Nodi Home Lab', link: '/tutorials/09-nodi-home-lab' },
            { text: '10 - Load Balancer', link: '/tutorials/10-load-balancer-lab' },
            { text: '11 - Database + App', link: '/tutorials/11-database-app-stack' },
            { text: '12 - Monitoring Stack', link: '/tutorials/12-monitoring-stack' },
            { text: '13 - WireGuard VPN', link: '/tutorials/13-wireguard-vpn-mesh' },
            { text: '14 - GitOps/CI', link: '/tutorials/14-gitops-ci-lab' },
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Twarga/yeast' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 TwargaOps'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/Twarga/yeast/edit/main/docs-site/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})

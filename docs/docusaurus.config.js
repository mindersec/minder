// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0


// Package apply provides the apply command for the medctl CLI
// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github')
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

const redocusaurus = [
  'redocusaurus',
  {
    specs: [
      {
        id: 'minder-api',
        spec: '../pkg/api/openapi/minder/v1/minder.swagger.json',
      },
    ],
    theme: {
      primaryColor: '#000000',
      primaryColorDark: '#b0e0e6',
    },
  }
]

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Minder docs',
  tagline: 'Documentation site for Minder',
  favicon: 'img/stacklok-favicon.png',
  themes: ['@docusaurus/theme-mermaid', 'docusaurus-theme-redoc'],

  // Set the production url of your site here
  url: 'https://minder-docs.stacklok.dev/',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'stacklok', // Usually your GitHub org/user name.
  projectName: 'minder', // Usually your repo name.
  trailingSlash: false,
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      {
        docs: {
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
    redocusaurus,
  ],

  plugins: [
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          {
            /* Trusty rebrand */
            to: '/integrations/stacklok-insight',
            from: '/integrations/trusty',
          },
        ],
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    (
      {
        colorMode: {
          defaultMode: 'dark',
          disableSwitch: false,
          respectPrefersColorScheme: false,
        },        
      // Replace with your project's social card
      image: 'img/Minder_darkMode.png',
      navbar: {
        title: 'Minder docs',
        logo: {
          alt: 'Minder Logo',
          src: 'img/Minder-whitetxt.svg',
        },
        items: [
          // {
          //   type: 'html',
          //   position: 'right',
          //   value: 'Minder version:',
          // },   
          // {
          //   type: 'docsVersionDropdown',
          //   position: 'right',
          // },
          {
            href: 'https://github.com/mindersec/minder',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Minder',
                to: '/',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/mindersec/minder',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Minder a Series of LF Projects, LLC
For web site terms of use, trademark policy and other project policies please see https://lfprojects.org.. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;

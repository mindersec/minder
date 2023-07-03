//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
        id: 'mediator-api',
        path: '../',
        spec: './pkg/generated/openapi/mediator/v1/mediator.swagger.json',
        route: '/api',
      },
    ],
  }
]

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Mediator docs',
  tagline: 'Documentation site for Mediator',
  favicon: 'img/stacklok-favicon.png',
  themes: ['docusaurus-theme-redoc'],

  // Set the production url of your site here
  url: 'https://stacklok.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'stacklok', // Usually your GitHub org/user name.
  projectName: 'mediator', // Usually your repo name.
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

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          routeBasePath: '/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
    redocusaurus,
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
      image: 'img/stacklok-logo.svg',
      navbar: {
        title: 'Mediator docs',
        logo: {
          alt: 'Stacklok Logo',
          src: 'img/stacklok-logo.svg',
        },
        items: [
          {
            type: 'html',
            position: 'right',
            value: 'Mediator version:',
          },   
          {
            type: 'docsVersionDropdown',
            position: 'right',
          },
          {
            href: 'https://github.com/stacklok/mediator',
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
                label: 'Mediator',
                to: '/',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'Website',
                href: 'https://stacklok.com',
              },
              {
                label: 'Twitter',
                href: 'https://twitter.com/StackLokHQ',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'Blog',
                to: 'https://www.stacklok.com/blog',
              },
              {
                label: 'GitHub',
                href: 'https://github.com/stacklok/mediator',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Stacklok, Inc. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;

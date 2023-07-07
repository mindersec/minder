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

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check
const fs = require('fs');

const folderPath = 'docs/cli/'; // Path to the folder containing the markdown files
const sidebarItems = fs
  .readdirSync(folderPath)
  .filter((file) => file.endsWith('.md'));

const sidebar = [];
sidebarItems.forEach((file) => {
  sidebar.push({
    type: 'doc',
    label: file.replace('.md', '').replace(/_/g, ' '),
    id: 'cli/' + file.replace('.md', ''),
  });
});

const sidebars = {
  // By default, Docusaurus generates a sidebar from the docs folder structure
  mediator: [
    {
      type: 'doc',
      label: 'Mediator introduction',
      id: 'mediator_intro',
    },
    {
      type: 'doc',
      label: 'Getting Started (Run the Server)',
      id: 'get_started',
    },
    {
      type: 'doc',
      label: 'Getting Started (Configure OAuth Provider)',
      id: 'config_oauth',
    },
    {
      type: 'doc',
      label: 'Getting Started (Enroll User & Register Repositories)',
      id: 'enroll_user',
    },
    {
      type: 'doc',
      label: 'Developer Guide',
      id: 'get_hacking',
    },
    {
      type: 'link',
      label: 'API documentation',
      href: '/api',
    },
    {
      type: 'doc',
      label: 'Proto documentation',
      id: 'protodocs/proto',
    },
    {
      type: 'category',
      label: 'Mediator client documentation', 
      items:   sidebar
    },
    {
      type: 'doc',
      label: 'DB schema',
      id: 'db/mediator_db_schema',
    },
  ],
};

module.exports = sidebars;

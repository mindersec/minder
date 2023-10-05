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

const folderCLIPath = 'docs/cli/'; // Path to the folder containing the markdown files
const sidebarCLIItems = fs
  .readdirSync(folderCLIPath)
  .filter((file) => file.endsWith('.md'));

const sidebarCLI = [];
sidebarCLIItems.forEach((file) => {
  sidebarCLI.push({
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
      label: 'Introduction',
      id: 'mediator_intro',
    },
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting_started/login_medic',
        'getting_started/register_repos',
      ],
    },
    {
      type: 'category',
      label: 'Profile Engine',
      items: [
        'profile_engine/profile_introduction',
        'profile_engine/manage_profiles',
      ],
    },
    {
      type: 'category',
      label: 'Run a Mediator Server',
      items: [
        'run_mediator_server/run_the_server',
        'run_mediator_server/config_oauth',
      ],
    },
    {
      type: 'category',
      label: 'Developer Guide',
      items: [
        'developer_guide/get_hacking',
      ],
    },
    {
      type: 'doc',
      label: 'Architecture',
      id: 'mediator_architecture',
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
      items:   sidebarCLI
    },
    {
      type: 'doc',
      label: 'DB schema',
      id: 'db/mediator_db_schema',
    },
  ],
};

module.exports = sidebars;

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
      type: 'link',
      label: 'API documentation',
      href: '/mediator/api',
    },
    {
      type: 'doc',
      label: 'Proto documentation',
      id: 'proto/mediator/v1/proto',
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

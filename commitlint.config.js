//
//  Copyright 2023 Stacklok, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

module.exports = {

    // This config should help enforce Chris Bean's commit message recommendations

    rules: {

        // Separate subject from body with a blank line
        'body-leading-blank': [2, 'always'],

        // - Limit the subject line to 50 characters
        'header-max-length': [2, 'always', 50],

        // Capitalize the subject line
        'header-case': [2, 'always', 'sentence-case'],
        'subject-case': [2, 'always', 'sentence-case'],

        // Do not end the subject line with a period
        'header-full-stop': [2, 'never'],
        'subject-full-stop': [0, 'never'],

        // Wrap the body at 72 characters
        'body-max-line-length': [2, 'always', 75],

        'type-empty': [2, 'always'], // Disallow types

        // TODO: write plugins to check for:
        // Use the imperative mood in the subject line
        // Use the body to explain what and why vs. how
    },
    ignores: [
        // Ignores Dependabot and WIP commits
        (message) => message.includes('build(') || message.includes('WIP')
    ]
};

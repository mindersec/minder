# Style guide for Minder docs <!-- omit in toc -->

This style guide is a reference for anyone who contributes to the user-facing
Minder docs contained in this directory. By adhering to these guidelines, we aim
to deliver clear, concise, and valuable information to Minder users.

## Contents <!-- omit in toc -->

- [Writing style](#writing-style)
  - [Language](#language)
  - [Tone and voice](#tone-and-voice)
    - [Active voice](#active-voice)
    - [Speak to the reader](#speak-to-the-reader)
  - [Capitalization](#capitalization)
  - [Punctuation](#punctuation)
  - [Links](#links)
  - [Formatting](#formatting)
- [Markdown style](#markdown-style)
- [Word list \& glossary](#word-list--glossary)
  - [Minder technical terms](#minder-technical-terms)
  - [Products/brands](#productsbrands)

## Writing style

This list is not exhaustive, it is intended to reflect the most common and
important style elements. For a more comprehensive guide that aligns with our
style goals, or if you need more details about any of these points, refer to the
[Google developer documentation style guide](https://developers.google.com/style).

### Language

The project's official language is **US English**.

### Tone and voice

Strive for a casual and conversational tone without becoming overly informal. We
aim to be friendly and relatable while retaining credibility and professionalism
– approachable yet polished.

Avoid slang and colloquial expressions. Use clear, straightforward language and
avoid overly complex jargon to make content accessible to a wide audience.

#### Active voice

Use **active voice** instead of passive voice. Active voice emphasizes the
subject performing the action, making the writing more direct and engaging.
Passive voice focuses on the recipient of the action rather than the actor,
often resulting in unclear sentences and misinterpretation of responsibility.

:white_check_mark: Yes: Click the **Add** button to add the rule.\
:x: No: The rule is added when the "Add" button is clicked.

:white_check_mark: Yes: Set the `debug` flag to `true` to enable verbose
logging.\
:x: No: Verbose logging is enabled when the `debug` flag is set to `true`.

#### Speak to the reader

Address the reader using the **second person** ("you", "your"). Avoid the first
person ("we", "our") and third person ("the user", "a developer").

### Capitalization

Capitalize **proper nouns** like names, companies, and products. Generally,
**don’t** capitalize features or generic terms. For non-Minder terms, follow the
norms of the third-party project/company (ex: npm is stylized in lowercase, even
when it begins a sentence).

:white_check_mark: Yes: Minder profiles are a collection of rule types that are
applied to entities.\
:x: No: Minder Profiles are a collection of Rule Types that are applied to
Entities.

Use **sentence case** in titles and headings.

:white_check_mark: Yes: Policy and profile management\
:x: No: Policy and Profile Management

Use ALL CAPS to indicate placeholder text, where the reader is expected to
change a value.

### Punctuation

**Oxford comma**: use the Oxford comma (aka serial commas) when listing items in
a series.

:white_check_mark: Yes: Minder acts on repositories, pull requests, and
artifacts.\
:x: No: Minder acts on repositories, pull requests and artifacts.

**Quotation marks**: in technical documentation, use straight double quotes and
apostrophes, not "fancy quotes" or "smart quotes" (the default in document
editors like Word/Docs). This is especially important in code examples where
smart quotes often cause syntax errors.

Tip: when drafting technical docs in Google Docs, disable the "Use smart quotes"
setting in the Tools → Preferences menu to avoid inadvertently copying smart
quotes into Markdown or other code.

### Links

Use descriptive link text. Besides providing clear context to the reader, this
improves accessibility for screen readers.

:white_check_mark: Yes: For more information, see
[Purpose and scope](?tab=t.0#heading=h.qaqvuha5efk).\
:x: No: For more information, see
[this section](?tab=t.0#heading=h.qaqvuha5efk).

Note on capitalization: when referencing other docs/headings by title, use
sentence case so the reference matches the corresponding title or heading.

### Formatting

**Bold**: use when referring to UI elements; prefer bold over quotes. For
example: Click **Add Rule** and select the rule you want to add to the profile.

**Italics**: emphasize particular words or phrases, such as when
introducing/defining a term. For example: A _profile_ defines which security
policies apply to your software supply chain.

**Underscore**: do not use; reserved for links.

**Code**: use a `monospaced font` for inline code or commands, code blocks, user
input, filenames, method/class names, and console output.

## Markdown style

Just like a consistent writing style is critical to clarity and messaging,
consistent formatting and syntax are needed to ensure the maintainability of
Markdown-based documentation.

We adopt the
[Google Markdown style guide](https://google.github.io/styleguide/docguide/style.html),
which is well-aligned with default settings in formatting tools like Prettier
and `markdownlint`.

Our preferred style elements include:

- Headings: use "ATX-style" headings (hash marks - `#` for Heading 1, `##` for
  Heading 2, and so on); use unique headings within a document
- Unordered lists: use hyphens (`-`), not asterisks (`*`)
- Ordered lists: use lazy numbering (`1.` for every item and let Markdown render
  the final order – this is more maintainable when inserting new items)
  - Note: this is a "soft" recommendation. It is also intended only for Markdown
    documents that are read through a rendering engine. If the Markdown will be
    consumed in raw form, use real numbering.
- Code blocks: use fenced code blocks (` ``` ` to begin/end) and explicitly
  declare the language
- Add blank lines around headings, lists, and code blocks
- No trailing whitespace on lines
  - Use the `\` character at the end of a line for a single-line break, not the
    two-space syntax which is easy to miss
- Line limit: wrap lines at 80 characters; exceptions for links, tables,
  headings, and code blocks

Specific guidelines for Docusaurus:

- Heading 1 is reserved for the page title, typically defined in the Markdown
  front matter section. Sections within a page begin with Heading 2 (`##`).
  [Reference](https://docusaurus.io/docs/markdown-features/toc)
- Use relative file links (with .md/.mdx extensions) when referring to other
  pages. [Reference](https://docusaurus.io/docs/markdown-features/links)
- Use the .mdx extension for pages containing JSX includes. Docusaurus v3
  currently runs all .md and .mdx files through an MDX parser but this will
  change in a future version.
  [Reference](https://docusaurus.io/docs/migration/v3#using-the-mdx-extension)
- Use the front matter section on all pages. At a minimum, set the `title` (this
  is rendered into the page as an H1) and a short `description`.
  [Reference](https://docusaurus.io/docs/api/plugins/@docusaurus/plugin-content-docs#markdown-front-matter)

## Word list & glossary

Common terms used in Minder content:

**open source**: we prefer using two words over the hyphenated form (not
"open-source"). It's not a proper noun, so don't capitalize unless it starts a
sentence.

**OSS**: abbreviation for "open source software"

### Minder technical terms

See also: [Key concepts](https://mindersec.github.io/understand/key_concepts)

**alert**: a rule evaluation that has a status of _failure_ or _error_; this is
something that should be shown to the user for a problem to resolve.

**entity**: a resource that is registered with Minder and which can be targeted
by profiles. Currently, these can be repositories, pull requests, and artifacts.

**profile**: a collection of rules that are applied to entities that match the
associated **profile selector**(s).

**project**: the unit of tenancy in Minder.

**provider**: a plugin that allows Minder to interface with an external system
like GitHub or GitLab.

**repository**: a Git repository is the place where your source code is kept,
which is usually hosted on a "forge" like GitHub, GitLab, Bitbucket, etc. The
term "repository" is preferred over the shorthand "repo" in documentation, which
matches the preferred style for GitHub and the Git project itself.

- **register a repository**: the mechanism for telling a Minder project that it
  should monitor a particular GitHub (and soon GitLab) repository.
- **repo**: should only be used where horizontal space is at a premium. In
  particular, it can be used in APIs and URLs.

**rule type**: defines an individual check for a specific aspect of an entity

- **rule evaluation -** the result of the evaluation of a rule in a profile
  within a profile. Rule evaluations _may_ be alerts, or they may simply be a
  successful result when the evaluation target is in the desired state.

### Products/brands

**Bitbucket**: Atlassian’s source code hosting and CI/CD tool which integrates
nicely with Jira. It’s written with only a leading capital as one word (not
"BitBucket").

**Git**: the most popular distributed version control system. It underpins most
commercial VCS offerings like GitHub, Bitbucket, and GitLab. Unless specifically
referring to the `git` command line tool, it's a proper noun and should be
capitalized.

**GitHub**: the most popular source code hosting provider, especially for open
source. It’s written bi-capitalized as one word (not "Git Hub" or "Github").

**GitHub Actions**: GitHub’s CI/CD system. "Actions" is capitalized when used to
refer to the service/system along with the GitHub name, but not when referring
to individual "actions".

- **action**: a reusable piece of code that can be called from a GitHub Actions
  workflow, for example `actions/checkout`; note that this is not capitalized.
- **workflow**: a YAML-described set of steps that will run on a trigger, for
  example, when a pull request is opened in a GitHub repository or on a
  schedule. A workflow may be run on a GitHub-hosted _runner_, or on a machine
  that is provided by the repository owner.
- **workflow run**: a single instance of an execution of a GitHub Actions
  workflow that is executed on a _runner_. A workflow run _may_ produce an
  artifact; for example, a CI/CD workflow may create a container and publish it
  to a container registry.

**GitHub Advanced Security**: often abbreviated "GHAS", which is pronounced like
"gas". A subscription service available to GitHub Enterprise customers that
entitles them to several security features like CodeQL and secret scanning. Some
or all of the functionality in GitHub Advanced Security is also available to
public repositories (therefore, available to open source projects) but it is not
strictly correct to say that "open source gets GHAS".

**GitLab**: another popular source code hosting provider focused especially on
on-premises installations. It’s written bi-capitalized as one word (not "Git
Lab" or "Gitlab").

**npm**: the registry for JavaScript packages (the "npm registry"), and the
default package manager for JavaScript. Since it’s both the registry _and_ the
package manager, it may be useful to disambiguate "the npm registry". It’s not
an abbreviation, so it’s not capitalized; it’s written all lowercase (not
"NPM").

**sigstore**: the package/artifact signing and verification technologies that we
believe are the best way to represent trusted provenance; note that the sigstore
project frequently intermixes "Sigstore" and "sigstore" but we prefer lowercase.

**Visual Studio Code**: a very popular free integrated development environment
(IDE) from Microsoft. Per Microsoft's
[brand guidelines](https://code.visualstudio.com/brand#brand-name), use the full
"Visual Studio Code" name the first time you reference it. "VS Code" is an
acceptable short form after the first reference. It's written as two words and
there are no other abbreviations/acronyms (not "VSCode", "VSC", or just "Code").

<!-- markdownlint-disable-file MD044 -->

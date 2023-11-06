"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[9767],{3905:(e,t,i)=>{i.d(t,{Zo:()=>p,kt:()=>h});var n=i(67294);function l(e,t,i){return t in e?Object.defineProperty(e,t,{value:i,enumerable:!0,configurable:!0,writable:!0}):e[t]=i,e}function o(e,t){var i=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),i.push.apply(i,n)}return i}function a(e){for(var t=1;t<arguments.length;t++){var i=null!=arguments[t]?arguments[t]:{};t%2?o(Object(i),!0).forEach((function(t){l(e,t,i[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(i)):o(Object(i)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(i,t))}))}return e}function r(e,t){if(null==e)return{};var i,n,l=function(e,t){if(null==e)return{};var i,n,l={},o=Object.keys(e);for(n=0;n<o.length;n++)i=o[n],t.indexOf(i)>=0||(l[i]=e[i]);return l}(e,t);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);for(n=0;n<o.length;n++)i=o[n],t.indexOf(i)>=0||Object.prototype.propertyIsEnumerable.call(e,i)&&(l[i]=e[i])}return l}var s=n.createContext({}),u=function(e){var t=n.useContext(s),i=t;return e&&(i="function"==typeof e?e(t):a(a({},t),e)),i},p=function(e){var t=u(e.components);return n.createElement(s.Provider,{value:t},e.children)},d="mdxType",c={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},k=n.forwardRef((function(e,t){var i=e.components,l=e.mdxType,o=e.originalType,s=e.parentName,p=r(e,["components","mdxType","originalType","parentName"]),d=u(i),k=l,h=d["".concat(s,".").concat(k)]||d[k]||c[k]||o;return i?n.createElement(h,a(a({ref:t},p),{},{components:i})):n.createElement(h,a({ref:t},p))}));function h(e,t){var i=arguments,l=t&&t.mdxType;if("string"==typeof e||l){var o=i.length,a=new Array(o);a[0]=k;var r={};for(var s in t)hasOwnProperty.call(t,s)&&(r[s]=t[s]);r.originalType=e,r[d]="string"==typeof e?e:l,a[1]=r;for(var u=2;u<o;u++)a[u]=i[u];return n.createElement.apply(null,a)}return n.createElement.apply(null,i)}k.displayName="MDXCreateElement"},56583:(e,t,i)=>{i.r(t),i.d(t,{assets:()=>s,contentTitle:()=>a,default:()=>c,frontMatter:()=>o,metadata:()=>r,toc:()=>u});var n=i(87462),l=(i(67294),i(3905));const o={title:"GitHub Actions",sidebar_position:70},a="GitHub Actions Configuration Policy",r={unversionedId:"ref/policies/github_actions",id:"ref/policies/github_actions",title:"GitHub Actions",description:"There are several rule types that can be used to configure GitHub Actions.",source:"@site/docs/ref/policies/github_actions.md",sourceDirName:"ref/policies",slug:"/ref/policies/github_actions",permalink:"/ref/policies/github_actions",draft:!1,tags:[],version:"current",sidebarPosition:70,frontMatter:{title:"GitHub Actions",sidebar_position:70},sidebar:"minder",previous:{title:"Known Vulnerabilities",permalink:"/ref/policies/vulnerabilities"},next:{title:"Presence of a LICENSE file",permalink:"/ref/policies/license"}},s={},u=[{value:"<code>github_actions_allowed</code> - Which actions are allowed to be used",id:"github_actions_allowed---which-actions-are-allowed-to-be-used",level:2},{value:"Entity",id:"entity",level:3},{value:"Type",id:"type",level:3},{value:"Rule parameters",id:"rule-parameters",level:3},{value:"Rule definition options",id:"rule-definition-options",level:3},{value:"<code>allowed_selected_actions</code> - Verifies that only allowed actions are used",id:"allowed_selected_actions---verifies-that-only-allowed-actions-are-used",level:2},{value:"Entity",id:"entity-1",level:3},{value:"Type",id:"type-1",level:3},{value:"Rule parameters",id:"rule-parameters-1",level:3},{value:"Rule definition options",id:"rule-definition-options-1",level:3},{value:"<code>default_workflow_permissions</code> - Sets the default permissions granted to the <code>GITHUB_TOKEN</code> when running workflows",id:"default_workflow_permissions---sets-the-default-permissions-granted-to-the-github_token-when-running-workflows",level:2},{value:"Entity",id:"entity-2",level:3},{value:"Type",id:"type-2",level:3},{value:"Rule parameters",id:"rule-parameters-2",level:3},{value:"Rule definition options",id:"rule-definition-options-2",level:3},{value:"<code>actions_check_pinned_tags</code> - Verifies that any actions use pinned tags",id:"actions_check_pinned_tags---verifies-that-any-actions-use-pinned-tags",level:2},{value:"Entity",id:"entity-3",level:3},{value:"Type",id:"type-3",level:3},{value:"Rule parameters",id:"rule-parameters-3",level:3},{value:"Rule definition options",id:"rule-definition-options-3",level:3}],p={toc:u},d="wrapper";function c(e){let{components:t,...i}=e;return(0,l.kt)(d,(0,n.Z)({},p,i,{components:t,mdxType:"MDXLayout"}),(0,l.kt)("h1",{id:"github-actions-configuration-policy"},"GitHub Actions Configuration Policy"),(0,l.kt)("p",null,"There are several rule types that can be used to configure GitHub Actions."),(0,l.kt)("h2",{id:"github_actions_allowed---which-actions-are-allowed-to-be-used"},(0,l.kt)("inlineCode",{parentName:"h2"},"github_actions_allowed")," - Which actions are allowed to be used"),(0,l.kt)("p",null,"This rule allows you to limit the actions that are allowed to run for a repository.\nIt is recommended to use the ",(0,l.kt)("inlineCode",{parentName:"p"},"selected")," option for allowed actions, and then\nselect the actions that are allowed to run."),(0,l.kt)("h3",{id:"entity"},"Entity"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"repository"))),(0,l.kt)("h3",{id:"type"},"Type"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"github_actions_allowed"))),(0,l.kt)("h3",{id:"rule-parameters"},"Rule parameters"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},"None")),(0,l.kt)("h3",{id:"rule-definition-options"},"Rule definition options"),(0,l.kt)("p",null,"The ",(0,l.kt)("inlineCode",{parentName:"p"},"github_actions_allowed")," rule supports the following options:"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"allowed_actions (enum)")," - Which actions are allowed to be used",(0,l.kt)("ul",{parentName:"li"},(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"all")," - Any action or reusable workflow can be used, regardless of who authored it or where it is defined."),(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"local_only")," - Only actions and reusable workflows that are defined in the repository or organization can be used."),(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"selected")," - Only the actions and reusable workflows that are explicitly listed are allowed. Use the ",(0,l.kt)("inlineCode",{parentName:"li"},"allowed_selected_actions")," ",(0,l.kt)("inlineCode",{parentName:"li"},"rule_type")," to set the list of allowed actions.")))),(0,l.kt)("h2",{id:"allowed_selected_actions---verifies-that-only-allowed-actions-are-used"},(0,l.kt)("inlineCode",{parentName:"h2"},"allowed_selected_actions")," - Verifies that only allowed actions are used"),(0,l.kt)("p",null,"To use this rule, the repository profile for ",(0,l.kt)("inlineCode",{parentName:"p"},"github_actions_allowed")," must\nbe configured to ",(0,l.kt)("inlineCode",{parentName:"p"},"selected"),"."),(0,l.kt)("h3",{id:"entity-1"},"Entity"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"repository"))),(0,l.kt)("h3",{id:"type-1"},"Type"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"allowed_selected_actions"))),(0,l.kt)("h3",{id:"rule-parameters-1"},"Rule parameters"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},"None")),(0,l.kt)("h3",{id:"rule-definition-options-1"},"Rule definition options"),(0,l.kt)("p",null,"The ",(0,l.kt)("inlineCode",{parentName:"p"},"allowed_selected_actions")," rule supports the following options:"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"github_owner_allowed (boolean)")," - Whether GitHub-owned actions are allowed. For example, this includes the actions in the ",(0,l.kt)("inlineCode",{parentName:"li"},"actions")," organization."),(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"verified_allowed (boolean)")," - Whether actions that are verified by GitHub are allowed."),(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"patterns_allowed (boolean)")," - Specifies a list of string-matching patterns to allow specific action(s) and reusable workflow(s). Wildcards, tags, and SHAs are allowed.")),(0,l.kt)("h2",{id:"default_workflow_permissions---sets-the-default-permissions-granted-to-the-github_token-when-running-workflows"},(0,l.kt)("inlineCode",{parentName:"h2"},"default_workflow_permissions")," - Sets the default permissions granted to the ",(0,l.kt)("inlineCode",{parentName:"h2"},"GITHUB_TOKEN")," when running workflows"),(0,l.kt)("p",null,"Verifies the default workflow permissions granted to the GITHUB_TOKEN\nwhen running workflows in a repository, as well as if GitHub Actions\ncan submit approving pull request reviews."),(0,l.kt)("h3",{id:"entity-2"},"Entity"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"repository"))),(0,l.kt)("h3",{id:"type-2"},"Type"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"default_workflow_permissions"))),(0,l.kt)("h3",{id:"rule-parameters-2"},"Rule parameters"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},"None")),(0,l.kt)("h3",{id:"rule-definition-options-2"},"Rule definition options"),(0,l.kt)("p",null,"The ",(0,l.kt)("inlineCode",{parentName:"p"},"default_workflow_permissions")," rule supports the following options:"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"default_workflow_permissions (boolean)")," - Whether GitHub-owned actions are allowed. For example, this includes the actions in the ",(0,l.kt)("inlineCode",{parentName:"li"},"actions")," organization."),(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"can_approve_pull_request_reviews (boolean)")," - Whether the ",(0,l.kt)("inlineCode",{parentName:"li"},"GITHUB_TOKEN")," can approve pull request reviews.")),(0,l.kt)("h2",{id:"actions_check_pinned_tags---verifies-that-any-actions-use-pinned-tags"},(0,l.kt)("inlineCode",{parentName:"h2"},"actions_check_pinned_tags")," - Verifies that any actions use pinned tags"),(0,l.kt)("p",null,"Verifies that actions use pinned tags as opposed to floating tags."),(0,l.kt)("h3",{id:"entity-3"},"Entity"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"repository"))),(0,l.kt)("h3",{id:"type-3"},"Type"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},(0,l.kt)("inlineCode",{parentName:"li"},"actions_check_pinned_tags"))),(0,l.kt)("h3",{id:"rule-parameters-3"},"Rule parameters"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},"None")),(0,l.kt)("h3",{id:"rule-definition-options-3"},"Rule definition options"),(0,l.kt)("ul",null,(0,l.kt)("li",{parentName:"ul"},"None")))}c.isMDXComponent=!0}}]);
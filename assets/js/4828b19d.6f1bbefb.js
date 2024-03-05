"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[4417],{78618:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>d,contentTitle:()=>s,default:()=>h,frontMatter:()=>o,metadata:()=>u,toc:()=>c});var n=r(74848),a=r(28453),l=r(11470),i=r(19365);const o={title:"Auto-remediation via pull request",sidebar_position:65},s="Creating a Pull Request for Autoremediation",u={id:"how-to/remediate-pullrequest",title:"Auto-remediation via pull request",description:"Prerequisites",source:"@site/docs/how-to/remediate-pullrequest.md",sourceDirName:"how-to",slug:"/how-to/remediate-pullrequest",permalink:"/how-to/remediate-pullrequest",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:65,frontMatter:{title:"Auto-remediation via pull request",sidebar_position:65},sidebar:"minder",previous:{title:"Setting up a profile for auto-remediation",permalink:"/how-to/setup-autoremediation"},next:{title:"Setting up a profile for alerts",permalink:"/how-to/setup-alerts"}},d={},c=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Create a rule type that has support for pull request auto remediation",id:"create-a-rule-type-that-has-support-for-pull-request-auto-remediation",level:2},{value:"Create a profile",id:"create-a-profile",level:2},{value:"Limitations",id:"limitations",level:2}];function p(e){const t={a:"a",code:"code",h1:"h1",h2:"h2",li:"li",p:"p",pre:"pre",ul:"ul",...(0,a.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(t.h1,{id:"creating-a-pull-request-for-autoremediation",children:"Creating a Pull Request for Autoremediation"}),"\n",(0,n.jsx)(t.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,n.jsxs)(t.ul,{children:["\n",(0,n.jsxs)(t.li,{children:["The ",(0,n.jsx)(t.code,{children:"minder"})," CLI application"]}),"\n",(0,n.jsx)(t.li,{children:"A Minder account"}),"\n",(0,n.jsx)(t.li,{children:"An enrolled Provider (e.g., GitHub) and registered repositories"}),"\n"]}),"\n",(0,n.jsx)(t.h2,{id:"create-a-rule-type-that-has-support-for-pull-request-auto-remediation",children:"Create a rule type that has support for pull request auto remediation"}),"\n",(0,n.jsx)(t.p,{children:"The pull request auto remediation feature provides the functionality to fix a failed rule type by creating a pull request."}),"\n",(0,n.jsxs)(t.p,{children:["This feature is only available for rule types that support it. To find out if a rule type supports it, check the\n",(0,n.jsx)(t.code,{children:"remediate"})," section in their ",(0,n.jsx)(t.code,{children:"<alert-type>.yaml"})," file. It should have the ",(0,n.jsx)(t.code,{children:"pull_request"})," section defined like below:"]}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-yaml",children:"version: v1\ntype: rule-type\n...\n  remediate:\n    type: pull_request\n...\n"})}),"\n",(0,n.jsxs)(t.p,{children:["In this example, we will use a rule type that checks if a repository has Dependabot enabled. If it's not enabled, Minder\nwill create a pull request that enables Dependabot. The rule type is called ",(0,n.jsx)(t.code,{children:"dependabot_configured.yaml"})," and is one of\nthe reference rule types provided by the Minder team."]}),"\n",(0,n.jsxs)(t.p,{children:["Fetch all the reference rules by cloning the ",(0,n.jsx)(t.a,{href:"https://github.com/stacklok/minder-rules-and-profiles",children:"minder-rules-and-profiles repository"}),"."]}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-bash",children:"git clone https://github.com/stacklok/minder-rules-and-profiles.git\n"})}),"\n",(0,n.jsx)(t.p,{children:"In that directory, you can find all the reference rules and profiles."}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-bash",children:"cd minder-rules-and-profiles\n"})}),"\n",(0,n.jsxs)(t.p,{children:["Create the ",(0,n.jsx)(t.code,{children:"dependabot_configured"})," rule type in Minder:"]}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-bash",children:"minder ruletype create -f rule-types/github/dependabot_configured.yaml\n"})}),"\n",(0,n.jsx)(t.h2,{id:"create-a-profile",children:"Create a profile"}),"\n",(0,n.jsx)(t.p,{children:"Next, create a profile that applies the rule to all registered repositories."}),"\n",(0,n.jsxs)(t.p,{children:["Create a new file called ",(0,n.jsx)(t.code,{children:"profile.yaml"}),"."]}),"\n",(0,n.jsx)(t.p,{children:"Based on your source code language, paste the following profile definition into the newly created file."}),"\n",(0,n.jsxs)(l.A,{children:[(0,n.jsx)(i.A,{value:"go",label:"Go",default:!0,children:(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-yaml",children:'---\nversion: v1\ntype: profile\nname: dependabot-profile\ncontext:\n  provider: github\nalert: "on"\nremediate: "on"\nrepository:\n  - type: dependabot_configured\n    def:\n      package_ecosystem: gomod\n      schedule_interval: weekly\n      apply_if_file: go.mod\n'})})}),(0,n.jsx)(i.A,{value:"npm",label:"NPM",children:(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-yaml",children:'---\nversion: v1\ntype: profile\nname: dependabot-profile\ncontext:\n  provider: github\nalert: "on"\nremediate: "on"\nrepository:\n  - type: dependabot_configured\n    def:\n      package_ecosystem: npm\n      schedule_interval: weekly\n      apply_if_file: package.json\n'})})})]}),"\n",(0,n.jsx)(t.p,{children:"Create the profile in Minder:"}),"\n",(0,n.jsx)(t.pre,{children:(0,n.jsx)(t.code,{className:"language-bash",children:"minder profile create -f profile.yaml\n"})}),"\n",(0,n.jsx)(t.p,{children:"Once the profile is created, Minder will monitor all of your registered repositories matching the expected ecosystem,\ni.e., Go, NPM, etc."}),"\n",(0,n.jsx)(t.p,{children:"If a repository does not have Dependabot enabled, Minder will create a pull request with the necessary configuration\nto enable it. Alongside the pull request, Minder will also create a Security Advisory alert that will be present until the issue\nis resolved."}),"\n",(0,n.jsxs)(t.p,{children:["Alerts are complementary to the remediation feature. If you have both ",(0,n.jsx)(t.code,{children:"alert"})," and ",(0,n.jsx)(t.code,{children:"remediation"})," enabled for a profile,\nMinder will attempt to remediate it first. If the remediation fails, Minder will create an alert. If the remediation\nsucceeds, Minder will close any previously opened alerts related to that rule."]}),"\n",(0,n.jsx)(t.h2,{id:"limitations",children:"Limitations"}),"\n",(0,n.jsxs)(t.ul,{children:["\n",(0,n.jsx)(t.li,{children:"The pull request auto remediation feature is only available for rule types that support it."}),"\n",(0,n.jsx)(t.li,{children:"There's no support for creating pull requests that modify the content of existing files yet."}),"\n",(0,n.jsx)(t.li,{children:"The created pull request should be closed manually if the issue is resolved through other means. The profile status and any related alerts will be updated/closed automatically."}),"\n"]})]})}function h(e={}){const{wrapper:t}={...(0,a.R)(),...e.components};return t?(0,n.jsx)(t,{...e,children:(0,n.jsx)(p,{...e})}):p(e)}},19365:(e,t,r)=>{r.d(t,{A:()=>i});r(96540);var n=r(34164);const a={tabItem:"tabItem_Ymn6"};var l=r(74848);function i(e){let{children:t,hidden:r,className:i}=e;return(0,l.jsx)("div",{role:"tabpanel",className:(0,n.A)(a.tabItem,i),hidden:r,children:t})}},11470:(e,t,r)=>{r.d(t,{A:()=>w});var n=r(96540),a=r(34164),l=r(23104),i=r(56347),o=r(205),s=r(57485),u=r(31682),d=r(89466);function c(e){return n.Children.toArray(e).filter((e=>"\n"!==e)).map((e=>{if(!e||(0,n.isValidElement)(e)&&function(e){const{props:t}=e;return!!t&&"object"==typeof t&&"value"in t}(e))return e;throw new Error(`Docusaurus error: Bad <Tabs> child <${"string"==typeof e.type?e.type:e.type.name}>: all children of the <Tabs> component should be <TabItem>, and every <TabItem> should have a unique "value" prop.`)}))?.filter(Boolean)??[]}function p(e){const{values:t,children:r}=e;return(0,n.useMemo)((()=>{const e=t??function(e){return c(e).map((e=>{let{props:{value:t,label:r,attributes:n,default:a}}=e;return{value:t,label:r,attributes:n,default:a}}))}(r);return function(e){const t=(0,u.X)(e,((e,t)=>e.value===t.value));if(t.length>0)throw new Error(`Docusaurus error: Duplicate values "${t.map((e=>e.value)).join(", ")}" found in <Tabs>. Every value needs to be unique.`)}(e),e}),[t,r])}function h(e){let{value:t,tabValues:r}=e;return r.some((e=>e.value===t))}function f(e){let{queryString:t=!1,groupId:r}=e;const a=(0,i.W6)(),l=function(e){let{queryString:t=!1,groupId:r}=e;if("string"==typeof t)return t;if(!1===t)return null;if(!0===t&&!r)throw new Error('Docusaurus error: The <Tabs> component groupId prop is required if queryString=true, because this value is used as the search param name. You can also provide an explicit value such as queryString="my-search-param".');return r??null}({queryString:t,groupId:r});return[(0,s.aZ)(l),(0,n.useCallback)((e=>{if(!l)return;const t=new URLSearchParams(a.location.search);t.set(l,e),a.replace({...a.location,search:t.toString()})}),[l,a])]}function m(e){const{defaultValue:t,queryString:r=!1,groupId:a}=e,l=p(e),[i,s]=(0,n.useState)((()=>function(e){let{defaultValue:t,tabValues:r}=e;if(0===r.length)throw new Error("Docusaurus error: the <Tabs> component requires at least one <TabItem> children component");if(t){if(!h({value:t,tabValues:r}))throw new Error(`Docusaurus error: The <Tabs> has a defaultValue "${t}" but none of its children has the corresponding value. Available values are: ${r.map((e=>e.value)).join(", ")}. If you intend to show no default tab, use defaultValue={null} instead.`);return t}const n=r.find((e=>e.default))??r[0];if(!n)throw new Error("Unexpected error: 0 tabValues");return n.value}({defaultValue:t,tabValues:l}))),[u,c]=f({queryString:r,groupId:a}),[m,b]=function(e){let{groupId:t}=e;const r=function(e){return e?`docusaurus.tab.${e}`:null}(t),[a,l]=(0,d.Dv)(r);return[a,(0,n.useCallback)((e=>{r&&l.set(e)}),[r,l])]}({groupId:a}),y=(()=>{const e=u??m;return h({value:e,tabValues:l})?e:null})();(0,o.A)((()=>{y&&s(y)}),[y]);return{selectedValue:i,selectValue:(0,n.useCallback)((e=>{if(!h({value:e,tabValues:l}))throw new Error(`Can't select invalid tab value=${e}`);s(e),c(e),b(e)}),[c,b,l]),tabValues:l}}var b=r(92303);const y={tabList:"tabList__CuJ",tabItem:"tabItem_LNqP"};var g=r(74848);function v(e){let{className:t,block:r,selectedValue:n,selectValue:i,tabValues:o}=e;const s=[],{blockElementScrollPositionUntilNextRender:u}=(0,l.a_)(),d=e=>{const t=e.currentTarget,r=s.indexOf(t),a=o[r].value;a!==n&&(u(t),i(a))},c=e=>{let t=null;switch(e.key){case"Enter":d(e);break;case"ArrowRight":{const r=s.indexOf(e.currentTarget)+1;t=s[r]??s[0];break}case"ArrowLeft":{const r=s.indexOf(e.currentTarget)-1;t=s[r]??s[s.length-1];break}}t?.focus()};return(0,g.jsx)("ul",{role:"tablist","aria-orientation":"horizontal",className:(0,a.A)("tabs",{"tabs--block":r},t),children:o.map((e=>{let{value:t,label:r,attributes:l}=e;return(0,g.jsx)("li",{role:"tab",tabIndex:n===t?0:-1,"aria-selected":n===t,ref:e=>s.push(e),onKeyDown:c,onClick:d,...l,className:(0,a.A)("tabs__item",y.tabItem,l?.className,{"tabs__item--active":n===t}),children:r??t},t)}))})}function x(e){let{lazy:t,children:r,selectedValue:a}=e;const l=(Array.isArray(r)?r:[r]).filter(Boolean);if(t){const e=l.find((e=>e.props.value===a));return e?(0,n.cloneElement)(e,{className:"margin-top--md"}):null}return(0,g.jsx)("div",{className:"margin-top--md",children:l.map(((e,t)=>(0,n.cloneElement)(e,{key:t,hidden:e.props.value!==a})))})}function j(e){const t=m(e);return(0,g.jsxs)("div",{className:(0,a.A)("tabs-container",y.tabList),children:[(0,g.jsx)(v,{...e,...t}),(0,g.jsx)(x,{...e,...t})]})}function w(e){const t=(0,b.A)();return(0,g.jsx)(j,{...e,children:c(e.children)},String(t))}},28453:(e,t,r)=>{r.d(t,{R:()=>i,x:()=>o});var n=r(96540);const a={},l=n.createContext(a);function i(e){const t=n.useContext(l);return n.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function o(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(a):e.components||a:i(e.components),n.createElement(l.Provider,{value:t},e.children)}}}]);
"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[5584],{96045:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>s,default:()=>p,frontMatter:()=>a,metadata:()=>i,toc:()=>d});const i=JSON.parse('{"id":"understand/remediations","title":"Automatic remediation","description":"Minder can perform automatic remediation for many rules in an attempt to","source":"@site/docs/understand/remediations.md","sourceDirName":"understand","slug":"/understand/remediations","permalink":"/understand/remediations","draft":false,"unlisted":false,"tags":[],"version":"current","sidebarPosition":70,"frontMatter":{"title":"Automatic remediation","sidebar_position":70},"sidebar":"minder","previous":{"title":"Alerting","permalink":"/understand/alerts"},"next":{"title":"Creating a profile","permalink":"/how-to/create_profile"}}');var r=t(74848),o=t(28453);const a={title:"Automatic remediation",sidebar_position:70},s=void 0,l={},d=[{value:"Enabling remediations in a profile",id:"enabling-remediations-in-a-profile",level:2},{value:"Limitations",id:"limitations",level:2}];function c(e){const n={a:"a",code:"code",em:"em",h2:"h2",p:"p",pre:"pre",...(0,o.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsxs)(n.p,{children:["Minder can perform ",(0,r.jsx)(n.em,{children:"automatic remediation"})," for many rules in an attempt to\nresolve problems in your software supply chain, and bring your resources into\ncompliance with your ",(0,r.jsx)(n.a,{href:"/understand/profiles",children:"profile"}),"."]}),"\n",(0,r.jsx)(n.p,{children:"The steps to take during automatic remediation are defined within the rule\nitself and can perform actions like sending a REST call to an endpoint to change\nconfiguration, or creating a pull request with a proposed fix."}),"\n",(0,r.jsx)(n.p,{children:"For example, if you have a rule in your profile that specifies that Secret\nScanning should be enabled, and you have enabled automatic remediation in your\nprofile, then Minder will attempt to turn Secret Scanning on in any repositories\nwhere it is not enabled."}),"\n",(0,r.jsx)(n.h2,{id:"enabling-remediations-in-a-profile",children:"Enabling remediations in a profile"}),"\n",(0,r.jsx)(n.p,{children:'To activate the remediation feature within a profile, you need to adjust the\nYAML definition. Specifically, you should set the remediate parameter to "on":'}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"remediate: 'on'\n"})}),"\n",(0,r.jsx)(n.p,{children:"Enabling remediation at the profile level means that for any rules included in\nthe profile, a remediation action will be taken for any rule failures."}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'---\nversion: v1\ntype: rule-type\nname: sample_rule\ndef:\n  remediate:\n    type: rest\n    rest:\n      method: PATCH\n      endpoint: \'/repos/{{.Entity.Owner}}/{{.Entity.Name}}\'\n      body: |\n        { "security_and_analysis": {"secret_scanning": { "status": "enabled" } } }\n'})}),"\n",(0,r.jsxs)(n.p,{children:["In this example, the ",(0,r.jsx)(n.code,{children:"sample_rule"})," defines a remediation action that performs a\nPATCH request to an endpoint. This request will modify the state of the\nrepository ensuring it complies with the rule."]}),"\n",(0,r.jsx)(n.p,{children:"Now, let's see how this works in practice within a profile. Consider the\nfollowing profile configuration with remediation turned on:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"version: v1\ntype: profile\nname: sample-profile\ncontext:\n  provider: github\nremediate: 'on'\nrepository:\n  - type: sample_rule\n    def:\n      enabled: true\n"})}),"\n",(0,r.jsxs)(n.p,{children:["In this profile, all repositories that do not meet the conditions specified in\nthe ",(0,r.jsx)(n.code,{children:"sample_rule"})," will automatically receive a PATCH request to the specified\nendpoint. This action will make the repository compliant."]}),"\n",(0,r.jsx)(n.h2,{id:"limitations",children:"Limitations"}),"\n",(0,r.jsxs)(n.p,{children:["Some rule types do not support automatic remediations, due to platform\nlimitations. For example, it may be possible to query the status of a repository\nconfiguration, but there may not be an API to ",(0,r.jsx)(n.em,{children:"change"})," the configuration. In\nsuch case, a rule type could detect problems but would not be able to remediate."]}),"\n",(0,r.jsx)(n.p,{children:"To identify which rule types support remediation, you can run:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"minder ruletype list -oyaml\n"})}),"\n",(0,r.jsxs)(n.p,{children:["This will show all the rule types; a rule type with a ",(0,r.jsx)(n.code,{children:"remediate"})," attribute\nsupports automatic remediation."]}),"\n",(0,r.jsxs)(n.p,{children:["Furthermore, remediations that open a pull request such as the ",(0,r.jsx)(n.code,{children:"dependabot"})," rule\ntype only attempt to replace the target file, overwriting its contents. This\nmeans that if you want to keep the current changes, you need to merge the\ncontents manually."]})]})}function p(e={}){const{wrapper:n}={...(0,o.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(c,{...e})}):c(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>a,x:()=>s});var i=t(96540);const r={},o=i.createContext(r);function a(e){const n=i.useContext(o);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function s(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:a(e.components),i.createElement(o.Provider,{value:n},e.children)}}}]);
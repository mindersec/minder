"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[1206],{39148:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>o,default:()=>p,frontMatter:()=>a,metadata:()=>s,toc:()=>d});var i=t(74848),r=t(28453);const a={title:"Remediations",sidebar_position:40},o="Alerts and Automatic Remediation in Minder",s={id:"understand/remediation",title:"Remediations",description:"A profile in Minder offers a comprehensive view of your security posture, encompassing more than just the status report.",source:"@site/docs/understand/remediation.md",sourceDirName:"understand",slug:"/understand/remediation",permalink:"/understand/remediation",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:40,frontMatter:{title:"Remediations",sidebar_position:40},sidebar:"minder",previous:{title:"Profiles",permalink:"/understand/profiles"},next:{title:"Alerts",permalink:"/understand/alerts"}},l={},d=[{value:"Enabling alerts in a profile",id:"enabling-alerts-in-a-profile",level:3},{value:"Enabling remediations in a profile",id:"enabling-remediations-in-a-profile",level:3}];function c(e){const n={code:"code",h1:"h1",h3:"h3",p:"p",pre:"pre",...(0,r.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.h1,{id:"alerts-and-automatic-remediation-in-minder",children:"Alerts and Automatic Remediation in Minder"}),"\n",(0,i.jsx)(n.p,{children:"A profile in Minder offers a comprehensive view of your security posture, encompassing more than just the status report.\nIt actively responds to any rules that are not in compliance, taking specific actions. These actions can include the\ncreation of alerts for rules that have failed, as well as the execution of remediations to fix the non-compliant\naspects."}),"\n",(0,i.jsx)(n.p,{children:"When alerting is turned on in a profile, Minder will open an alert to bring your attention to the non-compliance issue.\nConversely, when the rule evaluation passes, Minder will automatically close any previously opened alerts related to\nthat rule."}),"\n",(0,i.jsx)(n.p,{children:"When remediation is turned on, Minder also supports the ability to automatically remediate failed rules based on their\ntype, i.e., by processing a REST call to enable/disable a non-compliant repository setting or creating a pull request\nwith a proposed fix. Note that not all rule types support automatic remediation yet."}),"\n",(0,i.jsx)(n.h3,{id:"enabling-alerts-in-a-profile",children:"Enabling alerts in a profile"}),"\n",(0,i.jsx)(n.p,{children:'To activate the alert feature within a profile, you need to adjust the YAML definition.\nSpecifically, you should set the alert parameter to "on":'}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'alert: "on"\n'})}),"\n",(0,i.jsx)(n.p,{children:"Enabling alerts at the profile level means that for any rules included in the profile, alerts will be generated for\nany rule failures. For better clarity, consider this rule snippet:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'---\nversion: v1\ntype: rule-type\nname: sample_rule\ndef:\n  alert:\n      type: security_advisory\n      security_advisory:\n        severity: "medium"\n'})}),"\n",(0,i.jsxs)(n.p,{children:["In this example, the ",(0,i.jsx)(n.code,{children:"sample_rule"})," defines an alert action that creates a medium severity security advisory in the\nrepository for any non-compliant repositories."]}),"\n",(0,i.jsx)(n.p,{children:"Now, let's see how this works in practice within a profile. Consider the following profile configuration with alerts\nturned on:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'version: v1\ntype: profile\nname: sample-profile\ncontext:\n  provider: github\nalert: "on"\nrepository:\n  - type: sample_rule\n    def:\n      enabled: true\n'})}),"\n",(0,i.jsxs)(n.p,{children:["In this profile, all repositories that do not meet the conditions specified in the ",(0,i.jsx)(n.code,{children:"sample_rule"})," will automatically\ngenerate security advisories."]}),"\n",(0,i.jsx)(n.h3,{id:"enabling-remediations-in-a-profile",children:"Enabling remediations in a profile"}),"\n",(0,i.jsx)(n.p,{children:'To activate the remediation feature within a profile, you need to adjust the YAML definition.\nSpecifically, you should set the remediate parameter to "on":'}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'remediate: "on"\n'})}),"\n",(0,i.jsx)(n.p,{children:"Enabling remediation at the profile level means that for any rules included in the profile, a remediation action will be\ntaken for any rule failures."}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'---\nversion: v1\ntype: rule-type\nname: sample_rule\ndef:\n  remediate:\n    type: rest\n    rest:\n      method: PATCH\n      endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"\n      body: |\n        { "security_and_analysis": {"secret_scanning": { "status": "enabled" } } }\n'})}),"\n",(0,i.jsxs)(n.p,{children:["In this example, the ",(0,i.jsx)(n.code,{children:"sample_rule"})," defines a remediation action that performs a PATCH request to an endpoint. This\nrequest will modify the state of the repository ensuring it complies with the rule."]}),"\n",(0,i.jsx)(n.p,{children:"Now, let's see how this works in practice within a profile. Consider the following profile configuration with\nremediation turned on:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-yaml",children:'version: v1\ntype: profile\nname: sample-profile\ncontext:\n  provider: github\nremediate: "on"\nrepository:\n  - type: sample_rule\n    def:\n      enabled: true\n'})}),"\n",(0,i.jsxs)(n.p,{children:["In this profile, all repositories that do not meet the conditions specified in the ",(0,i.jsx)(n.code,{children:"sample_rule"})," will automatically\nreceive a PATCH request to the specified endpoint. This action will make the repository compliant."]})]})}function p(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(c,{...e})}):c(e)}},28453:(e,n,t)=>{t.d(n,{R:()=>o,x:()=>s});var i=t(96540);const r={},a=i.createContext(r);function o(e){const n=i.useContext(a);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function s(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),i.createElement(a.Provider,{value:n},e.children)}}}]);
"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[7898],{72554:(e,r,n)=>{n.r(r),n.d(r,{assets:()=>d,contentTitle:()=>s,default:()=>p,frontMatter:()=>o,metadata:()=>a,toc:()=>l});var t=n(74848),i=n(28453);const o={title:"Providers",sidebar_position:20},s="Providers",a={id:"integrations/providers",title:"Providers",description:"A provider connects Minder to your software supply chain. It lets Minder know where to look for your repositories, artifacts,",source:"@site/docs/integrations/providers.md",sourceDirName:"integrations",slug:"/integrations/providers",permalink:"/integrations/providers",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:20,frontMatter:{title:"Providers",sidebar_position:20},sidebar:"minder",previous:{title:"Minder Integrations",permalink:"/integrations/overview"},next:{title:"Community Tooling Integrations",permalink:"/integrations/community_integrations"}},d={},l=[{value:"Enrolling a provider",id:"enrolling-a-provider",level:2}];function c(e){const r={code:"code",h1:"h1",h2:"h2",li:"li",p:"p",pre:"pre",ul:"ul",...(0,i.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(r.h1,{id:"providers",children:"Providers"}),"\n",(0,t.jsx)(r.p,{children:"A provider connects Minder to your software supply chain. It lets Minder know where to look for your repositories, artifacts,\nand other entities are, in order to make them available for registration. It also tells Minder how to interact with your\nsupply chain to enable features such as alerting and remediation. Finally, it handles the way Minder authenticates\nto the external service."}),"\n",(0,t.jsx)(r.p,{children:"The currently supported providers are:"}),"\n",(0,t.jsxs)(r.ul,{children:["\n",(0,t.jsx)(r.li,{children:"GitHub"}),"\n"]}),"\n",(0,t.jsx)(r.p,{children:"Stay tuned as we add more providers in the future!"}),"\n",(0,t.jsx)(r.h2,{id:"enrolling-a-provider",children:"Enrolling a provider"}),"\n",(0,t.jsx)(r.p,{children:"To enroll GitHub as a provider, use the following command:"}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{children:"minder provider enroll\n"})}),"\n",(0,t.jsx)(r.p,{children:"Once a provider is enrolled, public repositories from that provider can be registered with Minder. Security profiles\ncan then be applied to the registered repositories, giving you an overview of your security posture and providing\nremediations to improve your security posture."})]})}function p(e={}){const{wrapper:r}={...(0,i.R)(),...e.components};return r?(0,t.jsx)(r,{...e,children:(0,t.jsx)(c,{...e})}):c(e)}},28453:(e,r,n)=>{n.d(r,{R:()=>s,x:()=>a});var t=n(96540);const i={},o=t.createContext(i);function s(e){const r=t.useContext(o);return t.useMemo((function(){return"function"==typeof e?e(r):{...r,...e}}),[r,e])}function a(e){let r;return r=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:s(e.components),t.createElement(o.Provider,{value:r},e.children)}}}]);
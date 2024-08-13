"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[3802],{36520:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>a,contentTitle:()=>s,default:()=>u,frontMatter:()=>o,metadata:()=>c,toc:()=>l});var r=n(74848),i=n(28453);const o={title:"Architecture overview",sidebar_position:60},s="System Architecture",c={id:"developer_guide/architecture",title:"Architecture overview",description:"While it is built as a single container, Minder implements several external",source:"@site/docs/developer_guide/architecture.md",sourceDirName:"developer_guide",slug:"/developer_guide/architecture",permalink:"/developer_guide/architecture",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:60,frontMatter:{title:"Architecture overview",sidebar_position:60},sidebar:"minder",previous:{title:"Feature flags",permalink:"/developer_guide/feature_flags"},next:{title:"Adding Users to your Project",permalink:"/user_management/adding_users"}},a={},l=[];function d(e){const t={a:"a",h1:"h1",header:"header",mermaid:"mermaid",p:"p",...(0,i.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(t.header,{children:(0,r.jsx)(t.h1,{id:"system-architecture",children:"System Architecture"})}),"\n",(0,r.jsxs)(t.p,{children:["While it is built as a single container, Minder implements several external\ninterfaces for different components. In addition to the GRPC and HTTP service\nports, it also leverages the ",(0,r.jsx)(t.a,{href:"https://watermill.io",children:"watermill library"})," to queue\nand route events within the application."]}),"\n",(0,r.jsx)(t.p,{children:"The following is a high-level diagram of the Minder architecture"}),"\n",(0,r.jsx)(t.mermaid,{value:'flowchart LR\n    subgraph minder\n        %% flow from top to bottom\n        direction TB\n\n        grpc>GRPC endpoint]\n        click grpc "/api" "GRPC auto-generated documentation"\n        web>HTTP endpoint]\n        click web "https://github.com/stacklok/minder/blob/main/internal/controlplane/server.go#L210" "Webserver URL registration code"\n        events("watermill")\n        click events "https://watermill.io/docs" "Watermill event processing library"\n\n        handler>Event handlers]\n        click handler "https://github.com/stacklok/minder/blob/main/cmd/server/app/serve.go#L69" "Registered event handlers"\n    end\n\n    cloud([GitHub])\n    cli("<code>minder</code> CLI")\n    click cli "https://github.com/stacklok/minder/tree/main/cmd/cli"\n\n    db[(Postgres)]\n    click postgres "/db/minder_db_schema" "Database schema"\n\n    cli --\x3e grpc\n    cli --OAuth--\x3e web\n    cloud --\x3e web\n\n    grpc --\x3e db\n    web --\x3e db\n\n    web --\x3e events\n\n    events --\x3e handler'})]})}function u(e={}){const{wrapper:t}={...(0,i.R)(),...e.components};return t?(0,r.jsx)(t,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>s,x:()=>c});var r=n(96540);const i={},o=r.createContext(i);function s(e){const t=r.useContext(o);return r.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function c(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:s(e.components),r.createElement(o.Provider,{value:t},e.children)}}}]);
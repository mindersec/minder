"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[4505],{45432:(e,n,i)=>{i.r(n),i.d(n,{assets:()=>l,contentTitle:()=>c,default:()=>m,frontMatter:()=>s,metadata:()=>r,toc:()=>d});const r=JSON.parse('{"id":"ref/cli/minder_config","title":"minder config","description":"minder config","source":"@site/docs/ref/cli/minder_config.md","sourceDirName":"ref/cli","slug":"/ref/cli/minder_config","permalink":"/ref/cli/minder_config","draft":false,"unlisted":false,"tags":[],"version":"current","frontMatter":{"title":"minder config"},"sidebar":"minder","previous":{"title":"minder completion zsh","permalink":"/ref/cli/minder_completion_zsh"},"next":{"title":"minder docs","permalink":"/ref/cli/minder_docs"}}');var t=i(74848),o=i(28453);const s={title:"minder config"},c=void 0,l={},d=[{value:"minder config",id:"minder-config",level:2},{value:"Synopsis",id:"synopsis",level:3},{value:"Options",id:"options",level:3},{value:"Options inherited from parent commands",id:"options-inherited-from-parent-commands",level:3},{value:"SEE ALSO",id:"see-also",level:3}];function a(e){const n={a:"a",code:"code",h2:"h2",h3:"h3",li:"li",p:"p",pre:"pre",ul:"ul",...(0,o.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(n.h2,{id:"minder-config",children:"minder config"}),"\n",(0,t.jsx)(n.p,{children:"How to manage minder CLI configuration"}),"\n",(0,t.jsx)(n.h3,{id:"synopsis",children:"Synopsis"}),"\n",(0,t.jsx)(n.p,{children:"In addition to the command-line flags, many minder options can be set via a configuration file in the YAML format."}),"\n",(0,t.jsx)(n.p,{children:"Configuration options include:"}),"\n",(0,t.jsxs)(n.ul,{children:["\n",(0,t.jsx)(n.li,{children:"provider"}),"\n",(0,t.jsx)(n.li,{children:"project"}),"\n",(0,t.jsx)(n.li,{children:"output"}),"\n",(0,t.jsx)(n.li,{children:"grpc_server.host"}),"\n",(0,t.jsx)(n.li,{children:"grpc_server.port"}),"\n",(0,t.jsx)(n.li,{children:"grpc_server.insecure"}),"\n",(0,t.jsx)(n.li,{children:"identity.cli.issuer_url"}),"\n",(0,t.jsx)(n.li,{children:"identity.cli.client_id"}),"\n"]}),"\n",(0,t.jsx)(n.p,{children:"By default, we look for the file as $PWD/config.yaml and $XDG_CONFIG_PATH/minder/config.yaml. You can specify a custom path via the --config flag, or by setting the MINDER_CONFIG environment variable."}),"\n",(0,t.jsx)(n.h3,{id:"options",children:"Options"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{children:"  -h, --help   help for config\n"})}),"\n",(0,t.jsx)(n.h3,{id:"options-inherited-from-parent-commands",children:"Options inherited from parent commands"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{children:'      --config string            Config file (default is $PWD/config.yaml)\n      --grpc-host string         Server host (default "api.stacklok.com")\n      --grpc-insecure            Allow establishing insecure connections\n      --grpc-port int            Server port (default 443)\n      --identity-client string   Identity server client ID (default "minder-cli")\n      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")\n  -v, --verbose                  Output additional messages to STDERR\n'})}),"\n",(0,t.jsx)(n.h3,{id:"see-also",children:"SEE ALSO"}),"\n",(0,t.jsxs)(n.ul,{children:["\n",(0,t.jsxs)(n.li,{children:[(0,t.jsx)(n.a,{href:"/ref/cli/minder",children:"minder"}),"\t - Minder controls the hosted minder service"]}),"\n"]})]})}function m(e={}){const{wrapper:n}={...(0,o.R)(),...e.components};return n?(0,t.jsx)(n,{...e,children:(0,t.jsx)(a,{...e})}):a(e)}},28453:(e,n,i)=>{i.d(n,{R:()=>s,x:()=>c});var r=i(96540);const t={},o=r.createContext(t);function s(e){const n=r.useContext(o);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function c(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:s(e.components),r.createElement(o.Provider,{value:n},e.children)}}}]);
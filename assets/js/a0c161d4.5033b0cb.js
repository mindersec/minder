"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[3084],{3905:(e,t,r)=>{r.d(t,{Zo:()=>u,kt:()=>m});var n=r(67294);function i(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function l(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function p(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?l(Object(r),!0).forEach((function(t){i(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):l(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function a(e,t){if(null==e)return{};var r,n,i=function(e,t){if(null==e)return{};var r,n,i={},l=Object.keys(e);for(n=0;n<l.length;n++)r=l[n],t.indexOf(r)>=0||(i[r]=e[r]);return i}(e,t);if(Object.getOwnPropertySymbols){var l=Object.getOwnPropertySymbols(e);for(n=0;n<l.length;n++)r=l[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(i[r]=e[r])}return i}var o=n.createContext({}),c=function(e){var t=n.useContext(o),r=t;return e&&(r="function"==typeof e?e(t):p(p({},t),e)),r},u=function(e){var t=c(e.components);return n.createElement(o.Provider,{value:t},e.children)},s="mdxType",d={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},y=n.forwardRef((function(e,t){var r=e.components,i=e.mdxType,l=e.originalType,o=e.parentName,u=a(e,["components","mdxType","originalType","parentName"]),s=c(r),y=i,m=s["".concat(o,".").concat(y)]||s[y]||d[y]||l;return r?n.createElement(m,p(p({ref:t},u),{},{components:r})):n.createElement(m,p({ref:t},u))}));function m(e,t){var r=arguments,i=t&&t.mdxType;if("string"==typeof e||i){var l=r.length,p=new Array(l);p[0]=y;var a={};for(var o in t)hasOwnProperty.call(t,o)&&(a[o]=t[o]);a.originalType=e,a[s]="string"==typeof e?e:i,p[1]=a;for(var c=2;c<l;c++)p[c]=r[c];return n.createElement.apply(null,p)}return n.createElement.apply(null,r)}y.displayName="MDXCreateElement"},72700:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>o,contentTitle:()=>p,default:()=>d,frontMatter:()=>l,metadata:()=>a,toc:()=>c});var n=r(87462),i=(r(67294),r(3905));const l={title:"minder ruletype apply"},p=void 0,a={unversionedId:"ref/cli/minder_ruletype_apply",id:"ref/cli/minder_ruletype_apply",title:"minder ruletype apply",description:"minder ruletype apply",source:"@site/docs/ref/cli/minder_ruletype_apply.md",sourceDirName:"ref/cli",slug:"/ref/cli/minder_ruletype_apply",permalink:"/ref/cli/minder_ruletype_apply",draft:!1,tags:[],version:"current",frontMatter:{title:"minder ruletype apply"},sidebar:"minder",previous:{title:"minder ruletype",permalink:"/ref/cli/minder_ruletype"},next:{title:"minder ruletype create",permalink:"/ref/cli/minder_ruletype_create"}},o={},c=[{value:"minder ruletype apply",id:"minder-ruletype-apply",level:2},{value:"Synopsis",id:"synopsis",level:3},{value:"Options",id:"options",level:3},{value:"Options inherited from parent commands",id:"options-inherited-from-parent-commands",level:3},{value:"SEE ALSO",id:"see-also",level:3}],u={toc:c},s="wrapper";function d(e){let{components:t,...r}=e;return(0,i.kt)(s,(0,n.Z)({},u,r,{components:t,mdxType:"MDXLayout"}),(0,i.kt)("h2",{id:"minder-ruletype-apply"},"minder ruletype apply"),(0,i.kt)("p",null,"Apply a rule type within a minder control plane"),(0,i.kt)("h3",{id:"synopsis"},"Synopsis"),(0,i.kt)("p",null,"The minder rule type apply subcommand lets you create or update rule types for a project\nwithin a minder control plane."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},"minder ruletype apply [flags]\n")),(0,i.kt)("h3",{id:"options"},"Options"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},"  -f, --file stringArray   Path to the YAML defining the rule type (or - for stdin). Can be specified multiple times. Can be a directory.\n  -h, --help               help for apply\n")),(0,i.kt)("h3",{id:"options-inherited-from-parent-commands"},"Options inherited from parent commands"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},'      --config string            Config file (default is $PWD/config.yaml)\n      --grpc-host string         Server host (default "api.stacklok.com")\n      --grpc-insecure            Allow establishing insecure connections\n      --grpc-port int            Server port (default 443)\n      --identity-client string   Identity server client ID (default "minder-cli")\n      --identity-url string      Identity server issuer URL (default "https://auth.stacklok.com")\n')),(0,i.kt)("h3",{id:"see-also"},"SEE ALSO"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_ruletype"},"minder ruletype"),"\t - Manage rule types within a minder control plane")))}d.isMDXComponent=!0}}]);
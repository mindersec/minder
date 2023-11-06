"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[5727],{3905:(e,t,r)=>{r.d(t,{Zo:()=>d,kt:()=>f});var n=r(67294);function i(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function a(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function o(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?a(Object(r),!0).forEach((function(t){i(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):a(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function l(e,t){if(null==e)return{};var r,n,i=function(e,t){if(null==e)return{};var r,n,i={},a=Object.keys(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||(i[r]=e[r]);return i}(e,t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(i[r]=e[r])}return i}var c=n.createContext({}),u=function(e){var t=n.useContext(c),r=t;return e&&(r="function"==typeof e?e(t):o(o({},t),e)),r},d=function(e){var t=u(e.components);return n.createElement(c.Provider,{value:t},e.children)},p="mdxType",s={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},m=n.forwardRef((function(e,t){var r=e.components,i=e.mdxType,a=e.originalType,c=e.parentName,d=l(e,["components","mdxType","originalType","parentName"]),p=u(r),m=i,f=p["".concat(c,".").concat(m)]||p[m]||s[m]||a;return r?n.createElement(f,o(o({ref:t},d),{},{components:r})):n.createElement(f,o({ref:t},d))}));function f(e,t){var r=arguments,i=t&&t.mdxType;if("string"==typeof e||i){var a=r.length,o=new Array(a);o[0]=m;var l={};for(var c in t)hasOwnProperty.call(t,c)&&(l[c]=t[c]);l.originalType=e,l[p]="string"==typeof e?e:i,o[1]=l;for(var u=2;u<a;u++)o[u]=r[u];return n.createElement.apply(null,o)}return n.createElement.apply(null,r)}m.displayName="MDXCreateElement"},83290:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>c,contentTitle:()=>o,default:()=>s,frontMatter:()=>a,metadata:()=>l,toc:()=>u});var n=r(87462),i=(r(67294),r(3905));const a={title:"minder auth"},o=void 0,l={unversionedId:"ref/cli/minder_auth",id:"ref/cli/minder_auth",title:"minder auth",description:"minder auth",source:"@site/docs/ref/cli/minder_auth.md",sourceDirName:"ref/cli",slug:"/ref/cli/minder_auth",permalink:"/ref/cli/minder_auth",draft:!1,tags:[],version:"current",frontMatter:{title:"minder auth"},sidebar:"minder",previous:{title:"minder artifact list",permalink:"/ref/cli/minder_artifact_list"},next:{title:"minder auth delete",permalink:"/ref/cli/minder_auth_delete"}},c={},u=[{value:"minder auth",id:"minder-auth",level:2},{value:"Synopsis",id:"synopsis",level:3},{value:"Options",id:"options",level:3},{value:"Options inherited from parent commands",id:"options-inherited-from-parent-commands",level:3},{value:"SEE ALSO",id:"see-also",level:3}],d={toc:u},p="wrapper";function s(e){let{components:t,...r}=e;return(0,i.kt)(p,(0,n.Z)({},d,r,{components:t,mdxType:"MDXLayout"}),(0,i.kt)("h2",{id:"minder-auth"},"minder auth"),(0,i.kt)("p",null,"Authorize and manage accounts within a minder control plane"),(0,i.kt)("h3",{id:"synopsis"},"Synopsis"),(0,i.kt)("p",null,"The minder auth command project lets you create accounts and grant or revoke\nauthorization to existing accounts within a minder control plane."),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},"minder auth [flags]\n")),(0,i.kt)("h3",{id:"options"},"Options"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},"  -h, --help   help for auth\n")),(0,i.kt)("h3",{id:"options-inherited-from-parent-commands"},"Options inherited from parent commands"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre"},'      --config string            Config file (default is $PWD/config.yaml)\n      --grpc-host string         Server host (default "staging.stacklok.dev")\n      --grpc-insecure            Allow establishing insecure connections\n      --grpc-port int            Server port (default 443)\n      --identity-client string   Identity server client ID (default "minder-cli")\n      --identity-realm string    Identity server realm (default "stacklok")\n      --identity-url string      Identity server issuer URL (default "https://auth.staging.stacklok.dev")\n')),(0,i.kt)("h3",{id:"see-also"},"SEE ALSO"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder"},"minder"),"\t - Minder controls the hosted minder service"),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_auth_delete"},"minder auth delete"),"\t - Permanently delete account"),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_auth_login"},"minder auth login"),"\t - Login to a minder control plane."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_auth_logout"},"minder auth logout"),"\t - Logout from minder control plane."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_auth_revoke_provider"},"minder auth revoke_provider"),"\t - Revoke access tokens for provider"),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/ref/cli/minder_auth_whoami"},"minder auth whoami"),"\t - whoami for current user")))}s.isMDXComponent=!0}}]);
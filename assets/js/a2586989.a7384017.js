"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[4322],{3905:(e,t,r)=>{r.d(t,{Zo:()=>c,kt:()=>f});var n=r(67294);function i(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function a(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function o(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?a(Object(r),!0).forEach((function(t){i(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):a(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function s(e,t){if(null==e)return{};var r,n,i=function(e,t){if(null==e)return{};var r,n,i={},a=Object.keys(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||(i[r]=e[r]);return i}(e,t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(i[r]=e[r])}return i}var l=n.createContext({}),u=function(e){var t=n.useContext(l),r=t;return e&&(r="function"==typeof e?e(t):o(o({},t),e)),r},c=function(e){var t=u(e.components);return n.createElement(l.Provider,{value:t},e.children)},p="mdxType",d={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},m=n.forwardRef((function(e,t){var r=e.components,i=e.mdxType,a=e.originalType,l=e.parentName,c=s(e,["components","mdxType","originalType","parentName"]),p=u(r),m=i,f=p["".concat(l,".").concat(m)]||p[m]||d[m]||a;return r?n.createElement(f,o(o({ref:t},c),{},{components:r})):n.createElement(f,o({ref:t},c))}));function f(e,t){var r=arguments,i=t&&t.mdxType;if("string"==typeof e||i){var a=r.length,o=new Array(a);o[0]=m;var s={};for(var l in t)hasOwnProperty.call(t,l)&&(s[l]=t[l]);s.originalType=e,s[p]="string"==typeof e?e:i,o[1]=s;for(var u=2;u<a;u++)o[u]=r[u];return n.createElement.apply(null,o)}return n.createElement.apply(null,r)}m.displayName="MDXCreateElement"},83993:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>l,contentTitle:()=>o,default:()=>d,frontMatter:()=>a,metadata:()=>s,toc:()=>u});var n=r(87462),i=(r(67294),r(3905));const a={title:"Quickstart with Minder (< 1 min)",sidebar_position:20},o="Quickstart with Minder (< 1 min)",s={unversionedId:"getting_started/quickstart",id:"getting_started/quickstart",title:"Quickstart with Minder (< 1 min)",description:'Minder provides a "happy path" that guides you through the process of creating your first profile in Minder. In just a few seconds, you will register your repositories and enable secret scanning protection for all of them!',source:"@site/docs/getting_started/quickstart.md",sourceDirName:"getting_started",slug:"/getting_started/quickstart",permalink:"/getting_started/quickstart",draft:!1,tags:[],version:"current",sidebarPosition:20,frontMatter:{title:"Quickstart with Minder (< 1 min)",sidebar_position:20},sidebar:"minder",previous:{title:"Install Minder CLI",permalink:"/getting_started/install_cli"},next:{title:"Logging in to Minder",permalink:"/getting_started/login"}},l={},u=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Quickstart",id:"quickstart",level:2},{value:"What&#39;s next?",id:"whats-next",level:2}],c={toc:u},p="wrapper";function d(e){let{components:t,...r}=e;return(0,i.kt)(p,(0,n.Z)({},c,r,{components:t,mdxType:"MDXLayout"}),(0,i.kt)("h1",{id:"quickstart-with-minder--1-min"},"Quickstart with Minder (< 1 min)"),(0,i.kt)("p",null,'Minder provides a "happy path" that guides you through the process of creating your first profile in Minder. In just a few seconds, you will register your repositories and enable secret scanning protection for all of them!'),(0,i.kt)("h2",{id:"prerequisites"},"Prerequisites"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},"A running Minder server, including a running KeyCloak installation"),(0,i.kt)("li",{parentName:"ul"},"A GitHub account"),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/getting_started/install_cli"},"The ",(0,i.kt)("inlineCode",{parentName:"a"},"minder")," CLI application")),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("a",{parentName:"li",href:"/getting_started/login"},"Logged in to Minder server"))),(0,i.kt)("h2",{id:"quickstart"},"Quickstart"),(0,i.kt)("p",null,"Now that you have installed your minder cli and have logged in to your Minder server, you can start using Minder!"),(0,i.kt)("p",null,"Minder has a ",(0,i.kt)("inlineCode",{parentName:"p"},"quickstart")," command which guides you through the process of creating your first profile.\nIn just a few seconds, you will register your repositories and enable secret scanning protection for all of them.\nTo do so, run:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-bash"},"minder quickstart\n")),(0,i.kt)("p",null,"This will prompt you to enroll your provider, select the repositories you'd like, create the ",(0,i.kt)("inlineCode",{parentName:"p"},"secret_scanning"),"\nrule type and create a profile which enables secret scanning for the selected repositories."),(0,i.kt)("p",null,"To see the status of your profile, run:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-bash"},"minder profile_status list --profile quickstart-profile --detailed\n")),(0,i.kt)("p",null,"You should see the overall profile status and a detailed view of the rule evaluation statuses for each of your registered repositories."),(0,i.kt)("p",null,"Minder will continue to keep track of your repositories and will ensure to fix any drifts from the desired state by\nusing the ",(0,i.kt)("inlineCode",{parentName:"p"},"remediate")," feature or alert you, if needed, using the ",(0,i.kt)("inlineCode",{parentName:"p"},"alert")," feature."),(0,i.kt)("p",null,"Congratulations! \ud83c\udf89 You've now successfully created your first profile!"),(0,i.kt)("h2",{id:"whats-next"},"What's next?"),(0,i.kt)("p",null,"You can now continue to explore Minder's features by adding or removing more repositories, create more profiles with\nvarious rules, and much more. There's a lot more to Minder than just secret scanning."),(0,i.kt)("p",null,"The ",(0,i.kt)("inlineCode",{parentName:"p"},"secret_scanning")," rule is just one of the many rule types that Minder supports."),(0,i.kt)("p",null,"You can see the full list of ready-to-use rules and profiles\nmaintained by Minder's team here - ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/stacklok/minder-rules-and-profiles"},"stacklok/minder-rules-and-profiles"),"."),(0,i.kt)("p",null,"In case there's something you don't find there yet, Minder is designed to be extensible.\nThis allows for users to create their own custom rule types and profiles and ensure the specifics of their security\nposture are attested to."),(0,i.kt)("p",null,"Now that you have everything set up, you can continue to run ",(0,i.kt)("inlineCode",{parentName:"p"},"minder")," commands against the public instance of Minder\nwhere you can manage your registered repositories, create profiles, rules and much more, so you can ensure your repositories are\nconfigured consistently and securely."),(0,i.kt)("p",null,"For more information about ",(0,i.kt)("inlineCode",{parentName:"p"},"minder"),", see:"),(0,i.kt)("ul",null,(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"minder")," CLI commands - ",(0,i.kt)("a",{parentName:"li",href:"https://minder-docs.stacklok.dev/ref/cli/minder"},"Docs"),"."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"minder")," REST API Documentation - ",(0,i.kt)("a",{parentName:"li",href:"https://minder-docs.stacklok.dev/ref/api"},"Docs"),"."),(0,i.kt)("li",{parentName:"ul"},(0,i.kt)("inlineCode",{parentName:"li"},"minder")," rules and profiles maintained by Minder's team - ",(0,i.kt)("a",{parentName:"li",href:"https://github.com/stacklok/minder-rules-and-profiles"},"GitHub"),"."),(0,i.kt)("li",{parentName:"ul"},"Minder documentation - ",(0,i.kt)("a",{parentName:"li",href:"https://minder-docs.stacklok.dev"},"Docs"),".")))}d.isMDXComponent=!0}}]);
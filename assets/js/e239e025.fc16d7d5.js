"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[4374],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>g});var r=n(67294);function i(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function a(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function l(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?a(Object(n),!0).forEach((function(t){i(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):a(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function o(e,t){if(null==e)return{};var n,r,i=function(e,t){if(null==e)return{};var n,r,i={},a=Object.keys(e);for(r=0;r<a.length;r++)n=a[r],t.indexOf(n)>=0||(i[n]=e[n]);return i}(e,t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(r=0;r<a.length;r++)n=a[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(i[n]=e[n])}return i}var s=r.createContext({}),c=function(e){var t=r.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):l(l({},t),e)),n},p=function(e){var t=c(e.components);return r.createElement(s.Provider,{value:t},e.children)},d="mdxType",u={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},m=r.forwardRef((function(e,t){var n=e.components,i=e.mdxType,a=e.originalType,s=e.parentName,p=o(e,["components","mdxType","originalType","parentName"]),d=c(n),m=i,g=d["".concat(s,".").concat(m)]||d[m]||u[m]||a;return n?r.createElement(g,l(l({ref:t},p),{},{components:n})):r.createElement(g,l({ref:t},p))}));function g(e,t){var n=arguments,i=t&&t.mdxType;if("string"==typeof e||i){var a=n.length,l=new Array(a);l[0]=m;var o={};for(var s in t)hasOwnProperty.call(t,s)&&(o[s]=t[s]);o.originalType=e,o[d]="string"==typeof e?e:i,l[1]=o;for(var c=2;c<a;c++)l[c]=n[c];return r.createElement.apply(null,l)}return r.createElement.apply(null,n)}m.displayName="MDXCreateElement"},12736:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>s,contentTitle:()=>l,default:()=>u,frontMatter:()=>a,metadata:()=>o,toc:()=>c});var r=n(87462),i=(n(67294),n(3905));const a={title:"Install Minder",sidebar_position:10},l="Installing the Minder CLI",o={unversionedId:"getting_started/install_cli",id:"getting_started/install_cli",title:"Install Minder",description:"Minder consists of two components: a server-side application, and the minder",source:"@site/docs/getting_started/install_cli.md",sourceDirName:"getting_started",slug:"/getting_started/install_cli",permalink:"/getting_started/install_cli",draft:!1,tags:[],version:"current",sidebarPosition:10,frontMatter:{title:"Install Minder",sidebar_position:10},sidebar:"minder",previous:{title:"Minder",permalink:"/"},next:{title:"Logging in to Minder",permalink:"/getting_started/login"}},s={},c=[{value:"MacOS (Homebrew)",id:"macos-homebrew",level:2},{value:"Windows (Winget)",id:"windows-winget",level:2},{value:"Linux",id:"linux",level:2},{value:"Building from source",id:"building-from-source",level:2}],p={toc:c},d="wrapper";function u(e){let{components:t,...n}=e;return(0,i.kt)(d,(0,r.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,i.kt)("h1",{id:"installing-the-minder-cli"},"Installing the Minder CLI"),(0,i.kt)("p",null,"Minder consists of two components: a server-side application, and the ",(0,i.kt)("inlineCode",{parentName:"p"},"minder"),"\nCLI application for interacting with the server.  Minder is built for ",(0,i.kt)("inlineCode",{parentName:"p"},"amd64"),"\nand ",(0,i.kt)("inlineCode",{parentName:"p"},"arm64")," architectures on Windows, MacOS, and Linux."),(0,i.kt)("p",null,"You can install ",(0,i.kt)("inlineCode",{parentName:"p"},"minder")," using one of the following methods:"),(0,i.kt)("h2",{id:"macos-homebrew"},"MacOS (Homebrew)"),(0,i.kt)("p",null,"The easiest way to install ",(0,i.kt)("inlineCode",{parentName:"p"},"minder")," is through ",(0,i.kt)("a",{parentName:"p",href:"https://brew.sh/"},"Homebrew"),":"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-bash"},"brew install stacklok/tap/minder\n")),(0,i.kt)("p",null,"Alternatively, you can ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/stacklok/minder/releases"},"download a ",(0,i.kt)("inlineCode",{parentName:"a"},".tar.gz")," release")," and unpack it with the following:"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-bash"},"tar -xzf minder_${RELEASE}_darwin_${ARCH}.tar.gz minder\nxattr -d com.apple.quarantine minder\n")),(0,i.kt)("h2",{id:"windows-winget"},"Windows (Winget)"),(0,i.kt)("p",null,"For Windows, the built-in ",(0,i.kt)("inlineCode",{parentName:"p"},"winget")," tool is the simplest way to install ",(0,i.kt)("inlineCode",{parentName:"p"},"minder"),":"),(0,i.kt)("pre",null,(0,i.kt)("code",{parentName:"pre",className:"language-bash"},"winget install stacklok.minder\n")),(0,i.kt)("p",null,"Alternatively, you can ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/stacklok/minder/releases"},"download a zipfile containing the ",(0,i.kt)("inlineCode",{parentName:"a"},"minder")," CLI")," and install the binary yourself."),(0,i.kt)("h2",{id:"linux"},"Linux"),(0,i.kt)("p",null,"We provide pre-built static binaries for Linux at: ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/stacklok/minder/releases"},"https://github.com/stacklok/minder/releases"),"."),(0,i.kt)("h2",{id:"building-from-source"},"Building from source"),(0,i.kt)("p",null,"You can also build the ",(0,i.kt)("inlineCode",{parentName:"p"},"minder")," CLI from source using ",(0,i.kt)("inlineCode",{parentName:"p"},"go install github.com/stacklok/minder/cmd/cli@latest"),", or by ",(0,i.kt)("a",{parentName:"p",href:"https://github.com/stacklok/minder#build-from-source"},"following the build instructions in the repository"),"."))}u.isMDXComponent=!0}}]);
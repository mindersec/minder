"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[792],{83836:(e,r,i)=>{i.r(r),i.d(r,{assets:()=>d,contentTitle:()=>s,default:()=>c,frontMatter:()=>n,metadata:()=>a,toc:()=>l});var t=i(74848),o=i(28453);const n={title:"Repository registration",sidebar_position:50},s=void 0,a={id:"understand/repository_registration",title:"Repository registration",description:"Registering a repository tells Minder to apply the profiles that you've defined to that repository. Minder will continuously monitor that repository based on the profiles that you've defined, and optionally alert you or automatically remediate the problem when the repository is out of compliance.",source:"@site/docs/understand/repository_registration.md",sourceDirName:"understand",slug:"/understand/repository_registration",permalink:"/understand/repository_registration",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:50,frontMatter:{title:"Repository registration",sidebar_position:50},sidebar:"minder",previous:{title:"Alerting",permalink:"/understand/alerts"},next:{title:"Automatic remediations",permalink:"/understand/remediations"}},d={},l=[{value:"Registering repositories",id:"registering-repositories",level:2},{value:"Automatically registering new repositories",id:"automatically-registering-new-repositories",level:2},{value:"List and get Repositories",id:"list-and-get-repositories",level:2},{value:"Removing a registered repository",id:"removing-a-registered-repository",level:2}];function p(e){const r={a:"a",admonition:"admonition",code:"code",em:"em",h2:"h2",p:"p",pre:"pre",...(0,o.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsxs)(r.p,{children:[(0,t.jsx)(r.em,{children:"Registering a repository"})," tells Minder to apply the ",(0,t.jsx)(r.a,{href:"/understand/profiles",children:"profiles"})," that you've defined to that repository. Minder will continuously monitor that repository based on the profiles that you've defined, and optionally ",(0,t.jsx)(r.a,{href:"/understand/alerts",children:"alert you"})," or ",(0,t.jsx)(r.a,{href:"/understand/remediations",children:"automatically remediate the problem"})," when the repository is out of compliance."]}),"\n",(0,t.jsx)(r.h2,{id:"registering-repositories",children:"Registering repositories"}),"\n",(0,t.jsxs)(r.p,{children:["Once you have ",(0,t.jsx)(r.a,{href:"/understand/providers",children:"enrolled the GitHub Provider"}),", you can register repositories that you granted Minder access to within GitHub."]}),"\n",(0,t.jsx)(r.p,{children:"To get a list of repositories, and select them using a menu in Minder's text user interface, run:"}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder repo register\n"})}),"\n",(0,t.jsx)(r.p,{children:"You can also register an individual repository by name, or a set of repositories, comma-separated. For example:"}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:'minder repo register --name "owner/repo1,owner/repo2"\n'})}),"\n",(0,t.jsx)(r.p,{children:"After registering repositories, Minder will begin applying your existing profiles to those repositories and will identify repositories that are out of compliance with your security profiles."}),"\n",(0,t.jsx)(r.p,{children:"In addition, Minder will set up a webhook in each repository that was registered. This allows Minder to identify when configuration changes are made to your repositories and re-scan them for compliance with your profiles."}),"\n",(0,t.jsx)(r.h2,{id:"automatically-registering-new-repositories",children:"Automatically registering new repositories"}),"\n",(0,t.jsx)(r.p,{children:"The GitHub Provider can be configured to automatically register new repositories that are created in your organization. This is done by setting an attribute on the provider."}),"\n",(0,t.jsxs)(r.p,{children:["First, identify the ",(0,t.jsx)(r.em,{children:"name"})," of your GitHub Provider. You can list your enrolled providers by running:"]}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder provider list\n"})}),"\n",(0,t.jsxs)(r.p,{children:["To enable automatic registration for your repositories, set the ",(0,t.jsx)(r.code,{children:"auto_registration.entities.repository.enabled"})," attribute to ",(0,t.jsx)(r.code,{children:"true"})," for your provider. For example, if your provider was named ",(0,t.jsx)(r.code,{children:"github-app-myorg"}),", run:"]}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder provider update --set-attribute=auto_registration.entities.repository.enabled=true --name=github-app-myorg\n"})}),"\n",(0,t.jsx)(r.admonition,{type:"note",children:(0,t.jsx)(r.p,{children:"Enabling automatic registration only applies to new repositories that are created in your organization, it does not retroactively register existing repositories."})}),"\n",(0,t.jsxs)(r.p,{children:["To disable automatic registration, set the ",(0,t.jsx)(r.code,{children:"auto_registration.entities.repository.enabled"})," attribute to ",(0,t.jsx)(r.code,{children:"false"}),":"]}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder provider update --set-attribute=auto_registration.entities.repository.enabled=false --name=github-app-myorg\n"})}),"\n",(0,t.jsx)(r.admonition,{type:"note",children:(0,t.jsx)(r.p,{children:"Disabling automatic registration will not remove the repositories that have already been registered."})}),"\n",(0,t.jsx)(r.h2,{id:"list-and-get-repositories",children:"List and get Repositories"}),"\n",(0,t.jsx)(r.p,{children:"You can list all repositories registered in Minder:"}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder repo list\n"})}),"\n",(0,t.jsxs)(r.p,{children:["You can also get detailed information about a specific repository. For example, to view the information for ",(0,t.jsx)(r.code,{children:"owner/repo1"}),", run:"]}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:"minder repo get --name owner/repo1\n"})}),"\n",(0,t.jsx)(r.h2,{id:"removing-a-registered-repository",children:"Removing a registered repository"}),"\n",(0,t.jsxs)(r.p,{children:["If you want to stop monitoring a repository, you can remove it from Minder by using the ",(0,t.jsx)(r.code,{children:"repo delete"})," command:"]}),"\n",(0,t.jsx)(r.pre,{children:(0,t.jsx)(r.code,{className:"language-bash",children:'minder repo delete --name "owner/repo1"\n'})}),"\n",(0,t.jsx)(r.p,{children:"This will remove the repository configuration from Minder and remove the webhook from the GitHub repository."})]})}function c(e={}){const{wrapper:r}={...(0,o.R)(),...e.components};return r?(0,t.jsx)(r,{...e,children:(0,t.jsx)(p,{...e})}):p(e)}},28453:(e,r,i)=>{i.d(r,{R:()=>s,x:()=>a});var t=i(96540);const o={},n=t.createContext(o);function s(e){const r=t.useContext(n);return t.useMemo((function(){return"function"==typeof e?e(r):{...r,...e}}),[r,e])}function a(e){let r;return r=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:s(e.components),t.createElement(n.Provider,{value:r},e.children)}}}]);
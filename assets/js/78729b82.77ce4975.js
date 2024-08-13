"use strict";(self.webpackChunkminder_docs=self.webpackChunkminder_docs||[]).push([[106],{48065:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>l,contentTitle:()=>o,default:()=>h,frontMatter:()=>i,metadata:()=>s,toc:()=>c});var a=n(74848),r=n(28453);const i={title:"Feature flags",sidebar_position:20},o="Using Feature Flags",s={id:"developer_guide/feature_flags",title:"Feature flags",description:"Minder is using OpenFeature for feature flags.  For more complex configuration, refer to that documentation.  With that said, our goals are to allow for simple, straightforward usage of feature flags to allow merging code which is complete before the entire feature is complete.",source:"@site/docs/developer_guide/feature_flags.md",sourceDirName:"developer_guide",slug:"/developer_guide/feature_flags",permalink:"/developer_guide/feature_flags",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:20,frontMatter:{title:"Feature flags",sidebar_position:20},sidebar:"minder",previous:{title:"Get hacking",permalink:"/developer_guide/get-hacking"},next:{title:"Architecture overview",permalink:"/developer_guide/architecture"}},l={},c=[{value:"When to use feature flags",id:"when-to-use-feature-flags",level:2},{value:"Inappropriate Use Of Feature Flags",id:"inappropriate-use-of-feature-flags",level:3},{value:"How to Use Feature Flags",id:"how-to-use-feature-flags",level:2},{value:"Using Flags During Development",id:"using-flags-during-development",level:2}];function d(e){const t={a:"a",code:"code",em:"em",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",strong:"strong",ul:"ul",...(0,r.R)(),...e.components};return(0,a.jsxs)(a.Fragment,{children:[(0,a.jsx)(t.header,{children:(0,a.jsx)(t.h1,{id:"using-feature-flags",children:"Using Feature Flags"})}),"\n",(0,a.jsxs)(t.p,{children:["Minder is using ",(0,a.jsx)(t.a,{href:"https://openfeature.dev/",children:"OpenFeature"})," for feature flags.  For more complex configuration, refer to that documentation.  With that said, our goals are to allow for ",(0,a.jsx)(t.em,{children:"simple, straightforward"})," usage of feature flags to ",(0,a.jsx)(t.strong,{children:"allow merging code which is complete before the entire feature is complete"}),"."]}),"\n",(0,a.jsx)(t.h2,{id:"when-to-use-feature-flags",children:"When to use feature flags"}),"\n",(0,a.jsx)(t.p,{children:"Appropriate usages of feature flags:"}),"\n",(0,a.jsxs)(t.ul,{children:["\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:[(0,a.jsx)(t.strong,{children:"Stage Changes"}),".  Use a feature flag to (for example) add the ability to write values in one PR, and the ability to operate on them in another PR.  By putting all the functionality behind a feature flag, it can be released all at once (when the documentation is complete).  Depending on the functionality, this may also be used to achieve a ",(0,a.jsx)(t.strong,{children:"staged rollout"})," across a larger population of users, starting with people willing to beta-test the feature."]}),"\n"]}),"\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:[(0,a.jsx)(t.strong,{children:"Kill Switch"}),".  For features which introduce new load (e.g. 10x GitHub API token usage) or new access patterns (e.g. change message durability), feature flags can provide a quick way to be able to enable or revert changes without needing to build and push a new binary or config option (particularly if other code has changed in the meantime).  In this case, feature flags provide a consistent way of managing configuration as an alternative to ",(0,a.jsx)(t.code,{children:"internal/config/server"}),".  Note that ",(0,a.jsx)(t.em,{children:"feature flags"})," affect a particular invocation (based on the user or project in question), while ",(0,a.jsx)(t.em,{children:"config"})," generally affects all behavior of the server."]}),"\n"]}),"\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:[(0,a.jsx)(t.strong,{children:"Feature acceptance testing"})," (A/B testing).  When running Minder as a service, the Stacklok team may want to perform large-scale evaluation of whether a feature is useful to end-users.  Feature flags can allow comparing the usage of two groups with and without the feature enabled."]}),"\n"]}),"\n"]}),"\n",(0,a.jsx)(t.h3,{id:"inappropriate-use-of-feature-flags",children:"Inappropriate Use Of Feature Flags"}),"\n",(0,a.jsx)(t.p,{children:'We expect that feature flags will generally be short-lived (a few months in most cases).  There are costs (testing, maintenance, complexity, and general opportunity costs) to maintaining two code paths, so we aim to retire feature flags once the feature is considered "stable".  Here are some examples of alternative mechanisms to use for long-term behavior changes:'}),"\n",(0,a.jsxs)(t.ul,{children:["\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:[(0,a.jsx)(t.strong,{children:"Server Configuration"}),".  See ",(0,a.jsx)(t.a,{href:"https://github.com/stacklok/minder/tree/main/internal/config/server",children:(0,a.jsx)(t.code,{children:"internal/config/server"})})," for long-term options that should be on or off at server startup and don't need to change based on the invocation."]}),"\n"]}),"\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:[(0,a.jsx)(t.strong,{children:"Entitlements"}),".  See ",(0,a.jsx)(t.a,{href:"https://github.com/stacklok/minder/tree/main/internal/projects/features",children:(0,a.jsx)(t.code,{children:"internal/projects/features"})})," for functionality that should be able to be turned on or off on a per-project basis (for example, for paid customers)."]}),"\n"]}),"\n"]}),"\n",(0,a.jsx)(t.h2,{id:"how-to-use-feature-flags",children:"How to Use Feature Flags"}),"\n",(0,a.jsxs)(t.p,{children:["If you're working on a new Minder feature and want to merge it incrementally, check out ",(0,a.jsx)(t.a,{href:"https://github.com/stacklok/minder/blob/d8f7d5709540bd33a2200adc2dbd330bbeceae86/internal/controlplane/handlers_authz.go#L222",children:"this code (linked to commit)"})," for an example.  The process is basically:"]}),"\n",(0,a.jsxs)(t.ol,{children:["\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:["Add a feature flag declaration to ",(0,a.jsx)(t.a,{href:"https://github.com/stacklok/minder/blob/main/internal/flags/constants.go",children:(0,a.jsx)(t.code,{children:"internal/flags/constants.go"})})]}),"\n"]}),"\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:["At the call site(s), put the new functionality behind ",(0,a.jsx)(t.code,{children:"if flags.Bool(ctx, s.featureFlags, flags.MyFlagName) {..."})]}),"\n"]}),"\n",(0,a.jsxs)(t.li,{children:["\n",(0,a.jsxs)(t.p,{children:["You can use the ",(0,a.jsx)(t.a,{href:"https://github.com/stacklok/minder/blob/main/internal/flags/test_client.go",children:(0,a.jsx)(t.code,{children:"flags.FakeClient"})})," in tests to test the new code path as well as the old one."]}),"\n"]}),"\n"]}),"\n",(0,a.jsxs)(t.p,{children:["Using ",(0,a.jsx)(t.code,{children:"flags.Bool"})," from our own repo will enable a couple bits of default behavior over OpenFeature:"]}),"\n",(0,a.jsxs)(t.ul,{children:["\n",(0,a.jsxs)(t.li,{children:['We enforce that the default value of the flag is "off", so you can\'t end up with the confusing ',(0,a.jsx)(t.code,{children:"disable_feature=false"})," in a config."]}),"\n",(0,a.jsxs)(t.li,{children:["We extract the user, project, and provider from ",(0,a.jsx)(t.code,{children:"ctx"}),", so you don't need to."]}),"\n",(0,a.jsx)(t.li,{children:"Eventually, we'll also record the flag settings in our telemetry records (WIP)"}),"\n"]}),"\n",(0,a.jsx)(t.h2,{id:"using-flags-during-development",children:"Using Flags During Development"}),"\n",(0,a.jsxs)(t.p,{children:["You can create a ",(0,a.jsx)(t.code,{children:"flags-config.yaml"})," in the root Minder directory when running with ",(0,a.jsx)(t.code,{children:"make run-docker"}),", and the file (and future changes) will be mapped into the Minder container, so you can make changes live.  The ",(0,a.jsx)(t.code,{children:"flags-config.yaml"})," uses the ",(0,a.jsx)(t.a,{href:"https://gofeatureflag.org/docs/configure_flag/flag_format",children:"GoFeatureFlag format"}),", and is in the repo's ",(0,a.jsx)(t.code,{children:".gitignore"}),", so you don't need to worry about accidentally checking it in.  Note that the Minder server currently rechecks the flag configuration once a minute, so it may take a minute or two for flags changes to be visible."]}),"\n",(0,a.jsxs)(t.p,{children:["When deploying as a Helm chart, you can create a ConfigMap named ",(0,a.jsx)(t.code,{children:"minder-flags"})," containing a key ",(0,a.jsx)(t.code,{children:"flags-config.yaml"}),", and it will be mounted into the container.  Again, changes to the ",(0,a.jsx)(t.code,{children:"minder-flags"})," ConfigMap will be updated in the Minder server within about 2 minutes of update."]})]})}function h(e={}){const{wrapper:t}={...(0,r.R)(),...e.components};return t?(0,a.jsx)(t,{...e,children:(0,a.jsx)(d,{...e})}):d(e)}},28453:(e,t,n)=>{n.d(t,{R:()=>o,x:()=>s});var a=n(96540);const r={},i=a.createContext(r);function o(e){const t=a.useContext(i);return a.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function s(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:o(e.components),a.createElement(i.Provider,{value:t},e.children)}}}]);
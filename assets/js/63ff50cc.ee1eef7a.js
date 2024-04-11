"use strict";(self.webpackChunkstacklok=self.webpackChunkstacklok||[]).push([[3015],{29019:(e,n,i)=>{i.r(n),i.d(n,{assets:()=>a,contentTitle:()=>t,default:()=>h,frontMatter:()=>s,metadata:()=>l,toc:()=>c});var r=i(74848),o=i(28453);const s={title:"Run the Server",sidebar_position:10},t="Run a minder server",l={id:"run_minder_server/run_the_server",title:"Run the Server",description:"Minder is platform, comprising of a controlplane, a CLI, a database and an identity provider.",source:"@site/docs/run_minder_server/run_the_server.md",sourceDirName:"run_minder_server",slug:"/run_minder_server/run_the_server",permalink:"/run_minder_server/run_the_server",draft:!1,unlisted:!1,tags:[],version:"current",sidebarPosition:10,frontMatter:{title:"Run the Server",sidebar_position:10},sidebar:"minder",previous:{title:"Adding users to your project",permalink:"/how-to/add_users_to_project"},next:{title:"Configure GitHub Provider",permalink:"/run_minder_server/config_oauth"}},a={},c=[{value:"Prerequisites",id:"prerequisites",level:2},{value:"Download the latest release",id:"download-the-latest-release",level:2},{value:"Build from source",id:"build-from-source",level:2},{value:"Clone the repository",id:"clone-the-repository",level:3},{value:"Build the application",id:"build-the-application",level:3},{value:"OpenFGA",id:"openfga",level:2},{value:"Using a container",id:"using-a-container",level:3},{value:"Database creation",id:"database-creation",level:2},{value:"Using a container",id:"using-a-container-1",level:3},{value:"Create the database and OpenFGA model",id:"create-the-database-and-openfga-model",level:3},{value:"Identity Provider",id:"identity-provider",level:2},{value:"Using a container",id:"using-a-container-2",level:3},{value:"Social login",id:"social-login",level:3},{value:"Create a GitHub OAuth Application for Social Login",id:"create-a-github-oauth-application-for-social-login",level:4},{value:"Enable GitHub login",id:"enable-github-login",level:4},{value:"Create token key passphrase",id:"create-token-key-passphrase",level:2},{value:"Configure the Repository Provider",id:"configure-the-repository-provider",level:2},{value:"Updating the Webhook Configuration",id:"updating-the-webhook-configuration",level:2},{value:"Run the application",id:"run-the-application",level:2}];function d(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",img:"img",li:"li",ol:"ol",p:"p",pre:"pre",strong:"strong",ul:"ul",...(0,o.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(n.h1,{id:"run-a-minder-server",children:"Run a minder server"}),"\n",(0,r.jsx)(n.p,{children:"Minder is platform, comprising of a controlplane, a CLI, a database and an identity provider."}),"\n",(0,r.jsx)(n.p,{children:"The control plane runs two endpoints, a gRPC endpoint and a HTTP endpoint."}),"\n",(0,r.jsxs)(n.p,{children:["Minder is controlled and managed via the CLI application ",(0,r.jsx)(n.code,{children:"minder"}),"."]}),"\n",(0,r.jsx)(n.p,{children:"PostgreSQL is used as the database."}),"\n",(0,r.jsx)(n.p,{children:"Keycloak is used as the identity provider."}),"\n",(0,r.jsxs)(n.p,{children:["There are two methods to get started with Minder, either by downloading the\nlatest release, building from source or (quickest) using the provided ",(0,r.jsx)(n.code,{children:"docker-compose.yaml"}),"\nfile."]}),"\n",(0,r.jsx)(n.h2,{id:"prerequisites",children:"Prerequisites"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.a,{href:"https://golang.org/doc/install",children:"Go 1.20"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.a,{href:"https://www.postgresql.org/download/",children:"PostgreSQL"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.a,{href:"https://www.keycloak.org/guides",children:"Keycloak"})}),"\n",(0,r.jsx)(n.li,{children:(0,r.jsx)(n.a,{href:"https://openfga.dev/#quick-start",children:"OpenFGA"})}),"\n"]}),"\n",(0,r.jsx)(n.h2,{id:"download-the-latest-release",children:"Download the latest release"}),"\n",(0,r.jsx)(n.p,{children:"[stub for when we cut a first release]"}),"\n",(0,r.jsx)(n.h2,{id:"build-from-source",children:"Build from source"}),"\n",(0,r.jsx)(n.p,{children:"Alternatively, you can build from source."}),"\n",(0,r.jsx)(n.h3,{id:"clone-the-repository",children:"Clone the repository"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"git clone git@github.com:stacklok/minder.git\n"})}),"\n",(0,r.jsx)(n.h3,{id:"build-the-application",children:"Build the application"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"make build\n"})}),"\n",(0,r.jsxs)(n.p,{children:["This will create two binaries, ",(0,r.jsx)(n.code,{children:"bin/minder-server"})," and ",(0,r.jsx)(n.code,{children:"bin/minder"}),"."]}),"\n",(0,r.jsxs)(n.p,{children:["You may now copy these into a location on your path, or run them directly from the ",(0,r.jsx)(n.code,{children:"bin"})," directory."]}),"\n",(0,r.jsxs)(n.p,{children:["You will also need a configuration file. You can copy the example configuration file from ",(0,r.jsx)(n.code,{children:"configs/server-config.yaml.example"})," to ",(0,r.jsx)(n.code,{children:"$(PWD)/server-config.yaml"}),"."]}),"\n",(0,r.jsxs)(n.p,{children:["If you prefer to use a different file name or location, you can specify this using the ",(0,r.jsx)(n.code,{children:"--config"}),"\nflag, e.g. ",(0,r.jsx)(n.code,{children:"minder-server --config /file/path/server-config.yaml serve"})," when you later run the application."]}),"\n",(0,r.jsx)(n.h2,{id:"openfga",children:"OpenFGA"}),"\n",(0,r.jsx)(n.p,{children:"Minder requires a OpenFGA instance to be running. You can install this locally, or use a container."}),"\n",(0,r.jsxs)(n.p,{children:["Should you install locally, you will need to set certain configuration options in your ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file, to reflect your local OpenFGA configuration."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"authz:\n   api_url: http://localhost:8082\n   store_name: minder\n   auth:\n      # Set to token for production\n      method: none\n"})}),"\n",(0,r.jsx)(n.h3,{id:"using-a-container",children:"Using a container"}),"\n",(0,r.jsxs)(n.p,{children:["A simple way to get started is to use the provided ",(0,r.jsx)(n.code,{children:"docker-compose.yaml"})," file."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker compose up -d openfga\n"})}),"\n",(0,r.jsx)(n.h2,{id:"database-creation",children:"Database creation"}),"\n",(0,r.jsx)(n.p,{children:"Minder requires a PostgreSQL database to be running. You can install this locally, or use a container."}),"\n",(0,r.jsxs)(n.p,{children:["Should you install locally, you will need to set certain configuration options in your ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file, to reflect your local database configuration."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'database:\n  dbhost: "localhost"\n  dbport: 5432\n  dbuser: postgres\n  dbpass: postgres\n  dbname: minder\n  sslmode: disable\n'})}),"\n",(0,r.jsx)(n.h3,{id:"using-a-container-1",children:"Using a container"}),"\n",(0,r.jsxs)(n.p,{children:["A simple way to get started is to use the provided ",(0,r.jsx)(n.code,{children:"docker-compose.yaml"})," file."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker compose up -d postgres\n"})}),"\n",(0,r.jsx)(n.h3,{id:"create-the-database-and-openfga-model",children:"Create the database and OpenFGA model"}),"\n",(0,r.jsxs)(n.p,{children:["Once you have a running database and OpenFGA instance, you can create the\ndatabase and OpenFGA model using the ",(0,r.jsx)(n.code,{children:"minder-server"})," CLI tool or via the ",(0,r.jsx)(n.code,{children:"make"}),"\ncommand."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"make migrateup\n"})}),"\n",(0,r.jsx)(n.p,{children:"or:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"minder-server migrate up\n"})}),"\n",(0,r.jsx)(n.h2,{id:"identity-provider",children:"Identity Provider"}),"\n",(0,r.jsx)(n.p,{children:"Minder requires a Keycloak instance to be running. You can install this locally, or use a container."}),"\n",(0,r.jsx)(n.p,{children:"Should you install locally, you will need to configure the client on Keycloak.\nYou will need the following:"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsx)(n.li,{children:'A Keycloak realm named "stacklok" with event saving turned on for the "Delete account" event.'}),"\n",(0,r.jsxs)(n.li,{children:["A registered public client with the redirect URI ",(0,r.jsx)(n.code,{children:"http://localhost/*"}),". This is used for the minder CLI."]}),"\n",(0,r.jsx)(n.li,{children:"A registered confidential client with a service account that can manage users and view events. This is used for the minder server."}),"\n"]}),"\n",(0,r.jsxs)(n.p,{children:["You will also need to set certain configuration options in your ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file, to reflect your local Keycloak configuration."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"identity:\n  server:\n    issuer_url: http://localhost:8081\n    client_id: minder-server\n    client_secret: secret\n"})}),"\n",(0,r.jsxs)(n.p,{children:["Similarly, for the CLI ",(0,r.jsx)(n.code,{children:"config.yaml"}),"."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:"identity:\n  cli:\n    issuer_url: http://localhost:8081\n    client_id: minder-cli\n"})}),"\n",(0,r.jsx)(n.h3,{id:"using-a-container-2",children:"Using a container"}),"\n",(0,r.jsxs)(n.p,{children:["A simple way to get started is to use the provided ",(0,r.jsx)(n.code,{children:"docker-compose.yaml"})," file."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker compose up -d keycloak\n"})}),"\n",(0,r.jsx)(n.h3,{id:"social-login",children:"Social login"}),"\n",(0,r.jsx)(n.p,{children:"Once you have a Keycloak instance running locally, you can set up GitHub authentication."}),"\n",(0,r.jsx)(n.h4,{id:"create-a-github-oauth-application-for-social-login",children:"Create a GitHub OAuth Application for Social Login"}),"\n",(0,r.jsxs)(n.ol,{children:["\n",(0,r.jsxs)(n.li,{children:["Navigate to ",(0,r.jsx)(n.a,{href:"https://github.com/settings/profile",children:"GitHub Developer Settings"})]}),"\n",(0,r.jsx)(n.li,{children:'Select "Developer Settings" from the left hand menu'}),"\n",(0,r.jsx)(n.li,{children:'Select "OAuth Apps" from the left hand menu'}),"\n",(0,r.jsx)(n.li,{children:'Select "New OAuth App"'}),"\n",(0,r.jsxs)(n.li,{children:["Enter the following details:","\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:["Application Name: ",(0,r.jsx)(n.code,{children:"Stacklok Identity Provider"})," (or any other name you like)"]}),"\n",(0,r.jsxs)(n.li,{children:["Homepage URL: ",(0,r.jsx)(n.code,{children:"http://localhost:8081"})," or the URL you specified as the ",(0,r.jsx)(n.code,{children:"issuer_url"})," in your ",(0,r.jsx)(n.code,{children:"server-config.yaml"})]}),"\n",(0,r.jsxs)(n.li,{children:["Authorization callback URL: ",(0,r.jsx)(n.code,{children:"http://localhost:8081/realms/stacklok/broker/github/endpoint"})]}),"\n"]}),"\n"]}),"\n",(0,r.jsx)(n.li,{children:'Select "Register Application"'}),"\n",(0,r.jsx)(n.li,{children:"Generate a client secret"}),"\n"]}),"\n",(0,r.jsx)(n.p,{children:(0,r.jsx)(n.img,{alt:"github oauth2 page",src:i(37842).A+"",width:"1282",height:"2402"})}),"\n",(0,r.jsx)(n.h4,{id:"enable-github-login",children:"Enable GitHub login"}),"\n",(0,r.jsx)(n.p,{children:"Using the client ID and client secret you created above, enable GitHub login your local Keycloak instance by running the\nfollowing command:"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"make KC_GITHUB_CLIENT_ID=<client_id> KC_GITHUB_CLIENT_SECRET=<client_secret> github-login\n"})}),"\n",(0,r.jsx)(n.h2,{id:"create-token-key-passphrase",children:"Create token key passphrase"}),"\n",(0,r.jsx)(n.p,{children:"Create a token key passphrase that is used when storing the provider's token in the database."}),"\n",(0,r.jsxs)(n.p,{children:["The default configuration expects these keys to be in a directory named ",(0,r.jsx)(n.code,{children:".ssh"}),", relative to where you run the ",(0,r.jsx)(n.code,{children:"minder-server"})," binary.\nStart by creating the ",(0,r.jsx)(n.code,{children:".ssh"})," directory."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"mkdir .ssh\n"})}),"\n",(0,r.jsxs)(n.p,{children:["You can create the passphrase using the ",(0,r.jsx)(n.code,{children:"openssl"})," CLI tool."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"openssl rand -base64 32 > .ssh/token_key_passphrase\n"})}),"\n",(0,r.jsxs)(n.p,{children:["If your key lives in a directory other than ",(0,r.jsx)(n.code,{children:".ssh"}),", you can specify the location of the key in the ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'auth:\n   token_key: "./.ssh/token_key_passphrase"\n'})}),"\n",(0,r.jsx)(n.h2,{id:"configure-the-repository-provider",children:"Configure the Repository Provider"}),"\n",(0,r.jsx)(n.p,{children:"At this point, you should have the following:"}),"\n",(0,r.jsxs)(n.ul,{children:["\n",(0,r.jsxs)(n.li,{children:["A running PostgreSQL database, with the ",(0,r.jsx)(n.code,{children:"minder"})," database created"]}),"\n",(0,r.jsx)(n.li,{children:"A running Keycloak instance"}),"\n",(0,r.jsx)(n.li,{children:"A GitHub OAuth application configured for social login using Keycloak"}),"\n"]}),"\n",(0,r.jsxs)(n.p,{children:[(0,r.jsx)(n.strong,{children:"Prior to running the application"}),", you need to configure your repository provider. Currently, Minder only supports GitHub.\nSee ",(0,r.jsx)(n.a,{href:"/run_minder_server/config_oauth",children:"Configure Repository Provider"})," for more information."]}),"\n",(0,r.jsx)(n.h2,{id:"updating-the-webhook-configuration",children:"Updating the Webhook Configuration"}),"\n",(0,r.jsxs)(n.p,{children:["Minder requires a webhook to be configured on the repository provider. Currently, Minder only supports GitHub.\nThe webhook allows GitHub to notify Minder when certain events occur in your repositories.\nTo configure the webhook, Minder needs to be accessible from the internet. If you are running the server locally, you\ncan use a service like ",(0,r.jsx)(n.a,{href:"https://ngrok.com/",children:"ngrok"})," to expose your local server to the internet."]}),"\n",(0,r.jsx)(n.p,{children:"Here are the steps to configure the webhook:"}),"\n",(0,r.jsxs)(n.ol,{children:["\n",(0,r.jsxs)(n.li,{children:["\n",(0,r.jsxs)(n.p,{children:[(0,r.jsx)(n.strong,{children:"Expose your local server:"})," If you are running the server locally, start ngrok or a similar service to expose your\nlocal server to the internet. Note down the URL provided by ngrok (it will look something like ",(0,r.jsx)(n.code,{children:"https://<random-hash>.ngrok.io"}),").\nMake sure to expose the port that Minder is running on (by default, this is port ",(0,r.jsx)(n.code,{children:"8080"}),")."]}),"\n"]}),"\n",(0,r.jsxs)(n.li,{children:["\n",(0,r.jsxs)(n.p,{children:[(0,r.jsx)(n.strong,{children:"Update the Minder configuration:"})," Open your ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file and update the ",(0,r.jsx)(n.code,{children:"webhook-config"})," section with\nthe ngrok URL Minder is running on. The ",(0,r.jsx)(n.code,{children:"external_webhook_url"})," should point to the ",(0,r.jsx)(n.code,{children:"/api/v1/webhook/github"}),"\nendpoint on your Minder server, and the ",(0,r.jsx)(n.code,{children:"external_ping_url"})," should point to the ",(0,r.jsx)(n.code,{children:"/api/v1/health"})," endpoint. The ",(0,r.jsx)(n.code,{children:"webhook_secret"}),"\nshould match the secret configured in the GitHub webhook (under ",(0,r.jsx)(n.code,{children:"github.payload_secret"}),")."]}),"\n"]}),"\n"]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'webhook-config:\n    external_webhook_url: "https://<ngrok-url>/api/v1/webhook/github"\n    external_ping_url: "https://<ngrok-url>/api/v1/health"\n    webhook_secret: "your-password" # Should match the secret configured in the GitHub webhook (github.payload_secret)\n'})}),"\n",(0,r.jsx)(n.p,{children:"After these steps, your Minder server should be ready to receive webhook events from GitHub, and add webhooks to repositories."}),"\n",(0,r.jsxs)(n.p,{children:["In case you need to update the webhook secret, you can do so by putting the\nnew secret in ",(0,r.jsx)(n.code,{children:"webhook-config.webhook_secret"})," and for the duration of the\nmigration, the old secret(s) in a file referenced by\n",(0,r.jsx)(n.code,{children:"webhook-config.previous_webhook_secret_file"}),". The old webhook secrets will\nthen only be used to verify incoming webhooks messages, not for creating or\nupdating webhooks and can be removed after the migration is complete."]}),"\n",(0,r.jsxs)(n.p,{children:["In order to rotate webhook secrets, you can use the ",(0,r.jsx)(n.code,{children:"minder-server"})," CLI tool to update the webhook secret."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"minder-server webhook update -p github\n"})}),"\n",(0,r.jsx)(n.p,{children:"Note that the command simply replaces the webhook secret on the provider\nside. You will still need to update the webhook secret in the server configuration\nto match the provider's secret."}),"\n",(0,r.jsx)(n.h2,{id:"run-the-application",children:"Run the application"}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"minder-server serve\n"})}),"\n",(0,r.jsxs)(n.p,{children:["If the application is configured using ",(0,r.jsx)(n.code,{children:"docker compose"}),", you need to modify the ",(0,r.jsx)(n.code,{children:"server-config.yaml"})," file to reflect the database host url."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-yaml",children:'database:\n  dbhost: "postgres" # Changed from localhost to postgres\n  dbport: 5432\n  dbuser: postgres\n  dbpass: postgres\n  dbname: minder\n  sslmode: disable\n'})}),"\n",(0,r.jsxs)(n.p,{children:["After configuring ",(0,r.jsx)(n.code,{children:"server-config.yaml"}),", you can run the application using ",(0,r.jsx)(n.code,{children:"docker compose"}),"."]}),"\n",(0,r.jsx)(n.pre,{children:(0,r.jsx)(n.code,{className:"language-bash",children:"docker compose up -d minder\n"})}),"\n",(0,r.jsxs)(n.p,{children:["The application will be available on ",(0,r.jsx)(n.code,{children:"http://localhost:8080"})," and gRPC on ",(0,r.jsx)(n.code,{children:"localhost:8090"}),"."]})]})}function h(e={}){const{wrapper:n}={...(0,o.R)(),...e.components};return n?(0,r.jsx)(n,{...e,children:(0,r.jsx)(d,{...e})}):d(e)}},37842:(e,n,i)=>{i.d(n,{A:()=>r});const r=i.p+"assets/images/minder-social-login-github-bbd3fc6f7764a859d6d8a637ca834d08.png"},28453:(e,n,i)=>{i.d(n,{R:()=>t,x:()=>l});var r=i(96540);const o={},s=r.createContext(o);function t(e){const n=r.useContext(s);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function l(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(o):e.components||o:t(e.components),r.createElement(s.Provider,{value:n},e.children)}}}]);
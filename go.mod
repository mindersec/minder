module github.com/stacklok/minder

go 1.21.5

toolchain go1.21.6

require (
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/ThreeDotsLabs/watermill v1.3.5
	github.com/ThreeDotsLabs/watermill-sql/v2 v2.0.0
	github.com/alexdrl/zerowater v0.0.3
	github.com/barkimedes/go-deepcopy v0.0.0-20220514131651-17c30cfc62df
	github.com/cenkalti/backoff/v4 v4.2.1
	github.com/charmbracelet/lipgloss v0.9.1
	github.com/erikgeiser/promptkit v0.9.0
	github.com/evanphx/json-patch/v5 v5.9.0
	github.com/fergusstrange/embedded-postgres v1.25.0
	github.com/go-git/go-billy/v5 v5.5.0
	github.com/go-git/go-git/v5 v5.11.0
	github.com/go-playground/validator/v10 v10.18.0
	github.com/goccy/go-json v0.10.2
	github.com/golang-migrate/migrate/v4 v4.17.0
	github.com/golang/mock v1.6.0
	github.com/google/go-containerregistry v0.19.0
	github.com/google/go-github/v56 v56.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/securecookie v1.1.2
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.1
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1
	github.com/hashicorp/go-version v1.6.0
	github.com/itchyny/gojq v0.12.14
	github.com/lib/pq v1.10.9
	github.com/mitchellh/mapstructure v1.5.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/open-policy-agent/opa v0.61.0
	github.com/openfga/go-sdk v0.3.5
	github.com/openfga/openfga v1.4.3
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/prometheus/client_golang v1.18.0
	github.com/puzpuzpuz/xsync/v3 v3.0.2
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.32.0
	github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql v1.13.0
	github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq v1.13.0
	github.com/sigstore/protobuf-specs v0.3.0
	github.com/sigstore/sigstore-go v0.2.0
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.18.2
	github.com/sqlc-dev/pqtype v0.3.0
	github.com/stacklok/frizbee v0.0.13
	github.com/stretchr/testify v1.8.4
	github.com/styrainc/regal v0.17.0
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.48.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.48.0
	go.opentelemetry.io/otel v1.23.1
	go.opentelemetry.io/otel/exporters/prometheus v0.45.2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.23.1
	go.opentelemetry.io/otel/sdk v1.23.1
	go.opentelemetry.io/otel/sdk/metric v1.23.1
	go.opentelemetry.io/otel/trace v1.23.1
	golang.org/x/crypto v0.19.0
	golang.org/x/exp v0.0.0-20240103183307-be819d1f06fc
	golang.org/x/oauth2 v0.17.0
	golang.org/x/sync v0.6.0
	golang.org/x/term v0.17.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240125205218-1f4bbc51befe
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.32.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/apimachinery v0.29.2
	k8s.io/client-go v0.29.2
)

require (
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/MicahParks/keyfunc v1.9.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/bubbles v0.17.1 // indirect
	github.com/charmbracelet/bubbletea v0.25.0 // indirect
	github.com/containerd/console v1.0.4-0.20230313162750-1ae8d489ac81 // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-chi/chi/v5 v5.0.10 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/google/cel-go v0.18.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/gorilla/schema v1.2.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/hashicorp/go-sockaddr v1.0.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.1 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jon-whit/go-grpc-prometheus v1.4.0 // indirect
	github.com/karlseguin/ccache/v3 v3.0.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.4 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/natefinch/wrap v0.2.0 // indirect
	github.com/oklog/ulid/v2 v2.1.0 // indirect
	github.com/openfga/api/proto v0.0.0-20231222042535-3037910c90c0 // indirect
	github.com/openfga/language/pkg/go v0.0.0-20231220180656-9f0ec45883f3 // indirect
	github.com/pressly/goose/v3 v3.17.0 // indirect
	github.com/puzpuzpuz/xsync v1.5.2 // indirect
	github.com/rs/cors v1.10.1 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/sethvargo/go-retry v0.2.4 // indirect
	github.com/signalfx/splunk-otel-go/instrumentation/internal v1.13.0 // indirect
	github.com/sigstore/sigstore v1.8.1 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/theupdateframework/go-tuf/v2 v2.0.0-20240207172116-f5cf71290141 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.21.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
)

require (
	cloud.google.com/go/compute v1.23.3 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	dario.cat/mergo v1.0.0
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230923063757-afb1ddc0824c // indirect
	github.com/agnivade/levenshtein v1.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.15.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/cyberphone/json-canonicalization v0.0.0-20231011164504-785e29786b46 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/digitorus/pkcs7 v0.0.0-20230818184609-3a137a874352 // indirect
	github.com/digitorus/timestamp v0.0.0-20231217203849-220c5c2851b7 // indirect
	github.com/docker/cli v24.0.7+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.22.0 // indirect
	github.com/go-openapi/errors v0.21.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/loads v0.21.5 // indirect
	github.com/go-openapi/runtime v0.27.1 // indirect
	github.com/go-openapi/spec v0.20.14 // indirect
	github.com/go-openapi/strfmt v0.22.0 // indirect
	github.com/go-openapi/swag v0.22.9 // indirect
	github.com/go-openapi/validate v0.22.6 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/certificate-transparency-go v1.1.7 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.5 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/in-toto/in-toto-golang v0.9.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jedisct1/go-minisign v0.0.0-20230811132847-661be99b8267 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/jwx/v2 v2.0.20
	github.com/letsencrypt/boulder v0.0.0-20231030195133-fdaab3e21ab0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.15.2 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc5 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sassoftware/relic v7.2.1+incompatible // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shibumi/go-pathspec v1.3.0 // indirect
	github.com/sigstore/rekor v1.3.5 // indirect
	github.com/sigstore/timestamp-authority v1.2.1 // indirect
	github.com/sirupsen/logrus v1.9.4-0.20230606125235-dd1b4c2e81af // indirect
	github.com/skeema/knownhosts v1.2.1 // indirect
	github.com/sony/gobreaker v0.5.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/stretchr/objx v0.5.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tchap/go-patricia/v2 v2.3.1 // indirect
	github.com/theupdateframework/go-tuf v0.7.0 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/transparency-dev/merkle v0.0.2 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.2.0 // indirect
	github.com/zitadel/oidc/v2 v2.12.0
	go.mongodb.org/mongo-driver v1.13.1 // indirect
	go.opentelemetry.io/otel/metric v1.23.1
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/mod v0.15.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20240123012728-ef4313101c80 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240125205218-1f4bbc51befe // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/klog/v2 v2.120.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.12 // indirect

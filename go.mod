module admin

replace github.com/armon/go-metrics => github.com/hashicorp/go-metrics v0.4.1

replace github.com/chnxq/xkitmod/config => ../xkitmod/config
replace github.com/chnxq/xkitpkg/config => ../xkitpkg/config
replace github.com/chnxq/xkitpkg/conf => ../xkitpkg/conf

go 1.26.0

require (
	entgo.io/ent v0.14.6
	github.com/chnxq/x-crud/api v0.0.0-20260411151944-a61448f9f7bc
	github.com/chnxq/x-crud/entgo v0.0.0-20260411151944-a61448f9f7bc
	github.com/chnxq/x-crud/viewer v0.0.0-20260411151944-a61448f9f7bc
	github.com/chnxq/x-swagger v0.0.0-20260529105209-02745c8a5170
	github.com/chnxq/x-utils v0.0.0-20260612100514-4160a415201a
	github.com/chnxq/x-utils/copierutil v0.0.0-20260612100514-4160a415201a
	github.com/chnxq/x-utils/geoip v0.0.0-20260612100514-4160a415201a
	github.com/chnxq/x-utils/mapper v0.0.0-20260612100514-4160a415201a
	github.com/chnxq/xkitmod v0.0.0-20260613041109-c180136c5f7e
	github.com/chnxq/xkitmod/algs v0.0.0-20260613041109-c180136c5f7e
	github.com/chnxq/xkitmod/log v0.0.0-20260613041109-c180136c5f7e
	github.com/chnxq/xkitpkg/app v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/cache v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/conf v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/config v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/config/consul v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/config/etcd v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/logger v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/logger/fluentd v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/logger/zap v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/middleware v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/registry v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/registry/consul v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/registry/etcd v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/tracer v0.0.0-20260613042622-08bb39f0c9d1
	github.com/chnxq/xkitpkg/transport v0.0.0-20260613042622-08bb39f0c9d1
	github.com/envoyproxy/protoc-gen-validate v1.3.3
	github.com/getkin/kin-openapi v0.140.0
	github.com/go-sql-driver/mysql v1.10.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/gnostic v0.7.1
	github.com/gorilla/handlers v1.5.2
	github.com/lib/pq v1.12.3
	github.com/mattn/go-sqlite3 v1.14.45
	github.com/menta2k/protoc-gen-redact/v3 v3.0.0-20260213125431-7688a38967d4
	github.com/minio/minio-go/v7 v7.2.0
	github.com/mojocn/base64Captcha v1.3.8
	github.com/robfig/cron/v3 v3.0.1
	golang.org/x/crypto v0.53.0
	google.golang.org/genproto/googleapis/api v0.0.0-20260610212136-7ab31c22f7ad
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.11
)

require (
	ariga.io/atlas v1.2.2 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260415201107-50325440f8f2.1 // indirect
	buf.build/go/protovalidate v1.2.0 // indirect
	cel.dev/expr v0.25.2 // indirect
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/XSAM/otelsql v0.42.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/armon/go-metrics v0.5.4 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/bwmarrin/snowflake v0.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chnxq/x-crud/audit v0.0.0-20260411151944-a61448f9f7bc // indirect
	github.com/chnxq/x-crud/pagination v0.0.0-20260411151944-a61448f9f7bc // indirect
	github.com/chnxq/x-utils/id v0.0.0-20260612100514-4160a415201a // indirect
	github.com/chnxq/xkitmod/config v0.0.0-20260613041109-c180136c5f7e // indirect
	github.com/chnxq/xkitmod/selector v0.0.0-20260613035154-3cb0d92f7857 // indirect
	github.com/chnxq/xkitpkg v0.0.0-20260613032609-8cf815dcaea2 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fluent/fluent-logger-golang v1.10.1 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/inflect v0.21.6 // indirect
	github.com/go-openapi/jsonpointer v0.23.1 // indirect
	github.com/go-openapi/swag/jsonname v0.26.1 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/go-test/deep v1.0.8 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.28.1 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	github.com/hashicorp/consul/api v1.34.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/serf v0.10.2 // indirect
	github.com/hibiken/asynq v0.26.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/lithammer/shortuuid/v4 v4.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20260330125221-c963978e514e // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/oasdiff/yaml v0.1.0 // indirect
	github.com/oasdiff/yaml3 v0.0.13 // indirect
	github.com/oschwald/geoip2-golang v1.13.0 // indirect
	github.com/oschwald/maxminddb-golang v1.13.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.20.1 // indirect
	github.com/redis/go-redis/extra/redisotel/v9 v9.20.1 // indirect
	github.com/redis/go-redis/v9 v9.20.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shoenig/go-m1cpu v0.2.1 // indirect
	github.com/sony/sonyflake v1.3.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/swaggest/swgui v1.8.8 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/vearutop/statigz v1.5.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zclconf/go-cty v1.18.1 // indirect
	github.com/zclconf/go-cty-yaml v1.2.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.einride.tech/aip v0.86.3 // indirect
	go.etcd.io/etcd/api/v3 v3.6.12 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.6.12 // indirect
	go.etcd.io/etcd/client/v3 v3.6.12 // indirect
	go.mongodb.org/mongo-driver/v2 v2.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.19.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0 // indirect
	go.opentelemetry.io/otel/log v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260611194520-c48552f49976 // indirect
	golang.org/x/image v0.42.0 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260610212136-7ab31c22f7ad // indirect
	gopkg.in/cenkalti/backoff.v1 v1.1.0 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

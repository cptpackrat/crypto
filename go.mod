module go.step.sm/crypto

go 1.18

require (
	cloud.google.com/go/kms v1.10.0
	filippo.io/edwards25519 v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.5.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.2.2
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys v0.9.0
	github.com/Masterminds/sprig/v3 v3.2.3
	github.com/ThalesIgnite/crypto11 v1.2.5
	github.com/aws/aws-sdk-go v1.44.240
	github.com/go-piv/piv-go v1.11.0
	github.com/golang/mock v1.6.0
	github.com/google/go-attestation v0.4.4-0.20220404204839-8820d49b18d9
	github.com/google/go-tpm v0.3.3
	github.com/google/go-tpm-tools v0.3.11
	github.com/googleapis/gax-go/v2 v2.8.0
	github.com/peterbourgon/diskv/v3 v3.0.1
	github.com/pkg/errors v0.9.1
	github.com/schollz/jsonstore v1.1.0
	github.com/smallstep/assert v0.0.0-20200723003110-82e2b9b3b262
	github.com/stretchr/testify v1.8.2
	golang.org/x/crypto v0.7.0
	golang.org/x/net v0.9.0
	golang.org/x/sys v0.7.0
	google.golang.org/api v0.117.0
	google.golang.org/grpc v1.54.0
	gopkg.in/square/go-jose.v2 v2.6.0
)

require (
	cloud.google.com/go/compute v1.19.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v0.13.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/keyvault/internal v0.7.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v0.9.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/certificate-transparency-go v1.1.2 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-tspi v0.2.1-0.20190423175329-115dea689aad // indirect
	github.com/google/s2a-go v0.1.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.3 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/thales-e-security/pool v0.0.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/oauth2 v0.7.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230403163135-c38d8f061ccd // indirect
	google.golang.org/protobuf v1.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// relying on fork till changes get upstreamed.
replace github.com/google/go-attestation => github.com/smallstep/go-attestation v0.4.4-0.20230224121042-1bcb20a75add

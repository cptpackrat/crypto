//go:build !noazurekms
// +build !noazurekms

package azurekms

import (
	"context"
	"crypto"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azkeys"
	"github.com/pkg/errors"
	"go.step.sm/crypto/kms/apiv1"
	"go.step.sm/crypto/kms/uri"
)

func init() {
	apiv1.Register(apiv1.AzureKMS, func(ctx context.Context, opts apiv1.Options) (apiv1.KeyManager, error) {
		return New(ctx, opts)
	})
}

// Scheme is the scheme used for the Azure Key Vault uris.
const Scheme = "azurekms"

var (
	valueTrue       = true
	value2048 int32 = 2048
	value3072 int32 = 3072
	value4096 int32 = 4096
)

type keyType struct {
	Kty   azkeys.JSONWebKeyType
	Curve azkeys.JSONWebKeyCurveName
}

func (k keyType) KeyType(pl apiv1.ProtectionLevel) azkeys.JSONWebKeyType {
	switch k.Kty {
	case azkeys.JSONWebKeyTypeEC:
		if pl == apiv1.HSM {
			return azkeys.JSONWebKeyTypeECHSM
		}
		return k.Kty
	case azkeys.JSONWebKeyTypeRSA:
		if pl == apiv1.HSM {
			k.Kty = azkeys.JSONWebKeyTypeRSAHSM
		}
		return k.Kty
	case azkeys.JSONWebKeyTypeECHSM, azkeys.JSONWebKeyTypeRSAHSM:
		return k.Kty
	default:
		return ""
	}
}

var signatureAlgorithmMapping = map[apiv1.SignatureAlgorithm]keyType{
	apiv1.UnspecifiedSignAlgorithm: {
		Kty:   azkeys.JSONWebKeyTypeEC,
		Curve: azkeys.JSONWebKeyCurveNameP256,
	},
	apiv1.SHA256WithRSA: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.SHA384WithRSA: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.SHA512WithRSA: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.SHA256WithRSAPSS: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.SHA384WithRSAPSS: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.SHA512WithRSAPSS: {
		Kty: azkeys.JSONWebKeyTypeRSA,
	},
	apiv1.ECDSAWithSHA256: {
		Kty:   azkeys.JSONWebKeyTypeEC,
		Curve: azkeys.JSONWebKeyCurveNameP256,
	},
	apiv1.ECDSAWithSHA384: {
		Kty:   azkeys.JSONWebKeyTypeEC,
		Curve: azkeys.JSONWebKeyCurveNameP384,
	},
	apiv1.ECDSAWithSHA512: {
		Kty:   azkeys.JSONWebKeyTypeEC,
		Curve: azkeys.JSONWebKeyCurveNameP521,
	},
}

// KeyVaultClient is the interface implemented by keyvault.BaseClient. It will
// be used for testing purposes.
type KeyVaultClient interface {
	GetKey(ctx context.Context, name string, version string, options *azkeys.GetKeyOptions) (azkeys.GetKeyResponse, error)
	CreateKey(ctx context.Context, name string, parameters azkeys.CreateKeyParameters, options *azkeys.CreateKeyOptions) (azkeys.CreateKeyResponse, error)
	Sign(ctx context.Context, name string, version string, parameters azkeys.SignParameters, options *azkeys.SignOptions) (azkeys.SignResponse, error)
}

// KeyVault implements a KMS using Azure Key Vault.
//
// To initialize the client we need to define a URI with the following format:
//
//   - azurekms:
//   - azurekms:vault=vault-name
//   - azurekms:environment=env-name
//   - azurekms:vault=vault-name;environment=env-name
//   - azurekms:vault=vault-name?hsm=true
//
// The scheme is "azurekms"; "vault" defines the default key vault to use;
// "environment" defines the Azure Cloud environment to use, options are
// "public" or "AzurePublicCloud", "usgov" or "AzureUSGovernmentCloud", "china"
// or "AzureChinaCloud", "german" or "AzureGermanCloud", it will default to the
// public cloud if not specified; "hsm" defines if a key will be generated by an
// HSM by default.
//
// The URI format for a key in Azure Key Vault is the following:
//
//   - azurekms:name=key-name;vault=vault-name
//   - azurekms:name=key-name;vault=vault-name?version=key-version
//   - azurekms:name=key-name;vault=vault-name?hsm=true
//   - azurekms:name=key-name;vault=vault-name
//
// The "name" is the key name inside the "vault"; "version" is an optional
// parameter that defines the version of they key, if version is not given, the
// latest one will be used; "vault" and "hsm" will override the default value if
// set. The "environment" can only be set to initialize the client.
type KeyVault struct {
	client   *lazyClient
	defaults defaultOptions
}

// defaultDNSSuffix is the suffix of the Azure Public Cloud
const defaultDNSSuffix = "vault.azure.net"

// defaultOptions are custom options that can be passed as defaults using the
// URI in apiv1.Options.
type defaultOptions struct {
	Vault           string
	DNSSuffix       string
	ProtectionLevel apiv1.ProtectionLevel
}

var createCredentials = func(ctx context.Context, opts apiv1.Options) (azcore.TokenCredential, error) {
	var tenantID string
	var clientOptions policy.ClientOptions
	if opts.URI != "" {
		u, err := uri.ParseWithScheme(Scheme, opts.URI)
		if err != nil {
			return nil, err
		}

		// The 'environment' parameter in the URI defines the Cloud environment to
		// be used. By default Azure Public Cloud is used.
		cloudConf, err := getCloudConfiguration(u.Get("environment"))
		if err != nil {
			return nil, err
		}

		// Azure Active Directory endpoint, each environment defines one, but we
		// allow to update it.
		//
		// Defaults to https://login.microsoftonline.com/
		//
		// TODO(mariano): is this option still valid?
		if v := u.Get("aad-endpoint"); v != "" {
			cloudConf.ActiveDirectoryAuthorityHost = v
		}

		clientOptions.Cloud = cloudConf.Configuration

		// ClientSecret credential parameters.
		//
		// TenantID can also be used when using environment variables or managed
		// identities to initialize the credentials.
		clientID := u.Get("client-id")
		clientSecret := u.Get("client-secret")
		tenantID = u.Get("tenant-id")

		// Try to log in only using client credentials in the URI.
		// Client credentials requires:
		//   - client-id
		//   - client-secret
		//   - tenant-id
		if clientID != "" && clientSecret != "" && tenantID != "" {
			return azidentity.NewClientSecretCredential(tenantID, clientID, clientID, &azidentity.ClientSecretCredentialOptions{
				ClientOptions: clientOptions,
			})
		}
	}

	// Attempt to authorize with the following methods:
	// 1. Environment credentials
	//    - Client credentials: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET
	//    - Client certificate: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_CERTIFICATE_PATH, AZURE_CLIENT_CERTIFICATE_PASSWORD (optional)
	//    - Username and password: AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_USERNAME, AZURE_PASSWORD
	// 2. Managed identity credentials (MSI).
	// 3. Azure CLI credential
	return azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{
		ClientOptions: clientOptions,
		TenantID:      tenantID,
	})
}

// New initializes a new KMS implemented using Azure Key Vault.
//
// The URI format used to initialized the Azure Key Vault client is the
// following:
//
//   - azurekms:
//   - azurekms:vault=vault-name
//   - azurekms:vault=vault-name;environment=env-name
//   - azurekms:vault=vault-name?hsm=true
func New(ctx context.Context, opts apiv1.Options) (*KeyVault, error) {
	credential, err := createCredentials(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("error creating azure credentials: %w", err)
	}

	defaults := defaultOptions{
		DNSSuffix: defaultDNSSuffix,
	}
	if opts.URI != "" {
		u, err := uri.ParseWithScheme(Scheme, opts.URI)
		if err != nil {
			return nil, err
		}
		cloudConf, err := getCloudConfiguration(u.Get("environment"))
		if err != nil {
			return nil, err
		}
		defaults = defaultOptions{
			Vault:     u.Get("vault"),
			DNSSuffix: cloudConf.DNSSuffix,
		}
		if u.GetBool("hsm") {
			defaults.ProtectionLevel = apiv1.HSM
		}
	}

	return &KeyVault{
		client:   newLazyClient(defaults.DNSSuffix, lazyClientCreator(credential)),
		defaults: defaults,
	}, nil
}

// GetPublicKey loads a public key from Azure Key Vault by its resource name.
func (k *KeyVault) GetPublicKey(req *apiv1.GetPublicKeyRequest) (crypto.PublicKey, error) {
	if req.Name == "" {
		return nil, errors.New("getPublicKeyRequest 'name' cannot be empty")
	}

	vaultURL, name, version, _, err := parseKeyName(req.Name, k.defaults)
	if err != nil {
		return nil, err
	}

	client, err := k.client.Get(vaultURL)
	if err != nil {
		return nil, err
	}

	ctx, cancel := defaultContext()
	defer cancel()

	resp, err := client.GetKey(ctx, name, version, nil)
	if err != nil {
		return nil, errors.Wrap(err, "keyVault GetKey failed")
	}

	return convertKey(resp.Key)
}

// CreateKey creates a asymmetric key in Azure Key Vault.
func (k *KeyVault) CreateKey(req *apiv1.CreateKeyRequest) (*apiv1.CreateKeyResponse, error) {
	if req.Name == "" {
		return nil, errors.New("createKeyRequest 'name' cannot be empty")
	}

	vault, name, _, hsm, err := parseKeyName(req.Name, k.defaults)
	if err != nil {
		return nil, err
	}

	client, err := k.client.Get(vault)
	if err != nil {
		return nil, err
	}

	// Override protection level to HSM only if it's not specified, and is given
	// in the uri.
	protectionLevel := req.ProtectionLevel
	if protectionLevel == apiv1.UnspecifiedProtectionLevel && hsm {
		protectionLevel = apiv1.HSM
	}

	kt, ok := signatureAlgorithmMapping[req.SignatureAlgorithm]
	if !ok {
		return nil, errors.Errorf("keyVault does not support signature algorithm %q", req.SignatureAlgorithm)
	}

	var keySize *int32
	if kt.Kty == azkeys.JSONWebKeyTypeRSA || kt.Kty == azkeys.JSONWebKeyTypeRSAHSM {
		switch req.Bits {
		case 2048:
			keySize = &value2048
		case 0, 3072:
			keySize = &value3072
		case 4096:
			keySize = &value4096
		default:
			return nil, errors.Errorf("keyVault does not support key size %d", req.Bits)
		}
	}

	keyType := kt.KeyType(protectionLevel)
	created := now()

	ctx, cancel := defaultContext()
	defer cancel()

	resp, err := client.CreateKey(ctx, name, azkeys.CreateKeyParameters{
		Kty:     &keyType,
		KeySize: keySize,
		Curve:   &kt.Curve,
		KeyOps: []*azkeys.JSONWebKeyOperation{
			pointer(azkeys.JSONWebKeyOperationSign),
			pointer(azkeys.JSONWebKeyOperationVerify),
		},
		KeyAttributes: &azkeys.KeyAttributes{
			Enabled:   &valueTrue,
			Created:   &created,
			NotBefore: &created,
		},
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "keyVault CreateKey failed")
	}

	publicKey, err := convertKey(resp.Key)
	if err != nil {
		return nil, err
	}

	keyURI := getKeyName(vault, name, resp.Key)
	return &apiv1.CreateKeyResponse{
		Name:      keyURI,
		PublicKey: publicKey,
		CreateSignerRequest: apiv1.CreateSignerRequest{
			SigningKey: keyURI,
		},
	}, nil
}

// CreateSigner returns a crypto.Signer from a previously created asymmetric key.
func (k *KeyVault) CreateSigner(req *apiv1.CreateSignerRequest) (crypto.Signer, error) {
	if req.SigningKey == "" {
		return nil, errors.New("createSignerRequest 'signingKey' cannot be empty")
	}
	return NewSigner(k.client, req.SigningKey, k.defaults)
}

// Close closes the client connection to the Azure Key Vault. This is a noop.
func (k *KeyVault) Close() error {
	return nil
}

// ValidateName validates that the given string is a valid URI.
func (k *KeyVault) ValidateName(s string) error {
	_, _, _, _, err := parseKeyName(s, k.defaults)
	return err
}

type cloudConfiguration struct {
	cloud.Configuration
	DNSSuffix string
}

// getCloudConfiguration returns the cloud configuration for the different
// clouds.
//
// Note that the German configuration does not appear on the SDK. It might not
// work.
func getCloudConfiguration(cloudName string) (cloudConfiguration, error) {
	switch strings.ToUpper(cloudName) {
	case "", "PUBLIC", "AZURECLOUD", "AZUREPUBLICCLOUD":
		return cloudConfiguration{
			Configuration: cloud.AzurePublic,
			DNSSuffix:     "vault.azure.net",
		}, nil
	case "USGOV", "AZUREUSGOVERNMENT", "AZUREUSGOVERNMENTCLOUD":
		return cloudConfiguration{
			Configuration: cloud.AzureGovernment,
			DNSSuffix:     "vault.usgovcloudapi.net",
		}, nil
	case "CHINA", "AZURECHINACLOUD":
		return cloudConfiguration{
			Configuration: cloud.AzureChina,
			DNSSuffix:     "vault.azure.cn",
		}, nil
	case "GERMAN", "GERMANY", "AZUREGERMANCLOUD":
		return cloudConfiguration{
			Configuration: cloud.Configuration{
				ActiveDirectoryAuthorityHost: "https://login.microsoftonline.de/",
				Services:                     map[cloud.ServiceName]cloud.ServiceConfiguration{},
			},
			DNSSuffix: "vault.microsoftazure.de",
		}, nil
	default:
		return cloudConfiguration{}, fmt.Errorf("unknown key vault cloud environment with name %q", cloudName)
	}
}

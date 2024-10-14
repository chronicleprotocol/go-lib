package secrets

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/defiweb/go-eth/types"
	"github.com/defiweb/go-eth/wallet"
	ecies "github.com/ecies/go/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/zclconf/go-cty/cty"
)

const (
	etherumBlockName = "ethereum"
	keyBlockName     = "key"
	keyBlockLabel    = "default"

	varName          = "secrets"
	secretsBlockName = "secrets"

	skipDecryptEnv = "XXX_SECRETS_SKIP_DECRYPT"
)

// DecryptSecrets decrypts secrets in the given HCL body.
// Example:
//
//	ethereum {
//	  key "default" {
//	    address = "0x123...abc"
//		   keystore_path = "/path/to/keystore"
//		   passphrase_file = "/path/to/passwords"
//	  }
//	}
//
//	secrets {
//	  foo = {
//		  "0x123...abc" = "0x234..bcd"
//		  "0x1a3...abc" = "0x2a4..bcd"
//		  "0x1b3...abc" = "0x2b4..bcd"
//	  }
//	}
//
// decrypted = secrets.foo
//
// In this example decrypted will be equal to the decrypted value of "0x234...bcd"
//
// NOTE: To use in test scripts, there is an env variable XXX_SECRETS_SKIP_DECRYPT
// which allow to skip decryption and just check that secret is set set.
// If XXX_SECRETS_SKIP_DECRYPT is set to true, then actual decryption step will be
// skipped and secret value will be replaced with just "<encrypted>".
//
// NOTE: if there is no secret value for configured ethereum public key, then
// secrets.foo will return a hcl.Diagnostics that value is not set.
func DecryptSecrets(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	content, remain, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: secretsBlockName}},
	})
	if diags.HasErrors() {
		return nil, diags
	}

	// return early if there is no secrets to decrypt
	if len(content.Blocks) == 0 {
		return remain, nil
	}

	skipDecrypt, _ := strconv.ParseBool(os.Getenv(skipDecryptEnv))
	addr, key, diags := findEthereumKey(ctx, body, skipDecrypt)
	if diags.HasErrors() {
		return nil, diags
	}

	secrets := make(map[string]cty.Value)
	for _, block := range content.Blocks {
		attrs, diags := block.Body.JustAttributes()
		if diags.HasErrors() {
			return nil, diags
		}
		if len(attrs) == 0 {
			continue
		}

		bsecrets, diags := decryptVariables(ctx, addr, key, attrs, skipDecrypt)
		if diags.HasErrors() {
			return nil, diags
		}
		for k, v := range bsecrets {
			if prev, ok := secrets[k]; ok {
				return nil, hcl.Diagnostics{{
					Severity:    hcl.DiagError,
					Summary:     "Duplicate secret value",
					Detail:      fmt.Sprintf("previous value of %s was %q new value is %q", k, prev.AsString(), v.AsString()),
					Subject:     block.DefRange.Ptr(),
					EvalContext: ctx,
				}}
			}
			secrets[k] = v
		}
	}

	if ctx.Variables == nil {
		ctx.Variables = make(map[string]cty.Value)
	}
	ctx.Variables[varName] = cty.ObjectVal(secrets)
	return remain, nil
}

func findEthereumKey(
	ctx *hcl.EvalContext,
	body hcl.Body,
	skipDecrypt bool,
) (
	addr types.Address,
	key *wallet.PrivateKey,
	diags hcl.Diagnostics,
) {
	ethereum, diags := findEthereumBlock(ctx, body)
	if diags.HasErrors() {
		return addr, nil, diags
	}

	keys, _, diags := ethereum.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{
			Type:       keyBlockName,
			LabelNames: []string{keyBlockLabel},
		}},
	})
	if diags.HasErrors() {
		return addr, nil, diags
	}

	if len(keys.Blocks) == 0 {
		return addr, nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "There is no ethereum.key[default] blocks in config",
			Detail:      "can not determine which ethereum key to use",
			EvalContext: ctx,
		}}
	}

	return loadEthereumKey(ctx, keys.Blocks[0], skipDecrypt)
}

func findEthereumBlock(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	ethereum, _, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: etherumBlockName}},
	})
	if diags.HasErrors() {
		return nil, diags
	}
	if len(ethereum.Blocks) != 1 {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "Wrong number of etherum blocks",
			Detail:      fmt.Sprintf("There should be exactly one ethereum block in config. have %d", len(ethereum.Blocks)),
			EvalContext: ctx,
		}}
	}
	// NEED to expand dyn blocks in order to define all of the keys
	return dynblock.Expand(ethereum.Blocks[0].Body, ctx), nil
}

func loadEthereumKey(ctx *hcl.EvalContext, block *hcl.Block, skipDecrypt bool) (addr types.Address, key *wallet.PrivateKey, diags hcl.Diagnostics) {
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return addr, nil, diags
	}
	loader := &ethereumBlockLoader{block: block, attrs: attrs}

	addr, diags = loader.loadAddress(ctx, "address")
	if diags.HasErrors() {
		return addr, nil, diags
	}

	if skipDecrypt {
		return addr, nil, nil
	}

	keystore, diags := loader.loadKeystore(ctx, "keystore_path")
	if diags.HasErrors() {
		return addr, nil, diags
	}
	passphrase, diags := loader.loadPassphrase(ctx, "passphrase_file")
	if diags.HasErrors() {
		return addr, nil, diags
	}

	var err error
	if len(keystore) > 2 && (keystore[0] == '{' && keystore[len(keystore)-1] == '}') {
		key, err = wallet.NewKeyFromJSONContent([]byte(keystore), passphrase)
	} else {
		key, err = wallet.NewKeyFromDirectory(keystore, passphrase, addr)
	}
	if err != nil {
		return addr, nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "failed to read ethereum key from keystore",
			Detail:      err.Error(),
			EvalContext: ctx,
			Subject:     block.DefRange.Ptr(),
		}}
	}
	return addr, key, nil
}

type ethereumBlockLoader struct {
	block *hcl.Block
	attrs hcl.Attributes
}

func (l *ethereumBlockLoader) loadAddress(ctx *hcl.EvalContext, attrName string) (addr types.Address, diags hcl.Diagnostics) {
	attr := l.attrs[attrName]
	if attr == nil {
		return addr, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     l.block.DefRange.Ptr(),
			Summary:     "Missing key." + attrName + " attribute",
			EvalContext: ctx,
		}}
	}

	s, err := asString(ctx, attr)
	if err != nil {
		return addr, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     attr.Range.Ptr(),
			Summary:     "Failed to get key." + attrName + " attribute value",
			Detail:      err.Error(),
			EvalContext: ctx,
		}}
	}

	addr, err = types.AddressFromHex(s)
	if err != nil {
		return addr, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     attr.Range.Ptr(),
			Summary:     "Malformed ethereum address in key." + attrName + " attribute value",
			Detail:      err.Error(),
			EvalContext: ctx,
		}}
	}

	return addr, nil
}

func (l *ethereumBlockLoader) loadKeystore(ctx *hcl.EvalContext, attrName string) (string, hcl.Diagnostics) {
	attr := l.attrs[attrName]
	if attr == nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     l.block.DefRange.Ptr(),
			Summary:     "Missing key." + attrName + " attribute",
			EvalContext: ctx,
		}}
	}

	s, err := asString(ctx, attr)
	if err != nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     attr.Range.Ptr(),
			Summary:     "Failed to get key." + attrName + " attribute value",
			Detail:      err.Error(),
			EvalContext: ctx,
		}}
	}
	return s, nil
}

func (l *ethereumBlockLoader) loadPassphrase(ctx *hcl.EvalContext, attrName string) (string, hcl.Diagnostics) {
	attr := l.attrs[attrName]
	if attr == nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     l.block.DefRange.Ptr(),
			Summary:     "Missing key." + attrName + " attribute",
			EvalContext: ctx,
		}}
	}

	path, err := asString(ctx, attr)
	if err != nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     attr.Range.Ptr(),
			Summary:     "Failed to get key." + attrName + " attribute value",
			Detail:      err.Error(),
			EvalContext: ctx,
		}}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Subject:     attr.Range.Ptr(),
			Summary:     "Failed to read pathphrase from key." + attrName + " attribute value",
			Detail:      err.Error(),
			EvalContext: ctx,
		}}
	}
	return strings.TrimSpace(string(b)), nil
}

func asString(ctx *hcl.EvalContext, attr *hcl.Attribute) (string, error) {
	val, err := attr.Expr.Value(ctx)
	if err != nil {
		return "", err
	}
	if val.Type() != cty.String {
		return "", fmt.Errorf("wrong attribute value type: expected string, got %s", val.Type().FriendlyName())
	}
	return val.AsString(), nil
}

func decryptVariables(
	ctx *hcl.EvalContext,
	addr types.Address,
	key *wallet.PrivateKey,
	attrs hcl.Attributes,
	skipDecrypt bool,
) (map[string]cty.Value, hcl.Diagnostics) {
	m := make(map[string]cty.Value)
	ownAddr := cty.StringVal(strings.ToLower(addr.String()))
	var privateKey *ecies.PrivateKey
	if !skipDecrypt {
		keyBytes := crypto.FromECDSA(key.PrivateKey())
		privateKey = ecies.NewPrivateKeyFromBytes(keyBytes)
	}

	for name, attr := range attrs {
		value, diags := attr.Expr.Value(ctx)
		if diags.HasErrors() {
			return nil, diags
		}
		if !value.CanIterateElements() {
			return nil, hcl.Diagnostics{{
				Severity:    hcl.DiagError,
				Summary:     "Secret value is not a map",
				Subject:     attr.Range.Ptr(),
				EvalContext: ctx,
			}}
		}

		ciphertext := cty.NullVal(cty.String)
		value.ForEachElement(func(k, v cty.Value) bool {
			addr := strings.ToLower(k.AsString())
			if addr == ownAddr.AsString() {
				ciphertext = v
				return true
			}
			return false
		})

		if ciphertext.IsNull() {
			return nil, hcl.Diagnostics{{
				Severity:    hcl.DiagError,
				Summary:     "Secret has no value for own public key",
				Subject:     attr.Range.Ptr(),
				EvalContext: ctx,
			}}
		}

		if skipDecrypt {
			m[name] = cty.StringVal("<encrypted>")
			continue
		}

		b, err := hexutil.Decode(ciphertext.AsString())
		if err != nil {
			return nil, hcl.Diagnostics{{
				Severity:    hcl.DiagError,
				Summary:     "Secret is not hex encoded",
				Detail:      err.Error(),
				Subject:     attr.Range.Ptr(),
				EvalContext: ctx,
			}}
		}

		plaintext, err := ecies.Decrypt(privateKey, b)
		if err != nil {
			return nil, hcl.Diagnostics{{
				Severity:    hcl.DiagError,
				Summary:     "Secret can not be decrypted",
				Detail:      err.Error(),
				Subject:     attr.Range.Ptr(),
				EvalContext: ctx,
			}}
		}
		m[name] = cty.StringVal(string(plaintext))
	}
	return m, nil
}

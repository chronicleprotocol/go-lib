package secrets

import (
	"fmt"
	"os"
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

	key, diags := findKey(ctx, body)
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

		bsecrets, diags := decryptVariables(ctx, key, attrs)
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

func findKey(ctx *hcl.EvalContext, body hcl.Body) (*wallet.PrivateKey, hcl.Diagnostics) {
	parent, diags := findEthereumBlock(ctx, body)
	if diags.HasErrors() {
		return nil, diags
	}

	content, _, diags := parent.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{
			Type:       keyBlockName,
			LabelNames: []string{keyBlockLabel},
		}},
	})
	if diags.HasErrors() {
		return nil, diags
	}

	if len(content.Blocks) == 0 {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "There is no ethereum.key[default] blocks in config",
			Detail:      "Can not determine which ethereum key to use",
			EvalContext: ctx,
		}}
	}

	attrs, diags := content.Blocks[0].Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	saddr, diags := readStringAttr(ctx, attrs, "address")
	if diags.HasErrors() {
		return nil, diags
	}
	if saddr == "" {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "missing etehreum.key[default] address attribute",
			EvalContext: ctx,
			Subject:     content.Blocks[0].DefRange.Ptr(),
		}}
	}
	addr, err := types.AddressFromHex(saddr)
	if err != nil {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "marlformed etehreum.key[default].address attribute",
			Detail:      err.Error(),
			EvalContext: ctx,
			Subject:     content.Blocks[0].DefRange.Ptr(),
		}}
	}

	keystoreDir, diags := readStringAttr(ctx, attrs, "keystore_path")
	if diags.HasErrors() {
		return nil, diags
	}
	if keystoreDir == "" {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "missing etehreum.keystore_path attribute",
			EvalContext: ctx,
			Subject:     content.Blocks[0].DefRange.Ptr(),
		}}
	}

	passphraseFile, diags := readStringAttr(ctx, attrs, "passphrase_file")
	if diags.HasErrors() {
		return nil, diags
	}

	key, err := readAccountKey(keystoreDir, passphraseFile, addr)
	if err != nil {
		return nil, hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "failed to read ethereum key from keystore",
			Detail:      err.Error(),
			EvalContext: ctx,
			Subject:     content.Blocks[0].DefRange.Ptr(),
		}}
	}

	return key, nil
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

func readStringAttr(ctx *hcl.EvalContext, attrs hcl.Attributes, key string) (string, hcl.Diagnostics) {
	attr := attrs[key]
	if attr == nil {
		return "", nil
	}
	val, err := attr.Expr.Value(ctx)
	if err != nil {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "Failed to evaluate attribute extression",
			Detail:      err.Error(),
			EvalContext: ctx,
			Subject:     attr.Range.Ptr(),
		}}
	}
	if val.Type() != cty.String {
		return "", hcl.Diagnostics{{
			Severity:    hcl.DiagError,
			Summary:     "Invalid attribute type",
			Detail:      fmt.Sprintf("Expected string, got %s", val.Type().FriendlyName()),
			EvalContext: ctx,
			Subject:     attr.Range.Ptr(),
		}}
	}
	return val.AsString(), nil
}

func readPassphrase(passFile string) (string, error) {
	if passFile == "" {
		return "", nil
	}
	b, err := os.ReadFile(passFile)
	if err != nil {
		return "", fmt.Errorf("failed to read passphrase file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func readAccountKey(keystoreDir string, passFile string, address types.Address) (*wallet.PrivateKey, error) {
	passphrase, err := readPassphrase(passFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase file: %w", err)
	}
	key, err := wallet.NewKeyFromDirectory(keystoreDir, passphrase, address)
	if err == nil {
		return key, nil
	}
	return wallet.NewKeyFromJSONContent([]byte(keystoreDir), passFile)
}

func decryptVariables(ctx *hcl.EvalContext, key *wallet.PrivateKey, attrs hcl.Attributes) (map[string]cty.Value, hcl.Diagnostics) {
	m := make(map[string]cty.Value)
	ownAddr := cty.StringVal(strings.ToLower(key.Address().String()))
	keyBytes := crypto.FromECDSA(key.PrivateKey())
	privateKey := ecies.NewPrivateKeyFromBytes(keyBytes)

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

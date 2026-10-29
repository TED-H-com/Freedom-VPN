package billing

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

var ErrCipher = errors.New("billing payload encryption failed")

// Vault supports key rotation: new writes use Active; old key IDs must remain
// available until retention permits deleting their inbox/session ciphertexts.
type Vault struct {
	active string
	keys   map[string]cipher.AEAD
}

func NewVault(active string, keys map[string][]byte) (*Vault, error) {
	v := &Vault{active: active, keys: map[string]cipher.AEAD{}}
	for id, key := range keys {
		if id == "" || len(id) > 255 || len(key) != 32 {
			return nil, ErrCipher
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, ErrCipher
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, ErrCipher
		}
		v.keys[id] = aead
	}
	if v.keys[active] == nil {
		return nil, ErrCipher
	}
	return v, nil
}
func NewVaultFromFile(id, path string) (*Vault, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrCipher
	}
	defer clear(key)
	return NewVault(id, map[string][]byte{id: key})
}
func (v *Vault) Seal(raw []byte, aad string) ([]byte, error) {
	if v == nil {
		return nil, ErrCipher
	}
	a := v.keys[v.active]
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, ErrCipher
	}
	out := append([]byte{1, byte(len(v.active))}, []byte(v.active)...)
	out = append(out, nonce...)
	return a.Seal(out, nonce, raw, []byte(aad)), nil
}
func (v *Vault) Open(data []byte, aad string) ([]byte, error) {
	if v == nil || len(data) < 2 || data[0] != 1 {
		return nil, ErrCipher
	}
	n := int(data[1])
	if len(data) < 2+n {
		return nil, ErrCipher
	}
	a := v.keys[string(data[2:2+n])]
	if a == nil || len(data) < 2+n+a.NonceSize() {
		return nil, ErrCipher
	}
	nonce := data[2+n : 2+n+a.NonceSize()]
	raw, err := a.Open(nil, nonce, data[2+n+a.NonceSize():], []byte(aad))
	if err != nil {
		return nil, ErrCipher
	}
	return raw, nil
}

// Hash authenticated raw bytes through a canonical JSON envelope. This accepts
// provider JSON outside our integer-only signed-document subset without float
// conversions or losing duplicate-key/raw-body distinctions.
func payloadHash(raw []byte) ([]byte, error) {
	b, err := canonjson.Marshal(map[string]string{"raw_b64": base64.StdEncoding.EncodeToString(raw)})
	if err != nil {
		return nil, err
	}
	sum, err := canonjson.Hash(b)
	return sum[:], err
}
func aad(parts ...string) string {
	var b []byte
	for _, s := range parts {
		b = binary.BigEndian.AppendUint32(b, uint32(len(s)))
		b = append(b, s...)
	}
	return string(b)
}

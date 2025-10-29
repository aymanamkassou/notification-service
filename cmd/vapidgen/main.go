package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func main() {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pad := func(b []byte, n int) []byte { if len(b) >= n { return b }; p := make([]byte, n); copy(p[n-len(b):], b); return p }
	px := pad(priv.PublicKey.X.Bytes(), 32)
	py := pad(priv.PublicKey.Y.Bytes(), 32)
	pub := append([]byte{0x04}, append(px, py...)...)
	pubB64 := base64.RawURLEncoding.EncodeToString(pub)
	privB64 := base64.RawURLEncoding.EncodeToString(pad(priv.D.Bytes(), 32))
	out := map[string]string{"publicKey": pubB64, "privateKey": privB64}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

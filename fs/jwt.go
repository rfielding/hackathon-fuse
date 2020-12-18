package fs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"reflect"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	jwk "github.com/lestrrat/go-jwx/jwk"
)

type UserPolicy struct {
	Label     string `json:"label,omitempty" bson:"label,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty" bson:"exp,omitempty"` // in SECONDS, not NANOSEC!
	//	IssuedAt  int64               `json:"iat,omitempty" bson:"iat,omitempty"`
	//	NotBefore int64               `json:"nbf,omitempty" bson:"nbf,omitempty"` // Set this back in the past to allow for clock skews
	Subject  string              `json:"sub,omitempty" bson:"sub,omitempty"`
	Issuer   string              `json:"iss,omitempty" bson:"iss,omitempty"`
	Audience string              `json:"aud,omitempty" bson:"aud,omitempty"`
	Values   map[string][]string `json:"values,omitempty" bson:"values,omitempty"`
}

func Authenticate(issuer string, token string) (*UserPolicy, error) {
	pemPub, err := ioutil.ReadFile(issuer + ".pub")
	if err != nil {
		log.Fatal("unable to decode public key %.pub", issuer)
		return nil, err
	}
	pemPriv, err := ioutil.ReadFile(issuer + ".priv")
	if err != nil {
		log.Fatal("unable to decode private key %.pub", issuer)
		return nil, err
	}
	_, pub := decode(string(pemPriv), string(pemPub))
	claims, err := AuthenticateVsKey(pub, token)
	if err != nil {
		return nil, err
	}
	s, err := json.MarshalIndent(claims, "", "  ")
	if err != nil {
		return nil, err
	}
	var up UserPolicy
	err = json.Unmarshal(s, &up)
	if err != nil {
		return nil, err
	}
	return &up, nil
}

func AuthenticateVsKey(pubKey *ecdsa.PublicKey, token string) (jwt.Claims, error) {
	js, err := jwt.Parse(token, func(jt *jwt.Token) (interface{}, error) {
		if jt == nil {
			return nil, fmt.Errorf("invalid token")
		}
		if jt.Method != jwt.GetSigningMethod("ES512") {
			return nil, fmt.Errorf("We ONLY support ES512 tokens")
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("error checking token: %v", err)
	}
	return js.Claims, nil
}

func Sign(issuer string, claims string) string {
	pemPub, err := ioutil.ReadFile(issuer + ".pub")
	if err != nil {
		log.Fatal("unable to decode public key %.pub", issuer)
		return ""
	}
	pemPriv, err := ioutil.ReadFile(issuer + ".priv")
	if err != nil {
		log.Fatal("unable to decode private key %.pub", issuer)
		return ""
	}
	priv, _ := decode(string(pemPriv), string(pemPub))
	jwt, err := GetJWT(claims, priv, issuer)
	if err != nil {
		log.Fatal("unable to make jwt: %v", err)
		return ""
	}
	return jwt
}

func GetJWT(s string, privkey *ecdsa.PrivateKey, issuer string) (string, error) {
	var up UserPolicy
	err := json.Unmarshal([]byte(s), &up)
	if err != nil {
		return "", err
	}
	duration := time.Duration(2 * time.Hour)
	now := time.Now()
	up.ExpiresAt = now.Add(duration).Unix()
	up.Issuer = issuer

	s2, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return "", err
	}

	var claims jwt.MapClaims
	err = json.Unmarshal([]byte(s2), &claims)
	if err != nil {
		return "", err
	}
	alg := jwt.GetSigningMethod("ES512")
	if alg == nil {
		return "", fmt.Errorf("could not find ES512 algorithm")
	}
	token := jwt.NewWithClaims(alg, claims)

	tokenStr, err := token.SignedString(privkey)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func decode(pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to parse ec private key: %v", err))
		os.Exit(1)
	}
	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		fmt.Println(fmt.Errorf("unable to parse ec public key: %v", err))
		os.Exit(1)
	}
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	return privateKey, publicKey
}

func JwtKeyExport(pubkeyPem string, jwxDest string) error {
	pubPEM, err := ioutil.ReadFile(pubkeyPem)
	if err != nil {
		return err
	}

	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return fmt.Errorf("failed to parse PEM block containing the public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse DER encoded public key: " + err.Error())
	}

	jwxPub, err := jwk.New(publicKey)
	if err != nil {
		return fmt.Errorf("Unable to export to jwk: %v", err)
	}
	jwxPub.Set(jwk.KeyUsageKey, jwk.ForSignature)
	jwxPub.Set(jwk.KeyIDKey, "1")
	jwxPubBytes, err := json.MarshalIndent(jwxPub, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to marshal jwx representation to json: %v", err)
	}
	err = ioutil.WriteFile(jwxDest, jwxPubBytes, 0744)
	if err != nil {
		return fmt.Errorf("unable to write out public key jwk file to %s: %v", jwxDest, err)
	}
	return nil
}

func JwtKeygen(privFileDest string, pubFileDest string) error {

	pubkeyCurve := elliptic.P521()

	privateKey, err := ecdsa.GenerateKey(pubkeyCurve, rand.Reader)

	if err != nil {
		return fmt.Errorf("generating key pair: %v", privateKey)
	}

	publicKey := privateKey.PublicKey

	x509EncodedPriv, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("Encoding %s private key: %v", reflect.TypeOf(privateKey), err)
	}
	pemEncodedPriv := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: x509EncodedPriv})

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return fmt.Errorf("Encoding %s public key: %v", reflect.TypeOf(publicKey), err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	priv2, pub2 := decode(string(pemEncodedPriv), string(pemEncodedPub))

	// Check that the key actually works before issuing it.

	var h hash.Hash
	h = md5.New()
	r := big.NewInt(0)
	s := big.NewInt(0)

	io.WriteString(h, "This is a message to be signed and verified by ECDSA!")
	signHash := h.Sum(nil)

	r, s, serr := ecdsa.Sign(rand.Reader, priv2, signHash)
	if serr != nil {
		return fmt.Errorf("signature check failed: %v", serr)
	}

	signature := r.Bytes()
	signature = append(signature, s.Bytes()...)

	// Verify
	verifyStatus := ecdsa.Verify(pub2, signHash, r, s)
	if verifyStatus == false {
		return fmt.Errorf("testing signature on new keypair failed to verify: %v", err)
	}

	err = ioutil.WriteFile(pubFileDest, pemEncodedPub, 0744)
	if err != nil {
		return fmt.Errorf("unable to write out public key file to %s: %v", pubFileDest, err)
	}

	err = ioutil.WriteFile(privFileDest, pemEncodedPriv, 0700)
	if err != nil {
		return fmt.Errorf("unable to write out private key file to %s: %v", privFileDest, err)
	}

	jwxPub, err := jwk.New(&publicKey)
	if err != nil {
		return fmt.Errorf("Unable to export to jwk: %v", err)
	}
	jwxPub.Set(jwk.KeyUsageKey, jwk.ForSignature)
	jwxPub.Set(jwk.KeyIDKey, "1")
	jwxPubBytes, err := json.MarshalIndent(jwxPub, "", "  ")
	if err != nil {
		return fmt.Errorf("Unable to marshal jwx representation to json: %v", err)
	}
	jwxDest := pubFileDest + ".jwk"
	err = ioutil.WriteFile(jwxDest, jwxPubBytes, 0744)
	if err != nil {
		return fmt.Errorf("unable to write out public key jwk file to %s: %v", jwxDest, err)
	}

	fmt.Printf("Private key (keep secret!) written to: %s\n", privFileDest)
	fmt.Printf("Public key written to: %s\n", pubFileDest)
	fmt.Printf("Public key jwx export written to: %s\n", jwxDest)

	return nil
}

package issue

import (
	"backend/internal/config"
	"crypto/rsa"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	irma "github.com/privacybydesign/irmago"
)

type JwtCreator interface {
	CreateJwt(email string) (jwt string, err error)
}

func NewIrmaJwtCreator(cfg config.JWTConfig, privateKeyPath string,
	issuerId string,
	crediential string,
	attributes config.EmailCredentialAttributes,
) (*DefaultJwtCreator, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)

	if err != nil {
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)

	if err != nil {
		return nil, err
	}

	return &DefaultJwtCreator{
		issuerId:   issuerId,
		privateKey: privateKey,
		credential: crediential,
		attributes: cfg.Attributes,
	}, nil
}

type DefaultJwtCreator struct {
	privateKey *rsa.PrivateKey
	issuerId   string
	credential string
	attributes config.EmailCredentialAttributes
}

func (jc *DefaultJwtCreator) CreateJwt(email string) (string, error) {
	issuanceRequest := irma.NewIssuanceRequest([]*irma.CredentialRequest{
		{
			CredentialTypeID: irma.NewCredentialTypeIdentifier(jc.credential),
			Attributes: map[string]string{
				jc.attributes.Email:       email,
				jc.attributes.EmailDomain: email[strings.Index(email, "@")+1:],
			},
		},
	})

	return irma.SignSessionRequest(
		issuanceRequest,
		jwt.GetSigningMethod(jwt.SigningMethodRS256.Alg()),
		jc.privateKey,
		jc.issuerId,
	)
}

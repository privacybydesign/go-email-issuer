package issue

import (
	"crypto/rsa"
	"os"

	"github.com/golang-jwt/jwt/v4"
	irma "github.com/privacybydesign/irmago"
)

type JwtCreator interface {
	CreateJwt(email string) (jwt string, err error)
}

func NewIrmaJwtCreator(privateKeyPath string,
	issuerId string,
	crediential string,
	attribute string,
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
		attribute:  attribute,
	}, nil
}

type DefaultJwtCreator struct {
	privateKey *rsa.PrivateKey
	issuerId   string
	credential string
	attribute  string
}

func (jc *DefaultJwtCreator) CreateJwt(email string) (string, error) {
	issuanceRequest := irma.NewIssuanceRequest([]*irma.CredentialRequest{
		{
			CredentialTypeID: irma.NewCredentialTypeIdentifier(jc.credential),
			Attributes: map[string]string{
				jc.attribute: email,
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

package issue

import (
	"backend/internal/config"
	"crypto/rsa"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	irma "github.com/privacybydesign/irmago"
)

type JwtCreator interface {
	CreateJwt(email string) (jwt string, err error)
}

func NewIrmaJwtCreator(cfg config.JWTConfig) (*DefaultJwtCreator, error) {
	keyBytes, err := os.ReadFile(cfg.PrivateKeyPath)

	if err != nil {
		return nil, err
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)

	if err != nil {
		return nil, err
	}

	return &DefaultJwtCreator{
		issuerId:   cfg.IssuerID,
		privateKey: privateKey,
		credential: cfg.Credential,
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
	validity := irma.Timestamp(time.Unix(time.Now().AddDate(1, 0, 0).Unix(), 0))
	issuanceRequest := irma.NewIssuanceRequest([]*irma.CredentialRequest{
		{
			CredentialTypeID: irma.NewCredentialTypeIdentifier(jc.credential),
			Attributes: map[string]string{
				jc.attributes.Email:       email,
				jc.attributes.EmailDomain: email[strings.Index(email, "@")+1:],
			},
			Validity: &validity,
		},
	})

	return irma.SignSessionRequest(
		issuanceRequest,
		jwt.GetSigningMethod(jwt.SigningMethodRS256.Alg()),
		jc.privateKey,
		jc.issuerId,
	)
}

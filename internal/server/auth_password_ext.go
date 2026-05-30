package server

import (
	"context"
	"strings"

	authenticationv1 "admin/api/gen/authentication/v1"
	"admin/internal/data/repo"
)

func loginIdentifier(req *authenticationv1.LoginRequest) string {
	switch req.GetIdentifier().(type) {
	case *authenticationv1.LoginRequest_Email:
		return strings.TrimSpace(req.GetEmail())
	case *authenticationv1.LoginRequest_Mobile:
		return strings.TrimSpace(req.GetMobile())
	default:
		return strings.TrimSpace(req.GetUsername())
	}
}

func verifyAndUpgradePassword(ctx context.Context, finder userCredentialFinder, credential *repo.UserCredentialWithUser, plain string) error {
	if finder == nil || credential == nil || credential.Credential == nil || credential.Credential.Credential == nil {
		return authenticationv1.ErrorUnauthorized("credential not found")
	}
	matched, needsUpgrade, err := repo.VerifyPasswordCredential(plain, *credential.Credential.Credential)
	if err != nil {
		return err
	}
	if !matched {
		return authenticationv1.ErrorIncorrectPassword("incorrect password")
	}
	if needsUpgrade {
		if err := finder.UpgradePasswordCredential(ctx, credential.Credential.ID, plain); err != nil {
			return err
		}
	}
	return nil
}

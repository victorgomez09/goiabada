package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/leodip/goiabada/internal/core"
	core_account "github.com/leodip/goiabada/internal/core/account"
	core_admin "github.com/leodip/goiabada/internal/core/admin"
	core_authorize "github.com/leodip/goiabada/internal/core/authorize"
	core_token "github.com/leodip/goiabada/internal/core/token"
	"github.com/leodip/goiabada/internal/lib"
)

func (s *Server) initRoutes() {

	authorizeValidator := core_authorize.NewAuthorizeValidator(s.database)
	tokenValidator := core_token.NewTokenValidator(s.database)
	profileValidator := core_account.NewProfileValidator(s.database)
	emailValidator := core_account.NewEmailValidator(s.database)
	addressValidator := core_account.NewAddressValidator(s.database)
	phoneValidator := core_account.NewPhoneValidator(s.database)
	passwordValidator := core.NewPasswordValidator()
	identifierValidator := core_admin.NewIdentifierValidator(s.database)
	inputSanitizer := core.NewInputSanitizer()

	codeIssuer := core_authorize.NewCodeIssuer(s.database)
	loginManager := core_authorize.NewLoginManager(codeIssuer)
	otpSecretGenerator := core.NewOTPSecretGenerator()
	tokenIssuer := core_token.NewTokenIssuer()
	emailSender := core.NewEmailSender(s.database)
	smsSender := core.NewSMSSender(s.database)

	s.router.Get("/", s.handleIndexGet())
	s.router.Get("/unauthorized", s.handleUnauthorizedGet())
	s.router.Get("/forgot-password", s.handleForgotPasswordGet())
	s.router.Post("/forgot-password", s.handleForgotPasswordPost(emailSender))
	s.router.Get("/reset-password", s.handleResetPasswordGet())
	s.router.Post("/reset-password", s.handleResetPasswordPost(passwordValidator))

	s.router.Route("/auth", func(r chi.Router) {
		r.Get("/authorize", s.handleAuthorizeGet(authorizeValidator, codeIssuer, loginManager))
		r.Get("/pwd", s.handleAuthPwdGet())
		r.Post("/pwd", s.handleAuthPwdPost(authorizeValidator, loginManager))
		r.Get("/otp", s.handleAuthOtpGet(otpSecretGenerator))
		r.Post("/otp", s.handleAuthOtpPost())
		r.Get("/consent", s.handleConsentGet(codeIssuer))
		r.Post("/consent", s.handleConsentPost(codeIssuer))
		r.Post("/token", s.handleTokenPost(tokenIssuer, tokenValidator))
		r.Post("/callback", s.handleAuthCallbackPost(tokenIssuer, tokenValidator))
		r.Get("/logout", s.handleAccountLogoutGet())
	})
	s.router.Route("/account", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, lib.GetBaseUrl()+"/account/profile", http.StatusFound)
		})
		r.Get("/profile", s.withJwt(s.handleAccountProfileGet()))
		r.Post("/profile", s.withJwt(s.handleAccountProfilePost(profileValidator)))
		r.Get("/email", s.withJwt(s.handleAccountEmailGet()))
		r.Post("/email", s.withJwt(s.handleAccountEmailPost(emailValidator, emailSender)))
		r.Post("/email-send-verification", s.withJwt(s.handleAccountEmailSendVerificationPost(emailSender)))
		r.Get("/email-verify", s.withJwt(s.handleAccountEmailVerifyGet()))
		r.Get("/address", s.withJwt(s.handleAccountAddressGet()))
		r.Post("/address", s.withJwt(s.handleAccountAddressPost(addressValidator)))
		r.Get("/phone", s.withJwt(s.handleAccountPhoneGet()))
		r.Post("/phone", s.withJwt(s.handleAccountPhonePost(phoneValidator)))
		r.Post("/phone-send-verification", s.withJwt(s.handleAccountPhoneSendVerificationPost(smsSender)))
		r.Get("/phone-verify", s.withJwt(s.handleAccountPhoneVerifyGet()))
		r.Post("/phone-verify", s.withJwt(s.handleAccountPhoneVerifyPost()))
		r.Get("/change-password", s.withJwt(s.handleAccountChangePasswordGet()))
		r.Post("/change-password", s.withJwt(s.handleAccountChangePasswordPost(passwordValidator)))
		r.Get("/otp", s.withJwt(s.handleAccountOtpGet(otpSecretGenerator)))
		r.Post("/otp", s.withJwt(s.handleAccountOtpPost()))
		r.Get("/manage-consents", s.withJwt(s.handleAccountManageConsentsGet()))
		r.Post("/manage-consents", s.withJwt(s.handleAccountManageConsentsRevokePost()))
		r.Get("/sessions", s.withJwt(s.handleAccountSessionsGet()))
		r.Post("/sessions", s.withJwt(s.handleAccountSessionsEndSesssionPost()))
		r.Get("/register", s.handleAccountRegisterGet())
		r.Post("/register", s.handleAccountRegisterPost(emailValidator, passwordValidator, emailSender))
		r.Get("/activate", s.handleAccountActivateGet(emailSender))
	})

	s.router.Route("/admin", func(r chi.Router) {
		r.Get("/clients", s.withJwt(s.handleAdminClientsGet()))
		r.Get("/clients/{clientID}/settings", s.withJwt(s.handleAdminClientManageSettingsGet()))
		r.Post("/clients/{clientID}/settings", s.withJwt(s.handleAdminClientManageSettingsPost(identifierValidator)))
		r.Get("/clients/{clientID}/authentication", s.withJwt(s.handleAdminClientManageAuthenticationGet()))
		r.Post("/clients/{clientID}/authentication", s.withJwt(s.handleAdminClientManageAuthenticationPost()))
		r.Get("/clients/{clientID}/oauth2-flows", s.withJwt(s.handleAdminClientManageOAuth2Get()))
		r.Post("/clients/{clientID}/oauth2-flows", s.withJwt(s.handleAdminClientManageOAuth2Post()))
		r.Get("/clients/{clientID}/redirect-uris", s.withJwt(s.handleAdminClientManageRedirectURIsGet()))
		r.Post("/clients/{clientID}/redirect-uris", s.withJwt(s.handleAdminClientManageRedirectURIsPost()))
		r.Get("/clients/{clientID}/permissions", s.withJwt(s.handleAdminClientManagePermissionsGet()))
		r.Post("/clients/{clientID}/permissions", s.withJwt(s.handleAdminClientManagePermissionsPost()))
		r.Get("/clients/generate-new-secret", s.withJwt(s.handleGenerateNewSecretGet()))
		r.Get("/clients/get-permissions", s.withJwt(s.handlePermissionsGet()))
		r.Get("/clients/{clientID}/delete", s.withJwt(s.handleAdminClientsDeleteGet()))
		r.Post("/clients/{clientID}/delete", s.withJwt(s.handleAdminClientsDeletePost()))
		r.Get("/clients/new", s.withJwt(s.handleAdminClientsAddNewGet()))
		r.Post("/clients/new", s.withJwt(s.handleAdminClientsAddNewPost(identifierValidator)))

		r.Get("/resources", s.withJwt(s.handleAdminResourcesGet()))
		r.Get("/resources/{resourceID}/settings", s.withJwt(s.handleAdminResourceManageSettingsGet()))
		r.Post("/resources/{resourceID}/settings", s.withJwt(s.handleAdminResourceManageSettingsPost(identifierValidator)))
		r.Get("/resources/{resourceID}/permissions", s.withJwt(s.handleAdminResourceManagePermissionsGet()))
		r.Post("/resources/{resourceID}/permissions", s.withJwt(s.handleAdminResourceManagePermissionsPost(identifierValidator, inputSanitizer)))
		r.Post("/resources/validate-permission", s.withJwt(s.handleValidatePermissionPost(identifierValidator, inputSanitizer)))
		r.Get("/resources/{resourceID}/delete", s.withJwt(s.handleAdminResourcesDeleteGet()))
		r.Post("/resources/{resourceID}/delete", s.withJwt(s.handleAdminResourcesDeletePost()))
		r.Get("/resources/new", s.withJwt(s.handleAdminResourcesAddNewGet()))
		r.Post("/resources/new", s.withJwt(s.handleAdminResourcesAddNewPost(identifierValidator)))

		r.Get("/roles", s.withJwt(s.handleAdminRolesGet()))
		r.Get("/roles/{roleID}/settings", s.withJwt(s.handleAdminRoleManageSettingsGet()))
		r.Post("/roles/{roleID}/settings", s.withJwt(s.handleAdminRoleManageSettingsPost(identifierValidator)))
		r.Get("/roles/{roleID}/users-in-role", s.withJwt(s.handleAdminRoleManageUsersInRoleGet()))
		r.Get("/roles/{roleID}/users-in-role/add", s.withJwt(s.handleAdminRoleManageUsersInRoleAddGet()))
		r.Post("/roles/{roleID}/users-in-role/add", s.withJwt(s.handleAdminRoleManageUsersInRoleAddPost()))
		r.Get("/roles/{roleID}/users-in-role/search", s.withJwt(s.handleAdminRoleManageUsersInRoleSearchGet()))
	})
}

func (s *Server) withJwt(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return MiddlewareJwt(handlerFunc, s.database, s.sessionStore, s.tokenValidator)
}

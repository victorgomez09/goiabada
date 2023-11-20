package integrationtests

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/leodip/goiabada/internal/data"
	"github.com/leodip/goiabada/internal/entities"
	"github.com/leodip/goiabada/internal/enums"
	"github.com/leodip/goiabada/internal/initialization"
	"github.com/leodip/goiabada/internal/lib"
	"github.com/pquerna/otp/totp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/clause"
)

var database *data.Database

func setup() {
	if database == nil {
		initialization.Viper()
		db, err := data.NewDatabase()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		database = db
		seedTestData(database)
	}
}

type createHttpClientInput struct {
	T *testing.T
}

func createHttpClient(input *createHttpClientInput) *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		input.T.Fatal(err)
	}
	client := &http.Client{
		Jar: jar,
	}

	// disable follow redirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = tr
	return client
}

//lint:ignore U1000 This function is used to debug tests
func dumpResponseBody(t *testing.T, response *http.Response) {
	t.Log("Response body:")
	byteArr, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	response.Body = io.NopCloser(bytes.NewReader(byteArr))
	content := string(byteArr)
	t.Log(content)
}

func getCsrfValue(t *testing.T, response *http.Response) string {
	byteArr, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	response.Body = io.NopCloser(bytes.NewReader(byteArr))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(byteArr)))
	if err != nil {
		t.Fatal(err)
	}
	csrfNode := doc.Find("input[name='gorilla.csrf.Token']")
	if csrfNode.Length() != 1 {
		t.Fatal("expecting to find 'gorilla.csrf.Token' but it was not found")
	}
	csrf, exists := csrfNode.Attr("value")
	if !exists {
		t.Fatal("input 'gorilla.csrf.Token' does not have a value")
	}
	return csrf
}

func getOtpSecret(t *testing.T, response *http.Response) string {
	byteArr, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	response.Body = io.NopCloser(bytes.NewReader(byteArr))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(byteArr)))
	if err != nil {
		t.Fatal(err)
	}
	secret := doc.Find("pre.text-center")
	if secret.Length() != 1 {
		t.Fatal("expecting to find pre element with class 'text-center' but it was not found")
	}
	return secret.Text()
}

func authenticateWithPassword(t *testing.T, client *http.Client, email string, password string, csrf string) *http.Response {
	destUrl := lib.GetBaseUrl() + "/auth/pwd"
	formData := url.Values{
		"email":              {email},
		"password":           {password},
		"gorilla.csrf.Token": {csrf},
	}

	formDataString := formData.Encode()
	requestBody := strings.NewReader(formDataString)
	request, err := http.NewRequest("POST", destUrl, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func deleteAllUserConsents(t *testing.T) {
	err := database.DB.Exec("DELETE FROM user_consents").Error
	if err != nil {
		t.Fatal(err)
	}
}

func postConsent(t *testing.T, client *http.Client, consents []int, csrf string) (resp *http.Response) {
	destUrl := lib.GetBaseUrl() + "/auth/consent"

	formData := url.Values{
		"gorilla.csrf.Token": {csrf},
	}
	for _, consent := range consents {
		formData.Add(fmt.Sprintf("consent%d", consent), "[on]")
	}
	if len(consents) > 0 {
		formData.Add("btnSubmit", "submit")
	} else {
		formData.Add("btnCancel", "cancel")
	}

	formDataString := formData.Encode()
	requestBody := strings.NewReader(formDataString)
	request, err := http.NewRequest("POST", destUrl, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func authenticateWithOtp(t *testing.T, client *http.Client, otp string, csrf string) *http.Response {
	destUrl := lib.GetBaseUrl() + "/auth/otp"
	formData := url.Values{
		"otp":                {otp},
		"gorilla.csrf.Token": {csrf},
	}

	formDataString := formData.Encode()
	requestBody := strings.NewReader(formDataString)
	request, err := http.NewRequest("POST", destUrl, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func grantConsent(t *testing.T, clientIdentifier string, email string, scope string) {
	client, err := database.GetClientByClientIdentifier(clientIdentifier)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal(fmt.Errorf("can't grant consent because client %v does not exist", clientIdentifier))
	}

	user, err := database.GetUserByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	if user == nil {
		t.Fatal(fmt.Errorf("can't grant consent because user %v does not exist", email))
	}

	consent := entities.UserConsent{
		ClientId:  client.Id,
		UserId:    user.Id,
		Scope:     scope,
		GrantedAt: time.Now().UTC(),
	}
	err = database.DB.Create(&consent).Error
	if err != nil {
		t.Fatal(err)
	}
}

func postToTokenEndpoint(t *testing.T, client *http.Client, url string, formData url.Values) map[string]interface{} {

	formDataString := formData.Encode()
	requestBody := strings.NewReader(formDataString)
	request, err := http.NewRequest("POST", url, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		t.Fatal(err)
	}

	return data.(map[string]interface{})
}

func createAuthCode(t *testing.T, scope string) *entities.Code {
	setup()

	deleteAllUserConsents(t)

	codeChallenge := "0BnoD4e6xPCPip8rqZ9Zc2RqWOFfvryu9vzXJN4egoY"

	destUrl := viper.GetString("BaseUrl") +
		"/auth/authorize/?client_id=test-client-1&redirect_uri=https://goiabada-test-client:8090/callback.html&response_type=code" +
		"&code_challenge_method=S256&code_challenge=" + codeChallenge +
		"&response_mode=query&scope=" + url.QueryEscape(scope) + "&state=a1b2c3&nonce=m9n8b7" +
		"&acr_values=" + enums.AcrLevel1.String()

	client := createHttpClient(&createHttpClientInput{
		T: t,
	})

	resp, err := client.Get(destUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/pwd")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/pwd")
	defer resp.Body.Close()

	// pwd page
	csrf := getCsrfValue(t, resp)

	resp = authenticateWithPassword(t, client, "mauro@outlook.com", "abc123", csrf)
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/consent")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/consent")
	defer resp.Body.Close()

	// consent page
	csrf = getCsrfValue(t, resp)

	// grant consent to all possible scopes
	resp = postConsent(t, client, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, csrf)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode)

	assertRedirect(t, resp, "/callback.html")
	codeVal, stateVal := getCodeAndStateFromUrl(t, resp)
	assert.Equal(t, "a1b2c3", stateVal)

	codeHash, err := lib.HashString(codeVal)
	if err != nil {
		t.Fatal(err)
	}
	code, err := database.GetCodeByCodeHash(codeHash, false)
	if err != nil {
		t.Fatal(err)
	}
	code.Code = codeVal

	// unescape scope
	scope, err = url.QueryUnescape(scope)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, scope, code.Scope)
	assert.Equal(t, "a1b2c3", code.State)
	assert.Equal(t, "m9n8b7", code.Nonce)
	assert.Equal(t, enums.AcrLevel1.String(), code.AcrLevel)
	assert.Equal(t, "pwd", code.AuthMethods)
	assert.Equal(t, false, code.Used)
	assert.Equal(t, "test-client-1", code.Client.ClientIdentifier)
	assert.Equal(t, "https://goiabada-test-client:8090/callback.html", code.RedirectURI)
	assert.Equal(t, "mauro@outlook.com", code.User.Email)
	return code
}

func getClientSecret(t *testing.T, clientIdentifier string) string {
	client, err := database.GetClientByClientIdentifier(clientIdentifier)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := database.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	secret, err := lib.DecryptText(client.ClientSecretEncrypted, settings.AESEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

func getAnyUserWithOtpDisabled(t *testing.T) *entities.User {
	var user entities.User

	result := database.DB.
		Preload(clause.Associations).
		Order("id DESC").
		Where("enabled = ? and otp_enabled = ?", true, false).First(&user)

	if result.Error != nil {
		t.Fatal(result.Error)
	}
	return &user
}

func assertTimeWithinRange(t *testing.T, expected time.Time, actual time.Time, delta int) {
	assert.True(t, actual.After(expected.Add(time.Duration(-delta)*time.Second)))
	assert.True(t, actual.Before(expected.Add(time.Duration(delta)*time.Second)))
}

func loginUserWithAcrLevel1(t *testing.T, email string, password string) *http.Client {
	codeChallenge := "bQCdz4Hkhb3ctpajAwCCN899mNNfQGmRvMwruYT1Y9Y"
	destUrl := lib.GetBaseUrl() +
		"/auth/authorize/?client_id=test-client-2&redirect_uri=https://goiabada-test-client:8090/callback.html&response_type=code" +
		"&code_challenge_method=S256&code_challenge=" + codeChallenge +
		"&response_mode=query&scope=openid%20profile%20email&state=a1b2c3&nonce=m9n8b7" +
		"&acr_values=" + enums.AcrLevel1.String()

	client := createHttpClient(&createHttpClientInput{
		T: t,
	})

	resp, err := client.Get(destUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/pwd")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/pwd")
	defer resp.Body.Close()

	csrf := getCsrfValue(t, resp)

	resp = authenticateWithPassword(t, client, email, password, csrf)
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/consent")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/consent")
	defer resp.Body.Close()

	assertRedirect(t, resp, "/callback.html")
	codeVal, stateVal := getCodeAndStateFromUrl(t, resp)

	assert.Equal(t, "a1b2c3", stateVal)

	codeHash, err := lib.HashString(codeVal)
	if err != nil {
		t.Fatal(err)
	}
	code, err := database.GetCodeByCodeHash(codeHash, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "openid profile email", code.Scope)
	assert.Equal(t, enums.AcrLevel1.String(), code.AcrLevel)
	assert.Equal(t, enums.AuthMethodPassword.String(), code.AuthMethods)
	assert.Equal(t, false, code.Used)
	assert.Equal(t, "test-client-2", code.Client.ClientIdentifier)
	assert.Equal(t, "https://goiabada-test-client:8090/callback.html", code.RedirectURI)
	assert.Equal(t, email, code.User.Email)

	return client
}

func loginUserWithAcrLevel2(t *testing.T, email string, password string) *http.Client {
	codeChallenge := "bQCdz4Hkhb3ctpajAwCCN899mNNfQGmRvMwruYT1Y9Y"
	destUrl := lib.GetBaseUrl() +
		"/auth/authorize/?client_id=test-client-2&redirect_uri=https://goiabada-test-client:8090/callback.html&response_type=code" +
		"&code_challenge_method=S256&code_challenge=" + codeChallenge +
		"&response_mode=query&scope=openid%20profile%20email&state=a1b2c3&nonce=m9n8b7" +
		"&acr_values=" + enums.AcrLevel2.String()

	client := createHttpClient(&createHttpClientInput{
		T: t,
	})

	resp, err := client.Get(destUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/pwd")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/pwd")
	defer resp.Body.Close()

	csrf := getCsrfValue(t, resp)

	resp = authenticateWithPassword(t, client, email, password, csrf)
	defer resp.Body.Close()

	user, err := database.GetUserByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	if user.OTPEnabled {
		assertRedirect(t, resp, "/auth/otp")
		resp = getPage(t, client, lib.GetBaseUrl()+"/auth/otp")
		defer resp.Body.Close()

		// otp page
		csrf = getCsrfValue(t, resp)

		otp, err := totp.GenerateCode("ILMGDC577J4A4HTR5POU4BU5H5W7VYM2", time.Now())
		if err != nil {
			t.Fatal(err)
		}

		resp = authenticateWithOtp(t, client, otp, csrf)
		defer resp.Body.Close()
	}

	assertRedirect(t, resp, "/auth/consent")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/consent")
	defer resp.Body.Close()

	assertRedirect(t, resp, "/callback.html")
	codeVal, stateVal := getCodeAndStateFromUrl(t, resp)

	assert.Equal(t, "a1b2c3", stateVal)

	codeHash, err := lib.HashString(codeVal)
	if err != nil {
		t.Fatal(err)
	}
	code, err := database.GetCodeByCodeHash(codeHash, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "openid profile email", code.Scope)
	assert.Equal(t, enums.AcrLevel2.String(), code.AcrLevel)
	if user.OTPEnabled {
		assert.Equal(t, enums.AuthMethodPassword.String()+" "+enums.AuthMethodOTP.String(), code.AuthMethods)
	} else {
		assert.Equal(t, enums.AuthMethodPassword.String(), code.AuthMethods)
	}
	assert.Equal(t, false, code.Used)
	assert.Equal(t, "test-client-2", code.Client.ClientIdentifier)
	assert.Equal(t, "https://goiabada-test-client:8090/callback.html", code.RedirectURI)
	assert.Equal(t, email, code.User.Email)

	return client
}

func loginUserWithAcrLevel3(t *testing.T, email string, password string) *http.Client {
	codeChallenge := "bQCdz4Hkhb3ctpajAwCCN899mNNfQGmRvMwruYT1Y9Y"
	destUrl := lib.GetBaseUrl() +
		"/auth/authorize/?client_id=test-client-2&redirect_uri=https://goiabada-test-client:8090/callback.html&response_type=code" +
		"&code_challenge_method=S256&code_challenge=" + codeChallenge +
		"&response_mode=query&scope=openid%20profile%20email&state=a1b2c3&nonce=m9n8b7" +
		"&acr_values=" + enums.AcrLevel3.String()

	client := createHttpClient(&createHttpClientInput{
		T: t,
	})

	resp, err := client.Get(destUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assertRedirect(t, resp, "/auth/pwd")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/pwd")
	defer resp.Body.Close()

	csrf := getCsrfValue(t, resp)

	resp = authenticateWithPassword(t, client, email, password, csrf)
	defer resp.Body.Close()

	user, err := database.GetUserByEmail(email)
	if err != nil {
		t.Fatal(err)
	}

	enrolledInOtp := false

	if user.OTPEnabled {
		assertRedirect(t, resp, "/auth/otp")
		resp = getPage(t, client, lib.GetBaseUrl()+"/auth/otp")
		defer resp.Body.Close()

		// otp page
		csrf = getCsrfValue(t, resp)

		otp, err := totp.GenerateCode("ILMGDC577J4A4HTR5POU4BU5H5W7VYM2", time.Now())
		if err != nil {
			t.Fatal(err)
		}

		resp = authenticateWithOtp(t, client, otp, csrf)
		defer resp.Body.Close()
	} else {
		enrolledInOtp = true
		assertRedirect(t, resp, "/auth/otp")
		resp = getPage(t, client, lib.GetBaseUrl()+"/auth/otp")
		defer resp.Body.Close()

		csrf = getCsrfValue(t, resp)
		otpSecret := getOtpSecret(t, resp)

		otp, err := totp.GenerateCode(otpSecret, time.Now())
		if err != nil {
			t.Fatal(err)
		}

		resp = authenticateWithOtp(t, client, otp, csrf)
		defer resp.Body.Close()
	}

	assertRedirect(t, resp, "/auth/consent")
	resp = getPage(t, client, lib.GetBaseUrl()+"/auth/consent")
	defer resp.Body.Close()

	assertRedirect(t, resp, "/callback.html")
	codeVal, stateVal := getCodeAndStateFromUrl(t, resp)

	assert.Equal(t, "a1b2c3", stateVal)

	codeHash, err := lib.HashString(codeVal)
	if err != nil {
		t.Fatal(err)
	}
	code, err := database.GetCodeByCodeHash(codeHash, false)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "openid profile email", code.Scope)
	assert.Equal(t, enums.AcrLevel3.String(), code.AcrLevel)
	assert.Equal(t, enums.AuthMethodPassword.String()+" "+enums.AuthMethodOTP.String(), code.AuthMethods)
	assert.Equal(t, false, code.Used)
	assert.Equal(t, "test-client-2", code.Client.ClientIdentifier)
	assert.Equal(t, "https://goiabada-test-client:8090/callback.html", code.RedirectURI)
	assert.Equal(t, email, code.User.Email)

	// revert changes to user
	if enrolledInOtp {
		user.OTPEnabled = false
		user.OTPSecret = ""
		_, err = database.SaveUser(user)
		if err != nil {
			t.Fatal(err)
		}
	}

	return client
}

func assertRedirect(t *testing.T, resp *http.Response, location string) {
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	redirectLocation, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, location, redirectLocation.Path)
}

func getPage(t *testing.T, client *http.Client, url string) *http.Response {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func getCodeAndStateFromUrl(t *testing.T, resp *http.Response) (code string, state string) {
	redirectLocation, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}

	code = redirectLocation.Query().Get("code")
	state = redirectLocation.Query().Get("state")

	assert.NotEmpty(t, code, "code should not be empty")
	assert.NotEmpty(t, state, "state should not be empty")

	assert.Equal(t, 128, len(code))

	return code, state
}

func createNewKeyPair(t *testing.T) *entities.KeyPair {
	privateKey, err := lib.GeneratePrivateKey(4096)
	if err != nil {
		t.Fatal("unable to generate a private key")
	}
	privateKeyPEM := lib.EncodePrivateKeyToPEM(privateKey)

	publicKeyASN1_DER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal("unable to marshal public key to PKIX")
	}

	publicKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: publicKeyASN1_DER,
		},
	)

	kid := uuid.New().String()
	publicKeyJWK, err := lib.MarshalRSAPublicKeyToJWK(&privateKey.PublicKey, kid)
	if err != nil {
		t.Fatal(err)
	}

	keyPair := &entities.KeyPair{
		State:             enums.KeyStateCurrent.String(),
		KeyIdentifier:     kid,
		Type:              "RSA",
		Algorithm:         "RS256",
		PrivateKeyPEM:     privateKeyPEM,
		PublicKeyPEM:      publicKeyPEM,
		PublicKeyASN1_DER: publicKeyASN1_DER,
		PublicKeyJWK:      publicKeyJWK,
	}
	return keyPair
}

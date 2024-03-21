package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type MockTime struct {
	CurTime time.Time
}

func (m MockTime) UTCNow() time.Time {
	return m.CurTime
}

var GlobalMockTime *MockTime

func TestMain(m *testing.M) {
	OIDCTestingMode = true
	os.Setenv("DB_FILE", ":memory:")
	os.Setenv("TESLA_PRIVATE_KEY", ":none:")
	os.Setenv("CRYPT_KEY", "12345678901234567890123456789012")
	GetConfig().ReadConfig()
	GlobalMockTime = &MockTime{
		CurTime: time.Now().UTC(),
	}
	GetDB().Time = GlobalMockTime
	GetDB().Connect()
	ResetTestDB()
	InitHTTPRouter()
	code := m.Run()
	os.Exit(code)
}

func ResetTestDB() {
	GetDB().ResetDBStructure()
	GetDB().InitDBStructure()
	TeslaAPIInstance = &TeslaAPIMock{}
	GlobalMockTime.CurTime = time.Now().UTC()
}

func GetNextMondayMidnight() time.Time {
	now := time.Now().UTC()
	curWeekday := now.Weekday()
	if curWeekday == time.Sunday {
		curWeekday = 7
	}
	now = now.AddDate(0, 0, 8-int(curWeekday))
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return now
}

func getTestJWT(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().UTC().AddDate(0, 0, 1).Unix(),
		"iat": time.Now().UTC().Unix(),
		"sub": userID,
		"iss": "",
	})
	tokenString, err := token.SignedString([]byte(OIDCTestingSecret))
	if err != nil {
		log.Fatalln(err)
		return ""
	}
	return tokenString
}

func newHTTPRequest(method, url, bearer string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return req
}

func executeTestRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	httpRouter.ServeHTTP(rr, req)
	return rr
}

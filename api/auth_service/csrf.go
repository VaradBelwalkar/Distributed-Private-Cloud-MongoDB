package auth_service

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"time"
	"text/template"
)
//Here the csrf token is hidden in the form which will be submitted by user in its session
const csrfTemplate = `
<html>
<body>
<form action="/login" method="POST">
  <label for="name">User name</label>
  <input type="text" id="name" name="name">
  <label for="password">Password</label>
  <input type="password" id="password" name="password">
  <input type="hidden" name="csrf" value="{{.csrf}}">
  <input type="submit" value="Submit">
</form>
</body>
</html>
`

// Generate a random CSRF token
func generateCSRFToken() (string, error) {
	// Generate a random token
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}


// Render the form
func RenderForm(w http.ResponseWriter, r *http.Request) {
	// Generate a CSRF token
	csrfToken, err := generateCSRFToken()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}	

	 expiration := time.Now().Add(365 * 24 * time.Hour)
	 cookie    :=    http.Cookie{Name: "csrftoken",Value:csrfToken,Expires:expiration}


	http.SetCookie(w, &cookie)

	// Render the form
	tmpl, err := template.New("form").Parse(csrfTemplate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//tmpl object expects object  of type map[string]string to update values in the provided form tempate
	data := map[string]string{
		"csrf": csrfToken,
	}
	w.WriteHeader(http.StatusOK)
	//Here The Execute command is first updating the form with data object  i.e updating the fields the tmpl can 
	//understand i.e {{.csrf}} field and then updating their values and sending the template to frontend
	if err := tmpl.Execute(w, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}


// This handles the CSRF submission
func HandleSubmit(w http.ResponseWriter, r *http.Request) bool {
	// Check the request method
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)   // 405
		return false
	}

	// Get the CSRF token from the request
	csrfToken := r.FormValue("csrf")

	// Get the CSRF token from the user's session
	sessionCSRFToken, err := r.Cookie("csrftoken")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)   // 400
		return false
	}

	// Compare the CSRF tokens
	if csrfToken != sessionCSRFToken.Value {
		w.WriteHeader(http.StatusPreconditionFailed)   // 412 precondition i.e csrf is failed
		return false
	}
	//Returns boolean value whether user is credible or not
	return true

}

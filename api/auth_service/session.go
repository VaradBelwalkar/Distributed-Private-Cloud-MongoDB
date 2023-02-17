package auth_service

import (
	"net/http"
	"math/rand"
	"time"
	"errors"
)



// ******************** Here understand how sessions work in Golang ********************


//Sessions are important
//You mght think if we are using JWT to authenticate and trust the claims embedded in that, why would we ever
//require session objects to store on the server ?


//Well consider a scenario :

// John has taken a login page by making a GET reqeuest to the endpoint /login,
// He then submits the credentials secured by csrf token by making POST request to /login

// When the John is authenticated at backend, a JWT is created by taking the claims which contains info about John,
// Lets say, username of John simply, so when the response comes, John's client can retrieve this token and can use
// this token to make authorized requests to the backend 
// This is what John is doing by Login  

//Suppose when John is completed his work and want to Logout,
// What would he do?
// His work is to simply click or do specific task at frontend which will send a logout request with specified JWT

// But what backend will do at this point?
// Backend cannot ban the JWT, as it is itself invalid after specified time,
// Also, it is impractical to store JWTs of users who are logged in, which is totally illogical in the sense how 
// JWTs work

// HERE

// A Session comes into the picture
// What you can simply do is create a structure at the backend having simple "username" and a boolean field suppose
// "isActive" which will be "True" is user is logged in, and when user Logs out, this structure will get destroyed

// A map can be used to store the sessions
				//var sessions = map[string]bool
// Here we have used map as it works efficiently beacause of hashing, hence dropping case of array or slice



var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

var Sessions map[string]string

func generateSessionID(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
	_, ok := Sessions[string(b)]
	for ok==true{
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		_, ok = Sessions[string(b)]
	}
    return string(b)
}


// Save a session
func saveSession(sessionID string,username string) {
	Sessions[sessionID] = username
}


// A handler function that creates a new session and saves it
func CreateSession(w http.ResponseWriter,username string) {
	// Create a new session
	rand.Seed(time.Now().UnixNano())
	sessionID := generateSessionID(10)
	// Save the session
	saveSession(sessionID,username)
	// Set the session ID as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:  "session",
		Value: sessionID,
	})
	w.WriteHeader(http.StatusOK)
}

// A handler function that retrieves a session by ID
func RetrieveSession(r *http.Request) (string,string,error){
	// Get the session ID from the request
	sessionID, err := r.Cookie("session")
	if err != nil {
		return "","",err // Here the error means cookie doesn't exist
	}
	// Get the session by ID
	username, ok := Sessions[sessionID.Value]
	if ok == false {
			//If session doesn't exist do something here
			return "","",errors.New("errorHolder")
	} else{
		//Session exists, proceed with JWT authorization
		return sessionID.Value,username,nil
	}

}

//To be called in Logout handler
func DeleteSession(sessionID string){
	delete(Sessions,sessionID)
}


//Explaination ->
//Sessions are nothing but some data structures stored in the memory by the server to retrieve session info
//Like in above scenario, the session is nothing but,
//Whenever a user visits the website first, a session is created i.e a session ID is generated with empty session values 
//Then this session is stored in a Map interface to retrieve it the next time a request with the SAME session.ID comes in
//The above code explains everything

//Here we are passing the session ID to the frontend as a cookie, so frontend needs to save that cookie and 
//need to use that sessionID cookie for further requests so that server can retrieve session info 
//This way csrf is prevented like follow

//First time user visits the server, server creates a unique sessionID and a CSRF token, and saves that 
//session in a session Map and embeds the CSRF in hidden form field. Also it passes the sessionID as cookie to frontend

//The frontend uses this sessionID cookie to make the post request after filling the form, 
//When server receives the POST request, it first tries to get the session cookie, if it doesn't found, it rejects the form 
//Else when it finds a sessionID cookie, it then searches for the sessionID in session Map, and if found,
//Checks the CSRF token value from the stored session to the CSRF received from the form,
//If match is found, it processes the form, else rejects it

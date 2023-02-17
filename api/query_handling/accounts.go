package query_handling

import (

	"context"
	"net/http"
	db "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/database_handling"
	as "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/auth_service"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/docker/docker/api/types/volume"
)
// Here the user will be authenticated first and then request will be fulfilled


func RegisterUser(w http.ResponseWriter, r *http.Request) {

	//CSRF handling
	check:=as.HandleSubmit(w,r)
	if check!=true{
		return
	}
	
	// Parse the POST request body and retrieve the form values
	err := r.ParseForm()
	if err != nil {		
		w.WriteHeader(http.StatusBadRequest)  // Here status code is 400 something went wrong at your side!
		return
	}


	// Validate the form values
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest) // Here status code is 400 something went wrong at your side!
		return
	}
	// Check if a document with the given username already exists
	var result bson.M
	err = db.CollectionHandler.FindOne(context.TODO(), bson.M{"username": username}).Decode(&result)
	if err == nil {
		w.WriteHeader( http.StatusConflict)  //409 statuscode
		return
	} else{			// Here if error is not nil, means document is not found, so free to create new document for the user
		
		volumeOpts:=volume.CreateOptions{
			Name:username,
			Driver:"local",
		}
		
		_,err:=Cli.VolumeCreate(context.TODO(),volumeOpts)
		if err!=nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	// Insert the new user into the database
	_, err = db.CollectionHandler.InsertOne(context.TODO(), bson.M{"username": username, "password": password})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusOK)  
}
}

// Parse and authenticate the user and then remove the account from the database
func RemoveAccount(w http.ResponseWriter, r *http.Request) {

	//CSRF handling
	check:=as.HandleSubmit(w,r)
	if check!=true{
		return
	}
	
	// Parse the POST request body and retrieve the form values
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}


	// Validate the form values
	username := r.Form.Get("username")
	password := r.Form.Get("password")
	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}
	// Check if a document with the given username already exists
	type resultStruct struct{
		Username string
		Password string
	}

	result:=resultStruct{}
	err = db.CollectionHandler.FindOne(context.TODO(), bson.M{"username": username}).Decode(&result)
	if err == nil {

		//Check if the password matches
		if password == result.Password{
		// Remove the user document from the user_details

		err:=Cli.VolumeRemove(context.TODO(),username,true)
		if err!=nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_,err = db.CollectionHandler.DeleteOne(context.TODO(),bson.M{"username": username})
		if err!=nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)} else{
			w.WriteHeader(http.StatusUnauthorized)   // 401
			return
		}	

	} else{

		w.WriteHeader(http.StatusNotFound)  //   404
}
}

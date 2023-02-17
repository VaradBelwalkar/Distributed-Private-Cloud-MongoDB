package main

import (
	"context"
	"log"
	"github.com/gorilla/mux"
	"net/http"
	"fmt"
	"os"
	"path/filepath"
    db "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/database_handling"
    qh "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/query_handling"
	as "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/auth_service"
	"github.com/docker/docker/client"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/gorilla/handlers"

)


//const url = "mongodb://host1:27017,host2:27017,host3:27017/?replicaSet=myRS"

const url = "mongodb://localhost:27017/"


// The main function manages all the query handling and manages the database as well
func main() {

	as.Sessions=make(map[string]string)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
			log.Fatal(err)
	}

	fmt.Println(dir)
    // server main method

    var router = mux.NewRouter()

    //Initiate Mongo client
	mongo_client, err := mongo.NewClient(options.Client().ApplyURI(url)) 
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	err = mongo_client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer mongo_client.Disconnect(ctx)



    // Initiate Docker client
    qh.Cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
       panic(err)
    }
    defer qh.Cli.Close()

	//Get handler for the "user_details" collection (creates collection if not exists)
    db.CollectionHandler,db.Sys_CollectionHandler=db.InitiateMongoDB(mongo_client);
    
    //login to be handled separatly

	router.HandleFunc("/", as.RenderForm)								//DONE			
	router.HandleFunc("/login", as.LoginHandler).Methods("POST")		//DONE
	router.HandleFunc("/login", as.RenderForm).Methods("GET")			//DONE
	router.HandleFunc("/logout", as.LogoutHandler).Methods("POST")		//DONE
	router.HandleFunc("/container/run/{image}", qh.Container_Run).Methods("GET")
	router.HandleFunc("/container/run/{image}", qh.Container_Run).Methods("POST")
	router.HandleFunc("/container/resume/{container}", qh.Container_Resume).Methods("GET")
	router.HandleFunc("/container/stop/{container}", qh.Container_Stop).Methods("GET")
	router.HandleFunc("/container/remove/{container}", qh.Container_Remove).Methods("GET")
	router.HandleFunc("/container/list/containers", qh.Container_List).Methods("GET")
    router.HandleFunc("/register",as.RenderForm).Methods("GET")	   	//DONE
	router.HandleFunc("/register",qh.RegisterUser).Methods("POST")     //DONE
	router.HandleFunc("/remove_account",as.RenderForm).Methods("GET")	//DONE
	router.HandleFunc("/remove_account",qh.RemoveAccount).Methods("POST")  //DONE

	http.Handle("/", router)
	http.ListenAndServe(":8000", handlers.LoggingHandler(os.Stdout, http.DefaultServeMux))
}
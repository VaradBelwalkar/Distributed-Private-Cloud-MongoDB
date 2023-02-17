package query_handling

import (

	"context"
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	ca "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/container_apis"
	as "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/auth_service"
	"github.com/docker/docker/client"
)
// Here the user will be authenticated first and then request will be fulfilled

var Cli *client.Client
type responseStruct struct{
	privatekey string
	port string
}

type infoStruct struct{
	containerInfo []string
}


  // HandlerFunc to be registered
func Container_Run(w http.ResponseWriter, r *http.Request) {
	check,username:=as.Handle_auth(w,r)
	if check!=true{
		return
	}
	//Extracting required string from the request Structure
	vars := mux.Vars(r)
	//Get the requested Image from from the request-URL and pass it to the Container handler
	privateKey,Port,err:=ca.ContainerCreate(context.TODO(),Cli,vars["image"],username)	
	if err!=200{
		if err ==500{
		w.WriteHeader(http.StatusInternalServerError)
		return
		}
		if err==403{
			w.WriteHeader(http.StatusForbidden) // Means no more than 5 containers are allowed
			return
		} 
		if err == 404{
			w.WriteHeader(http.StatusNotFound)
			return
		}
		return
	}

	resp:=map[string]string{"privatekey":privateKey,"port":Port}
	b, _ := json.Marshal(resp)
	w.Write(b)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	//Send reponse in the body


}
  
func Container_Resume(w http.ResponseWriter, r *http.Request) {
	check,username:=as.Handle_auth(w,r)
	if check!=true{
		return
	}

	//Extracting required string from the request Structure
	vars := mux.Vars(r)
	//Get the requested Image from from the request-URL and pass it to the Container handler
	privateKey,Port,err:=ca.ContainerStart(context.TODO(),Cli,vars["container"],username)	
	if err!=200{
		if err ==500{
		w.WriteHeader(http.StatusInternalServerError)
		return
		}else if err==404{
			w.WriteHeader(http.StatusNotFound)
			return
		}
		return
	}
	resp:=map[string]string{"privatekey":privateKey,"port":Port}
	b, _ := json.Marshal(resp)
	w.Write(b)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	//Send reponse in the body


}
  
func Container_List(w http.ResponseWriter, r *http.Request) {
	check,username:=as.Handle_auth(w,r)
	if check!=true{
		return
	}

 	ctx := context.Background()
	//Extracting required string from the request Structure
	//Get the requested Image from from the request-URL and pass it to the Container handler
	containerInfo,err:=ca.OwnedContainerInfo(ctx,username)	
	if err!=200{
		if err ==500{
		w.WriteHeader(http.StatusInternalServerError)
		return
		}
		return
	}
	b, _ := json.Marshal(containerInfo)
	w.Write(b)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	//Send reponse in the body
	

}

func Container_Stop(w http.ResponseWriter, r *http.Request) {
	check,username:=as.Handle_auth(w,r)
	if check!=true{
		return
	}
 	ctx := context.Background()
	//Extracting required string from the request Structure
	vars := mux.Vars(r)
	//Get the requested Image from from the request-URL and pass it to the Container handler
	err:=ca.ContainerStop(ctx,Cli,vars["container"],username)	
	if err!=200{
		if err ==500{
		w.WriteHeader(http.StatusInternalServerError)
		return
		}
		if err== 404{
			w.WriteHeader(http.StatusNotFound)
			return
		}
	return
	}

	w.WriteHeader(http.StatusOK)

	//Send reponse in the body


}


func Container_Remove(w http.ResponseWriter, r *http.Request) {
	check,username:=as.Handle_auth(w,r)
	if check!=true{
		return
	}

 	ctx := context.Background()
	//Extracting required string from the request Structure
	vars := mux.Vars(r)
	//Get the requested Image from from the request-URL and pass it to the Container handler
	err:=ca.ContainerRemove(ctx,Cli,vars["container"],username)	
	if err!=200{
		if err ==500{
		w.WriteHeader(http.StatusInternalServerError)
		return
		}
		if err == 404{
			w.WriteHeader(http.StatusNotFound)
			return
		}
		return
	}
	
	w.WriteHeader(http.StatusOK)

	//Send reponse in the body


}


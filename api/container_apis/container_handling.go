package container_apisyy

import (
	"context"
	"io/ioutil"
	"strings"
	"bytes"
	"archive/tar"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/docker/docker/api/types"
	"github.com/docker/go-connections/nat"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"crypto/rsa"
	"crypto/rand"
	"crypto/x509"
	"golang.org/x/crypto/ssh"
	"encoding/pem"
	db "github.com/VaradBelwalkar/Private-Cloud-MongoDB/api/database_handling"
)

type resultStruct struct{
	Username string `bson:"username"`
	Password string `bson:"password"`
	ContainerInfo map[string]interface{} `bson:"containerInfo"`
	TotalOwnedContainers int `bson:"totalOwnedContainers,omitempty"`
}


//Creating private-public key pairs to be used by the client to ssh into registered container
func MakeSSHKeyPair() (string, string, int) {
    privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
    if err != nil {
        return "", "", 500
    }

    // generate and write private key as PEM
    var privKeyBuf strings.Builder

    privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
    if err := pem.Encode(&privKeyBuf, privateKeyPEM); err != nil {
        return "", "", 500
    }

    // generate and write public key
    pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
    if err != nil {
        return "", "", 500
    }

    var pubKeyBuf strings.Builder
    pubKeyBuf.Write(ssh.MarshalAuthorizedKey(pub))

    return  privKeyBuf.String(),pubKeyBuf.String(), 200
}


//Function to return err if document not found
func get_document(ctx context.Context,username string)(resultStruct, int){

	documentData:=resultStruct{}
	//Check user-document exists in the collection 
	//document_handler of type *SingleResult, see github code for more details
	err := db.CollectionHandler.FindOne(ctx,bson.M{"username":username}).Decode(&documentData)
	//If not then use following	
	if err != nil {		
			return resultStruct{},500	// Internal server error
	}	
return documentData,200

}


//Create a new container
func ContainerCreate(ctx context.Context,cli *client.Client,imageName string,username string) (string,string,int){
	documentData, status := get_document(context.TODO(),username)
	if status!= 200{
		return "","",status
	}
	//Here we get the document to work with

	if imageName !="some_ubuntu"{
		return "","",404
	}
	
	totalOwnedContainers := documentData.TotalOwnedContainers
	
		if totalOwnedContainers>=5 {
			return "","",403		// http.StatusForbidden  //This is because system cannot allocate more than 5 containers per user
		}
	
	//Else do allocate the container	
	containerCfg := &container.Config {
		Image: imageName,
		AttachStdin:false,
		AttachStdout:false,
		AttachStderr:false,
		OpenStdin:false,
		Cmd: []string{"service","ssh","start", "-D", "daemon on;"},
		ExposedPorts: nat.PortSet{
			//nat.Port("443/tcp"): {},
			nat.Port("22/tcp"): {},
		},
	}	
	volumeBinding:=username+":/mnt:rw"
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			//nat.Port("443/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
			nat.Port("22/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0"}}, //Here excluding HostPort to assign a random port 
		},
		Binds: []string{volumeBinding},
	}
	
    resp, err := cli.ContainerCreate(ctx,containerCfg,hostConfig,nil,nil,"")
    if err != nil {
        return "","",500
    }

    if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
        return "","",500
    }
	
	privateKey,publicKey,check:= MakeSSHKeyPair()
	if check!=200 {
		
		return "","",check
	}
	
	//First make a tar archive for the public key generated above 
	//buf := strings.NewReader(publicKey)

	str := []byte(publicKey)
	b := new(bytes.Buffer)

	// Create a new tar archive
	tw := tar.NewWriter(b)

	// Add the string to the archive
	hdr := &tar.Header{
		Name: "authorized_keys",
		Mode: 0600,
		Size: int64(len(str)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "","",500
	}
	if _, err := tw.Write(str); err != nil {
		return "","",500
	}
	if err := tw.Close(); err != nil {
		return "","",500
	}	
	r := bytes.NewReader(b.Bytes())

	t, err := ioutil.ReadAll(r)
	if err != nil {
		return "","",500
	}
	tempString:=string(t)
	readBuf := strings.NewReader(tempString)
	err =cli.CopyToContainer(context.Background(), resp.ID, "/root/.ssh/",readBuf ,types.CopyToContainerOptions{})
	if err!=nil{
		return "","",500
	}
	
	// Handle db call to store the resp.ID into the appropriate row for the user
	containerJSON,err:=cli.ContainerInspect(ctx,resp.ID)
	if err!=nil{		
	return "","",500
	}

	port:=containerJSON.NetworkSettings.NetworkSettingsBase.Ports["22/tcp"][0].HostPort
	containerName:=username+"_"+port
	// Here count is updated but not container information, hence do update that
	//opts := options.Update().SetUpsert(true)
	//fmt.Println(opts)
	filter:=bson.M{
		"username":username,
	}

	update:=bson.M{ "$set":bson.M{
		"totalOwnedContainers":totalOwnedContainers+1,
		"containerInfo."+containerName:bson.M{"containerID":resp.ID,"port":port,"status":"running"},
	}}

	updateResult,err:=db.CollectionHandler.UpdateOne(ctx,filter,update)
	if err!=nil || updateResult.MatchedCount!=1{
		return "","",500
	}
return privateKey,port,200
}



//Stop the container 
func ContainerStop(ctx context.Context,cli *client.Client,containerName string,username string) int{
	//** Check if containerName is valid or ContainerStart requires id to start the container
	//Handle db call to retrieve the 'id' for the container required to start the container
	//var documentData map[string]interface{} 
	//Make db call to retrieve user info about the containers it holds
	//var err error
	documentData, status := get_document(context.TODO(),username)
	if status!= 200{
		return status
	}
	var containerID string

	nesting1:=documentData.ContainerInfo//[containerName].(map[string]interface{})["containerID"].(string)
	if nesting2,ok:= nesting1[containerName]; ok{
		containerID=nesting2.(map[string]interface{})["containerID"].(string)
	} else{
		return 404 // Send StatusNotFound
	}

	stopOpts:=container.StopOptions{}
    if err := cli.ContainerStop(context.TODO(),containerID,stopOpts); err != nil {
        return 500
    }

	filter:=bson.M{
		"username":username,
	}
	update:=bson.M{ "$set":bson.M{
		"containerInfo."+containerName+"."+"status":"stopped",
	}}

	updateResult,err:=db.CollectionHandler.UpdateOne(ctx,filter,update)
	if err!=nil || updateResult.MatchedCount!=1{
		return 500
	}	
	
	return 200

}


//Stop the container 
func ContainerRemove(ctx context.Context,cli *client.Client,containerName string,username string) int{
	//** Check if containerName is valid or ContainerStart requires id to start the container
	//Handle db call to retrieve the 'id' for the container required to start the container
	//var documentData map[string]interface{} 
	//Make db call to retrieve user info about the containers it holds
	//var err error
	documentData,err := get_document(context.TODO(),username)
	if err!=200{
		return err
	}
	totalOwnedContainers:=documentData.TotalOwnedContainers
	var containerID string

	nesting1:=documentData.ContainerInfo//[containerName].(map[string]interface{})["containerID"].(string)
	if nesting2,ok:= nesting1[containerName]; ok{
		containerID=nesting2.(map[string]interface{})["containerID"].(string)
	} else{
		return 404 // Send StatusNotFound
	}

	options:=types.ContainerRemoveOptions{Force:true}

    if err := cli.ContainerRemove(context.TODO(), containerID,options); err != nil {
        return 500
    }

	filter:=bson.M{
		"username":username,	
	}
	update:=bson.M{ "$unset":bson.M{
		"containerInfo."+containerName:"",
	},
	"$set":bson.M{"totalOwnedContainers":totalOwnedContainers-1},

}
	updateResult,check:=db.CollectionHandler.UpdateOne(ctx,filter,update)
	if check!=nil || updateResult.MatchedCount!=1{
		return 500
	}		
	return 200

}



//Start the container if already created
func ContainerStart(ctx context.Context,cli *client.Client,containerName string,username string) (string,string,int){
	//** Check if containerName is valid or ContainerStart requires id to start the container
	//Handle db call to retrieve the 'id' for the container required to start the container

	documentData,status := get_document(ctx,username)
	if status!=200{
		return "","",status //Here the system failure has occured
	}


	var containerID string

	nesting1:=documentData.ContainerInfo//[containerName].(map[string]interface{})["containerID"].(string)
	if nesting2,ok:= nesting1[containerName]; ok{
		containerID=nesting2.(map[string]interface{})["containerID"].(string)
	} else{
		return "","",404 // Send StatusNotFound
	}
	// If returns error if container is already running, first do inspect the container and then only run



	containerINFO, errCase := cli.ContainerInspect(context.Background(), containerID)
	if errCase != nil {
		return "","",500
	}
	var oldPort string
	if containerINFO.State.Running == false{
    if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
        return "","",500
    }
} else {
	oldPort=containerINFO.NetworkSettings.NetworkSettingsBase.Ports["22/tcp"][0].HostPort
}

	privateKey,publicKey,err:= MakeSSHKeyPair()
	if err!=200 {
		return "","",err
	}

	str := []byte(publicKey)
	b := new(bytes.Buffer)

	// Create a new tar archive
	tw := tar.NewWriter(b)

	// Add the string to the archive
	hdr := &tar.Header{
		Name: "authorized_keys",
		Mode: 0600,
		Size: int64(len(str)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return "","",500
	}
	if _, err := tw.Write(str); err != nil {
		return "","",500
	}
	if err := tw.Close(); err != nil {
		return "","",500
	}
	
	r := bytes.NewReader(b.Bytes())

	t, errInfo := ioutil.ReadAll(r)
	if errInfo != nil {
		return "","",500
	}
	tempString:=string(t)
	readBuf := strings.NewReader(tempString)
	check :=cli.CopyToContainer(context.Background(), containerID, "/root/.ssh/", readBuf,types.CopyToContainerOptions{})
	if check!=nil{
		return "","",500
	}
	if containerINFO.State.Running == false{
	containerJSON,check:=cli.ContainerInspect(context.TODO(),containerID)
	if check!=nil{
	return "","",500
	}
	port:=containerJSON.NetworkSettings.NetworkSettingsBase.Ports["22/tcp"][0].HostPort
	newContainerName:=username+"_"+port
	

	filter:=bson.M{
		"username":username,
	}
	update:=bson.M{ "$unset":bson.M{
		"containerInfo."+containerName:"",
	},
	"$set":bson.M{
		"containerInfo."+newContainerName:bson.M{"containerID":containerID,"port":port,"status":"running"},
	},
	}
	updateResult,check:=db.CollectionHandler.UpdateOne(ctx,filter,update)
	if check!=nil || updateResult.MatchedCount!=1{
		return "","",500
	}
	
	return privateKey,port,200
	
	}else{
		return privateKey,oldPort,200
	}

}

//Gives information about the containers that user holds
func OwnedContainerInfo(ctx context.Context,username string)(map[string]string,int){

	documentData,err := get_document(context.TODO(),username)
	if err!=200{
		return nil,err
	}
	containerMap:=make(map[string]string)
	for k, v := range documentData.ContainerInfo {
		containerMap[k]=v.(map[string]interface{})["status"].(string)
	}
	return containerMap,200

}



//Gives images available on the server (the information is available on "system_details" collection)
func ImageInfo(ctx context.Context,cli *client.Client) []string{
	
	var imageArray=[]string{"ubuntu","development_server"}

	return imageArray

}












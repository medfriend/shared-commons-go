package zmq

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/medfriend/shared-commons-go/util/consul"
	"github.com/pebbe/zmq4"
)

func ConnZMQ(consulClient *api.Client, service string) (*zmq4.Socket, error) {
	serviceInfo, err := consul.GetKeyValue(consulClient, service)
	if err != nil {
		return nil, err
	}

	var resultServiceInfo map[string]string
	err = json.Unmarshal([]byte(serviceInfo), &resultServiceInfo)
	if err != nil {
		return nil, err
	}

	zmqPort := resultServiceInfo["SERVICE_PORT"]
	zmqHost := resultServiceInfo["SERVICE_PATH"]

	socket, err := zmq4.NewSocket(zmq4.PULL)
	if err != nil {
		return nil, err
	}

	zmqConn := fmt.Sprintf("tcp://%s:%s", zmqHost, zmqPort)

	// Conectar al servidor PUSH
	err = socket.Connect(zmqConn)
	if err != nil {
		socket.Close() // Close the socket if connect fails
		return nil, err
	}

	fmt.Println(fmt.Sprintf("Connected to %s", zmqConn))

	return socket, nil
}

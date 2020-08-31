package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"


)


type Status byte
const ALGO="eddsa2"


type Node struct {
	Pubkey         string `json:"pubkey"`
	BlameData      []byte `json:"data"`
	BlameSignature []byte `json:"signature"`
}

// Blame is used to store the blame nodes and the fail reason
type Blame struct {
	FailReason string `json:"fail_reason"`
	BlameNodes []Node `json:"blame_peers,omitempty"`
}

// Response keygen response
type KeyGenResponse struct {
	PubKey      string        `json:"pub_key"`
	PoolAddress string        `json:"pool_address"`
	Status      Status `json:"status"`
	Blame       Blame   `json:"blame"`
}

type KeyGenRequest struct {
	Keys []string `json:"keys"`
	Algo          string   `json:"algo"`
	
}

type KeySignRequest struct {
	PoolPubKey    string   `json:"pool_pub_key"` // pub key of the pool that we would like to send this message from
	Message       string   `json:"message"`      // base64 encoded message to be signed
	SignerPubKeys []string `json:"signer_pub_keys"`
		Algo          string   `json:"algo"`

}
func sendTestRequest(url string, request []byte) []byte {
	var resp *http.Response
	var err error
	fmt.Println(url)
	if len(request) == 0 {
		resp, err = http.Get(url)
		if err != nil {
			fmt.Println(err)
		}
	} else {
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(request))
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error!!!!!!!!!!+%v", err)
	}
	return body
}


func KeyGen2(testPubKeys []string, ip string, ports []int) string {
	var keyGenRespArr []*KeyGenResponse
	var locker sync.Mutex
	keyGenReq := KeyGenRequest{
		Keys: testPubKeys,
		Algo:ALGO,
	}
	request, err := json.Marshal(keyGenReq)
	if err != nil {
		return ""
	}

	requestGroup := sync.WaitGroup{}
	for i := 0; i < len(ports); i++ {
		requestGroup.Add(1)
		go func(i int, request []byte, keygenRespAddr *[]*KeyGenResponse, locker *sync.Mutex) {
			//sleepTime := time.Second * time.Duration(rand.Int()%5)
			//fmt.Printf("we sleep -------%v\n", sleepTime)
			//time.Sleep(sleepTime)
			defer requestGroup.Done()
			url := fmt.Sprintf("http://%s:%d/keygen", ip, ports[i])
			respByte := sendTestRequest(url, request)
			if respByte ==nil{
				fmt.Println("error in send keygen")
				return
			}
			var tempResp KeyGenResponse
			err = json.Unmarshal(respByte, &tempResp)
			if err != nil {
				fmt.Printf("22222>>>>>err%v, and bytes=%v\n", err, respByte)
			}
			locker.Lock()
			*keygenRespAddr = append(*keygenRespAddr, &tempResp)
			locker.Unlock()
		}(i, request, &keyGenRespArr, &locker)

	}

	requestGroup.Wait()

	for i := 0; i < len(ports); i++ {
		fmt.Printf("%d------%s\n", i, keyGenRespArr[i].PubKey)
	}
	return keyGenRespArr[0].PubKey
}



func KeySign2(poolPubKey string, ip string, ports []int, signersPubKey []string) {

	msg := base64.StdEncoding.EncodeToString([]byte("hello"))

	keySignReq := KeySignRequest{
		PoolPubKey:    poolPubKey,
		Message:       msg,
		SignerPubKeys: signersPubKey,
		Algo:ALGO,
	}
	request, _ := json.Marshal(keySignReq)
	requestGroup := sync.WaitGroup{}
	for i := 0; i < len(ports); i++ {
		requestGroup.Add(1)
		go func(idx int, request []byte) {
			defer requestGroup.Done()
			url := fmt.Sprintf("http://%s:%d/keysign", ip, ports[idx])
			if idx != 2 {
				time.Sleep(time.Second * time.Duration(rand.Int()%5))
			}
			respByte := sendTestRequest(url, request)
			if respByte ==nil{
				fmt.Println("error in send keysign")
				return
			}
			fmt.Printf("current time=%s::%d:--> %s\n",time.Now().Local(), idx, respByte)

		}(i, request)
	}
	requestGroup.Wait()

}


func main () {
	testPubKeys := []string{"thorpub1addwnpepqtdklw8tf3anjz7nn5fly3uvq2e67w2apn560s4smmrt9e3x52nt2svmmu3", "thorpub1addwnpepqtspqyy6gk22u37ztra4hq3hdakc0w0k60sfy849mlml2vrpfr0wvm6uz09", "thorpub1addwnpepq2ryyje5zr09lq7gqptjwnxqsy2vcdngvwd6z7yt5yjcnyj8c8cn559xe69", "thorpub1addwnpepqfjcw5l4ay5t00c32mmlky7qrppepxzdlkcwfs2fd5u73qrwna0vzag3y4j"}
	ip := "127.0.0.1"
	ports := []int{8320, 8321, 8322, 8323}
	poolAddr := KeyGen2(testPubKeys, ip, ports)
	fmt.Println(poolAddr)
	for i := 0; i < 1; i++ {
		KeySign2(poolAddr, ip, ports, testPubKeys[:3])
	}
}

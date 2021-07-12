package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gitlab.com/thorchain/tss/go-tss/keygen"
	"gitlab.com/thorchain/tss/go-tss/keysign"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func sendTestRequest(url, algo string, request []byte) []byte {
	var resp *http.Response
	var err error
	fmt.Printf("%v>>%v\n", algo, url)
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

func KeyGenAll(testPubKeys []string, ip string, ports []int, algo string) (string, string) {
	var keyGenRespArr [][]keygen.Response
	var locker sync.Mutex
	keyGenReq := keygen.NewRequest(testPubKeys, 10, "0.14.0", algo)
	request, err := json.Marshal(keyGenReq)
	if err != nil {
		return "", ""
	}

	requestGroup := sync.WaitGroup{}
	for i := 0; i < len(ports); i++ {
		requestGroup.Add(1)
		go func(i int, request []byte, locker *sync.Mutex) {
			//sleepTime := time.Second * time.Duration(rand.Int()%5)
			//fmt.Printf("we sleep -------%v\n", sleepTime)
			//time.Sleep(sleepTime)
			defer requestGroup.Done()
			url := fmt.Sprintf("http://%s:%d/keygenall", ip, ports[i])
			respByte := sendTestRequest(url, algo, request)
			if respByte == nil {
				fmt.Println("error in send keygen")
				return
			}
			var tempResp []keygen.Response
			err = json.Unmarshal(respByte, &tempResp)
			if err != nil {
				fmt.Println("error in unmarshal the data")
			}
			locker.Lock()
			keyGenRespArr = append(keyGenRespArr, tempResp)
			locker.Unlock()
		}(i, request, &locker)

	}
	requestGroup.Wait()
	return keyGenRespArr[0][0].PubKey, keyGenRespArr[0][1].PubKey
}

func KeyGen(testPubKeys []string, ip string, ports []int, algo string) string {
	var keyGenRespArr []*keygen.Response
	var locker sync.Mutex
	keyGenReq := keygen.NewRequest(testPubKeys, 10, "0.14.0", algo)
	request, err := json.Marshal(keyGenReq)
	if err != nil {
		return ""
	}

	requestGroup := sync.WaitGroup{}
	for i := 0; i < len(ports); i++ {
		requestGroup.Add(1)
		go func(i int, request []byte, locker *sync.Mutex) {
			//sleepTime := time.Second * time.Duration(rand.Int()%5)
			//fmt.Printf("we sleep -------%v\n", sleepTime)
			//time.Sleep(sleepTime)
			defer requestGroup.Done()
			url := fmt.Sprintf("http://%s:%d/keygen", ip, ports[i])
			respByte := sendTestRequest(url, algo, request)
			if respByte == nil {
				fmt.Println("error in send keygen")
				return
			}
			var tempResp keygen.Response
			err = json.Unmarshal(respByte, &tempResp)
			if err != nil {
				fmt.Println("error in unmarshal the data")
			}
			locker.Lock()
			keyGenRespArr = append(keyGenRespArr, &tempResp)
			locker.Unlock()
		}(i, request, &locker)

	}

	requestGroup.Wait()
	return keyGenRespArr[0].PubKey
}

func KeySign2(poolPubKey, msg1Str, msg2Str string, ip string, blockHeight int64, ports []int, signersPubKey []string, algo string) {

	msg1 := base64.StdEncoding.EncodeToString([]byte(msg1Str))
	msg2 := base64.StdEncoding.EncodeToString([]byte(msg2Str))

	keySignReq := keysign.NewRequest(poolPubKey, []string{msg1, msg2}, blockHeight, signersPubKey, "0.14.0", algo)
	request, _ := json.Marshal(keySignReq)

	keySignReq2 := keysign.NewRequest(poolPubKey, []string{msg1, msg2}, blockHeight, signersPubKey[1:], "0.14.0", algo)

	request2, _ := json.Marshal(keySignReq2)
	_ = request2
	requestGroup := sync.WaitGroup{}
	var result []keysign.Response
	locker := sync.Mutex{}
	for i := 0; i < len(ports); i++ {
		requestGroup.Add(1)
		go func(idx int, request []byte, locker *sync.Mutex) {
			defer requestGroup.Done()
			var respByte []byte
			url := fmt.Sprintf("http://%s:%d/keysign", ip, ports[idx])
			//if idx != 2 {
			//	time.Sleep(time.Second * time.Duration(rand.Int()%5))
			//}
			if idx == 1000000 {
				respByte = sendTestRequest(url, algo, request)
				if respByte == nil {
					fmt.Println("error in send keysign")
					return
				}
			} else {
				respByte = sendTestRequest(url, algo, request)
				if respByte == nil {
					fmt.Println("error in send keysign")
					return
				}
			}

			var ret keysign.Response
			err := json.Unmarshal(respByte, &ret)
			if err != nil {
				fmt.Println("error in send keysign")
				return
			}
			if ret.Status != 1 {
				panic("error")
			}
			locker.Lock()
			result = append(result, ret)
			locker.Unlock()
		}(i, request, &locker)
	}
	requestGroup.Wait()

	if len(result) != 4 {
		panic("error in keysign verification")
	}

	fmt.Printf("---------------\n")
	for _, el := range result[0].Signatures {
		fmt.Printf("%v-%v---s:%v----r:%v\n", algo, el.Msg, el.S, el.R)
	}
	fmt.Printf("---------------\n")

}

func doJob(job chan int, wg *sync.WaitGroup, ip string, algos, poolKeys, testPubKeys []string, ports []int) {
	defer wg.Done()
	for {
		i, ok := <-job
		if !ok {
			return
		}

		choose := rand.Intn(2)
		algo := algos[choose]
		poolKey := poolKeys[choose]
		msg1 := "he" + strconv.Itoa(i)
		msg2 := "hee" + strconv.Itoa(i)
		KeySign2(poolKey, msg1, msg2, ip, int64(i), ports, testPubKeys[:], algo)
	}

}

func main() {
	testPubKeys := []string{"thorpub1addwnpepqtdklw8tf3anjz7nn5fly3uvq2e67w2apn560s4smmrt9e3x52nt2svmmu3", "thorpub1addwnpepqtspqyy6gk22u37ztra4hq3hdakc0w0k60sfy849mlml2vrpfr0wvm6uz09", "thorpub1addwnpepq2ryyje5zr09lq7gqptjwnxqsy2vcdngvwd6z7yt5yjcnyj8c8cn559xe69", "thorpub1addwnpepqfjcw5l4ay5t00c32mmlky7qrppepxzdlkcwfs2fd5u73qrwna0vzag3y4j"}
	ip := "127.0.0.1"
	ports := []int{8320, 8321, 8322, 8323}
	wg := &sync.WaitGroup{}
	algos := []string{"ecdsa", "eddsa"}

	//poolAddrEcdsa := KeyGen(testPubKeys, ip, ports, "ecdsa")
	//pubkey1, pubkey2 := KeyGenAll(testPubKeys, ip, ports, "")
	//fmt.Printf("ecdsa:%v\neddsa:%v\n", pubkey1, pubkey2)
	//return
	poolAddrEcdsa := "thorpub1addwnpepqdpk0vtztrwdy3la57p4facsvs4fk6l59dkecvsrgldy9lyaga5ugntqqlz"
	poolAddrEddsa := "thorpub1zcjduepqhyy3gmt22uv3fnnnq9qls27f2suda3f68vasxlqa2cqzg8sag4qsmx26fm"
	poolkeys := []string{poolAddrEcdsa, poolAddrEddsa}
	_ = poolAddrEcdsa
	_ = poolAddrEddsa
	rand.Seed(time.Now().UnixNano())
	fmt.Println(algos)

	taskNum := 4
	jobChan := make(chan int, taskNum)

	base := 100
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := base; i < base+20; i++ {
			jobChan <- i
		}
		close(jobChan)
	}()

	for i := 0; i < taskNum; i++ {
		wg.Add(1)
		go doJob(jobChan, wg, ip, algos, poolkeys, testPubKeys[:], ports)
	}

	wg.Wait()
	fmt.Println("quit")
}

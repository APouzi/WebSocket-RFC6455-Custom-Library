package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"hash"
	"net"
	"net/http"
	"strings"
)

type WebSocketContainer struct{
	WSKey	string
	Hash	hash.Hash
	buffRW	*bufio.ReadWriter
	tcpConn net.Conn
}

func main() {
	fmt.Println("WebSocket Server Has Initiated")
	newContainer := WebSocketContainer{}
	http.HandleFunc("/",newContainer.WebSocketUpgrader)
	http.ListenAndServe(":9001",nil)
}
func HashAndNonce(wskey string) string{
	hashbuilder := sha1.New()
	hashbuilder.Write([]byte(wskey))
	hashbuilder.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	hashByte := hashbuilder.Sum(nil)
	return base64.StdEncoding.EncodeToString(hashByte)
}
func(wsc *WebSocketContainer) WebSocketUpgrader(w http.ResponseWriter, r *http.Request) {
	wsKey := r.Header.Get("Sec-WebSocket-Key")
	hash := sha1.New()
	hj := w.(http.Hijacker)//Type asserion. asserts that the interface value w holds a concrete type http.Hijacker. If w indeed does hold an http.Hijacker, then it will assign that value to hj. If it does not, a panic will occur at runtime.
	tcpConn, bfwr,err := hj.Hijack()
	if err != nil{
		fmt.Println("error in attempting to hijack this")
		return
	}
	wsc.WSKey = wsKey
	wsc.Hash = hash
	wsc.buffRW = bfwr
	wsc.tcpConn = tcpConn
	var strbdr strings.Builder
	strbdr.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	strbdr.WriteString("Upgrade: WebSocket\r\n") 
	strbdr.WriteString("Connection: Upgrade\r\n")
	strbdr.WriteString("Sec-WebSocket-Accept: ")
	strbdr.WriteString(HashAndNonce(wsKey) + "\r\n")
	strbdr.WriteString("" + "\r\n")
	strbdr.WriteString("")
	fmt.Println(strbdr.String())
	wsc.tcpConn.Write([]byte(strbdr.String()))

	wsc.WebSocketLoop()
}

func(wsc *WebSocketContainer) ReceiveFrame(w http.ResponseWriter, r *http.Request) {
	
}
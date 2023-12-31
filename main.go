package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
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

func(wsc *WebSocketContainer) WebSocketLoop(){
	for{
		frame := wsc.ReceiveFrameStart()
		if frame.Opcode == 0x08{
			fmt.Println("closed connection")
			wsc.tcpConn.Close() // maybe defer this
			return
		}
		if frame.Opcode == 1{
			frame, _ = wsc.ReadFramePayloadStart(frame)
			wsc.ReadPayloadWithMask(frame)
			fmt.Println("frame opcode is text")
			// We are just inserting here a message send off regarding 
			// SendDataFrameCanned(*frame,wsc.buffRW)
		}
		if frame.Opcode == 2{
			fmt.Println("frame opcode is binary")
			frame, _ = wsc.ReadFramePayloadStart(frame)
		}
		
		
	
		fmt.Println("loop iteration done")
	}
}
type Frame struct{
	FIN byte
	Opcode byte
	Mask byte
	MaskPayLoad []byte
	PayloadLength uint64
}
func(wsc *WebSocketContainer) ReceiveFrameStart() *Frame{
	data := make([]byte, 1)
	_, err := wsc.buffRW.Read(data)
	if err != nil{
		fmt.Println("reading error")
	}
	
	frame := Frame{}
	frame.FIN = data[0] & 0x80
	frame.Opcode = data[0] & 0x0F
	
	fmt.Println("data frame ",int32(data[0]))
	fmt.Println("opcode: ",int32(frame.Opcode))
	fmt.Println("FIN: ",int32(frame.FIN))
	
	return &frame
}

func (wsc *WebSocketContainer) ReadFramePayloadStart(frame *Frame)  (*Frame,bool){
	data := make([]byte, 1)
	_,err := wsc.buffRW.Read(data)
	frame.Mask = data[0] & 0x80
	payloadLength := uint64(data[0] & 0x7F)
	if payloadLength < 126{
		frame.PayloadLength = payloadLength
	}
	if payloadLength == 126{
		getPayloadLength := make([]byte, 2)
		_, err := io.ReadFull(wsc.buffRW, getPayloadLength)
		if err != nil{
			fmt.Println("error reading payloadlength")
		}
		frame.PayloadLength = uint64(binary.BigEndian.Uint16(getPayloadLength))
	}
	if payloadLength == 127{
		getPayloadLength := make([]byte, 8)
		_, err := io.ReadFull(wsc.buffRW, getPayloadLength)
		if err != nil{
			fmt.Println("error reading payloadlength")
		}
		frame.PayloadLength = uint64(binary.BigEndian.Uint64(getPayloadLength))
	}
	mask := make([]byte,4)
	if frame.Mask != 0x00{
		_, err := wsc.buffRW.Read(mask)
		if err != nil{
			fmt.Println("error reading mask")
		}
	}
	frame.MaskPayLoad = mask	
	if err != nil{
		fmt.Println("reading error")
	}
	return frame, false
}

func (wsc *WebSocketContainer) ReadPayloadWithMask(frame *Frame){
	payload := make([]byte, frame.PayloadLength)
	
	dataRead := 0
	for {
		fmt.Println("read in loop:", dataRead)
		n, _ := wsc.buffRW.Read(payload)
		dataRead+=n
		if dataRead == int(frame.PayloadLength){
			break
		}
	}

	payload = MaskOrUnMask(payload,frame.PayloadLength,frame.MaskPayLoad)
	// for i := uint64(0); i < frame.PayloadLength; i++{
	// 	payload[i] ^= frame.MaskPayLoad[i%4]
	// }
	
	
	fmt.Println("payload: ",string(payload))
}

func MaskOrUnMask(payload []byte, length uint64,mask []byte) []byte{
	var i uint64
	for i = 0; i < length; i++{
		payload[i] ^= mask[i%4]
	}
	return payload
}


func SendDataFrameCanned(frame Frame, buffwr *bufio.ReadWriter){
	sendFrame := Frame{}
	sendFrame.FIN = frame.FIN
	sendFrame.Mask = 0x00
	sendFrame.PayloadLength = 0x28
	sendFrame.Opcode = frame.Opcode
	sendOff := "Message was successfully received! Cool!"
	data := []byte{}
	data = append(data, sendFrame.FIN)
	// data = append(data, sendFrame.Mask)
	// data = append(data, sendFrame.Opcode)
	data = append(data, 0x28)
	fmt.Println(len(sendOff))
	for _, v := range sendOff{
		data = append(data, byte(v))
	}
	
	buffwr.Write(data)
	buffwr.Flush()
	
}

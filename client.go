package main

import (
	"fmt"
	"net"
	"bytes"
	"encoding/binary"
	"github.com/hajimehoshi/oto"
	"log"
	"os"
	"io"
 "github.com/hajimehoshi/go-mp3"
)

const(
	FlagACK    = 1 << 0 // 00000001
	FlagSYNC   = 1 << 1 // 00000010
	FlagAUDIO  = 1 << 2 // 00000100
	FlagSTOP   = 1 << 3 // 00001000
	FlagMETA   = 1 << 4 // 00010000
	FlagCONFIG = 1 << 5 // 00100000
	FlagCHOICE = 1 << 6 // 01000000
	FlagEND    = 1 << 7 // 10000000
	sampleRate = 44100
	numChannels = 2
	bytesPerSample = 2 // 16-bit PCM
)

type Packet struct {
    Seq   uint32
    Ack   uint32
    Flags byte
    Data  []byte
}
func createPacket(seq uint32, ack uint32, flags byte, payload []byte) []byte  {

	buf := new(bytes.Buffer)
	
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, ack)

	buf.WriteByte(flags)
	buf.Write(payload)
	return buf.Bytes()
}

func DeserializePacket(buf []byte) Packet {
    seq := binary.BigEndian.Uint32(buf[0:4])
    ack := binary.BigEndian.Uint32(buf[4:8])
    flags := buf[8]
    data := buf[9:]

    return Packet{
        Seq:   seq,
        Ack:   ack,
        Flags: flags,
        Data:  data,
    }
}


func main() {

ctx, err := oto.NewContext(sampleRate, 2, 2, 16384)
if err != nil {
	log.Fatal(err)
}
	player := ctx.NewPlayer()
	defer player.Close()

    serverAddr := net.UDPAddr{
        IP:   net.ParseIP("127.0.0.1"),
        Port: 9000,
    }
    conn, err := net.DialUDP("udp", nil, &serverAddr)
    if err != nil {
        panic(err)
    }
    defer conn.Close()


    // Pedir archivo
	 songName := []byte("NoMoreTears.pcm")
   packetChoice := createPacket(0, 0, FlagCHOICE, songName)
    _, err = conn.Write(packetChoice)
    if err != nil {
        panic(err)
    }
    
    buffer := make([]byte, 1024)
    audioChan := make(chan []byte, 100) 

    go func() {
        for data := range audioChan {
		    player.Write(data)
        }
    }()

    for {
        n, err := conn.Read(buffer)
				BufferPacket := DeserializePacket(buffer)

				fmt.Println(n, "bytes recibidos")
        if  BufferPacket.Flags&FlagEND != 0 {
            fmt.Println("Fin de la transmisión:", err)
            break
        }

        if BufferPacket.Flags&FlagAUDIO != 0 {
        // Guardar chunk al player
				fmt.Println("Guardando" )
        chunk := make([]byte, n)
        copy(chunk, BufferPacket.Data)
        audioChan <- chunk

				//enviar ACK
				response := createPacket(0, BufferPacket.Seq + uint32(len(BufferPacket.Data)), FlagACK, nil)
				fmt.Println("enviando ack", response)
				conn.Write(response)
			  }
    }
	}

	func apelo()  {
		f, _ := os.Open("/home/fran/Downloads/apelo.mp3")
decoder, _ := mp3.NewDecoder(f)
fmt.Println("decodificando...")
pcmData, _ := io.ReadAll(decoder)
sampleRate := decoder.SampleRate()
ctx, err := oto.NewContext(sampleRate, 2, 2, 8192)
if err != nil {
	log.Fatal(err)
}


	player := ctx.NewPlayer()
	defer player.Close()
	for i := 0; i < len(pcmData); i += 1024{
	end := i + 1024

			if end > len(pcmData){
				end = len(pcmData)
			}
  fmt.Println("enviando", i, len(pcmData))
	player.Write(pcmData[i:end])
	}
	fmt.Println("out")
}

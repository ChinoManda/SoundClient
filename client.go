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
	FlagSONGS	 = 1 << 7 // 10000000
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

func handShake(conn *net.UDPConn) bool {
	ack := uint32(1000)
	handShakePacket := createPacket(0, ack, FlagSYNC, nil)
	conn.Write(handShakePacket)
  buffer := make([]byte, 1024)
  conn.Read(buffer)
	response := DeserializePacket(buffer)
	if response.Flags&FlagSYNC != 0 && response.Flags&FlagACK != 0 {
  if response.Ack == ack+1 {
   handShakePacket = createPacket(ack+1, response.Seq+1, FlagACK, nil)
	 conn.Write(handShakePacket)
	 return true
	}
	}
 return false
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

		success := handShake(conn)
	  if !success{
		fmt.Println("handShake fallido")
    log.Fatal("HandShake fallido")
		conn.Close()
		} else {
			fmt.Println("handShake valido")
		}

		//pedir lista canciones
    var choices []string
		ask := createPacket(0, 0, FlagSONGS, nil)
    _, err = conn.Write(ask)
		for {
		buf := make([]byte, 1024) 
    n, err := conn.Read(buf)
    if err != nil {
        log.Println("Error leyendo del servidor:", err)
        break
    }
    data := DeserializePacket(buf[:n])
		if data.Flags&FlagSONGS != 0 {
			choices = append(choices, string(data.Data))
		}
		if data.Flags&FlagSTOP != 0 {
			break
		}
		}
		for i := 0; i < len(choices); i++ {
			fmt.Printf("%d - %s \n", i, choices[i])
		}
		fmt.Println("Elija una cancion con el numero: ")
		var a int
		fmt.Scan(&a)
    // Pedir archivo
	 songName := choices[a]
	 fmt.Println("Reproduciendo ", songName)
   packetChoice := createPacket(0, 0, FlagCHOICE, []byte(songName))
    _, err = conn.Write(packetChoice)
    if err != nil {
        panic(err)
    }
    
    buffer := make([]byte, 2048)
    audioChan := make(chan []byte, 2048) 

    go func() {
        for data := range audioChan {
		    player.Write(data)
        }
    }()

    for {
        n, err := conn.Read(buffer)
				fmt.Println(n, "bytes recibidos")
				
				BufferPacket := DeserializePacket(buffer[:n])

        if  BufferPacket.Flags&FlagSTOP != 0 {
            fmt.Println("Fin de la transmisión:", err)
            break
        }

        if BufferPacket.Flags&FlagAUDIO != 0 {
        // Guardar chunk al player
				fmt.Println("Guardando" )
        chunk := make([]byte, len(BufferPacket.Data))
        copy(chunk, BufferPacket.Data)
        audioChan <- chunk

				//enviar ACK
				ackValue := BufferPacket.Seq + uint32(len(BufferPacket.Data))
				response := createPacket(0, ackValue, FlagACK, nil)
				fmt.Println("enviando ack", ackValue, BufferPacket.Seq, len(BufferPacket.Data))
				conn.Write(response)
			  }
    }
		select{}
	}

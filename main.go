package main

import (
	"bufio"
	"bytes"
	crypto_rand "crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/gliderlabs/ssh"
	"go.bug.st/serial"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	dataChan        = make(chan []byte, 512)
	sshChan         = make(chan []byte, 512)
	tempPassword    string
	dataBuf         []byte
	openConnections = 0
	BUFFER_LENGTH   = 16 * 1024
)

func runForPort(portName string, mode *serial.Mode) bool {

	port, err := serial.Open(portName, mode)
	if err != nil {
		//log.Printf("Failed to open port %v: %v", portName, err)
		return false
	}

	defer port.Close()

	readError := false
	buf := make([]byte, 512)
	quit := make(chan bool)

	go func() {
		for {
			select {
			case <-quit:
				return
			case dataIn := <-sshChan:
				port.Write(dataIn)
			default:
				//fmt.Println("no new inline message")
			}
		}
	}()

	for {
		n, err := port.Read(buf)
		if err != nil || n == 0 {
			readError = true
			quit <- true
			break
		}

		dataBuf = append(dataBuf, buf[:n]...)
		if len(dataBuf) > BUFFER_LENGTH {
			dataBuf = dataBuf[len(dataBuf)-BUFFER_LENGTH:]
		}

		if openConnections > 0 {
			select {
			case dataChan <- buf[:n]:
				//fmt.Println("wrote data to channel")
			default:
				//fmt.Println("no message sent from ssh")
			}
		}

	}

	if readError {
		log.Printf("Failed to read port %v", portName)
		return false
	}
	return true
}

func whileRun(port string, config *serial.Mode) {
	for {
		runForPort(port, config)
		time.Sleep(5 * time.Second)
	}
}

func readBanner() string {
	file, err := ioutil.ReadFile("banner")
	if err != nil {
		log.Print(err)
		return ""
	}

	return string(file)
}

func main() {
	parser := argparse.NewParser("serial to ssh proxy", "by Niklas Schütrumpf <niklas@mc8051.de>")
	port := parser.String("p", "port", &argparse.Options{Required: false, Help: "COM ports which should be scanned", Default: os.Getenv("COM_PORT")})

	err := parser.Parse(os.Args)
	if err != nil || *port == "" {
		log.Print(parser.Usage(err))
		os.Exit(0)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()

	buf := make([]byte, 256)
	_, err = crypto_rand.Read(buf)
	if err != nil {
		log.Fatal("cannot seed math/rand package with cryptographically secure random number generator")
	}
	rand.Seed(int64(binary.LittleEndian.Uint64(buf)))

	tempPassword = generatePassword(14, 1, 2, 4)
	log.Println(fmt.Sprintf("Temporary password: %s\n", tempPassword))

	mode := serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	go whileRun(*port, &mode)

	log.Println("starting ssh server on port 2222...")

	server := ssh.Server{
		Addr: ":2222",
		Handler: ssh.Handler(func(s ssh.Session) {
			quit := make(chan bool)
			openConnections++

			io.WriteString(s, fmt.Sprintf("%s\n", readBanner()))
			log.Println(fmt.Sprintf("New successfull connection from %s - number open connections: %d", s.RemoteAddr(), openConnections))

			if len(dataBuf) > 0 {
				io.WriteString(s, fmt.Sprintf("%s\n--- past ---\n", dataBuf))
			}

			go func() {
				for {

					select {
					case <-quit:
						log.Println("received ssh quit")
						openConnections--
						return
					case dataIn := <-dataChan:
						s.Write(dataIn)
					default:
						//log.Println("received ssh quit")
					}
				}
			}()

			buf := make([]byte, 512)

			for {
				n, err := s.Read(buf)
				if err != nil || n == 0 {
					quit <- true
					continue
				}

				// CTRL + F1
				if bytes.Equal(buf[0:6], []byte{0x1b, 0x5b, 0x31, 0x3b, 0x35, 0x50}) {
					sshChan <- []byte(fmt.Sprintf("TERM=xterm && $SHELL\n"))
					log.Print("send interactive shell command")
					continue
				}

				if buf[0] == 0x04 {
					io.WriteString(s, fmt.Sprintf("Bye - have a nice day\n"))
					s.Exit(0)
					quit <- true
					break
				}
				sshChan <- buf[:n]
			}
		}),
		PasswordHandler: ssh.PasswordHandler(func(ctx ssh.Context, password string) bool {
			return password == tempPassword
		}),
		PublicKeyHandler: ssh.PublicKeyHandler(func(ctx ssh.Context, key ssh.PublicKey) bool {
			file, err := os.Open("sshkeys")
			if err != nil {
				log.Print(err)
				return false
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				parsedKey, _, _, _, err := ssh.ParseAuthorizedKey(scanner.Bytes())
				if err == nil && ssh.KeysEqual(key, parsedKey) {
					return true
				}
			}

			if err := scanner.Err(); err != nil {
				log.Print(err)
				return false
			}

			return false
		}),
	}

	log.Fatal(server.ListenAndServe())
}

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/imroc/req/v3"
	utls "github.com/refraction-networking/utls"
)

type TLSConn struct {
	*utls.UConn
}

func (conn *TLSConn) ConnectionState() tls.ConnectionState {
	cs := conn.UConn.ConnectionState()
	return tls.ConnectionState{
		Version:                     cs.Version,
		HandshakeComplete:           cs.HandshakeComplete,
		DidResume:                   cs.DidResume,
		CipherSuite:                 cs.CipherSuite,
		NegotiatedProtocol:          cs.NegotiatedProtocol,
		ServerName:                  cs.ServerName,
		PeerCertificates:            cs.PeerCertificates,
		VerifiedChains:              cs.VerifiedChains,
		SignedCertificateTimestamps: cs.SignedCertificateTimestamps,
		OCSPResponse:                cs.OCSPResponse,
	}
}

func main() {
	marketCode := flag.String("market-code", "", "market code")
	startDate := flag.String("start-date", "", "start date, format YYYY-MM-DD: 2022-12-06 (6th December of 2022)")
	endDate := flag.String("end-date", "", "end date, format YYYY-MM-DD: 2022-12-06 (6th December of 2022)")
	timeFrame := flag.String("time-frame", "", "time frame")
	output := flag.String("output", "", "absolute path for the output json file with the respone")
	flag.Parse()

	if *marketCode == "" || *startDate == "" || *endDate == "" || *timeFrame == "" || *output == "" {
		log.Fatalln("market-code, start-date, end-date, time-frame and output flags are mandatory")
	}

	file, fileError := os.Create(*output)
	if fileError != nil {
		log.Fatalln("could not create output file:" + fileError.Error())
	}
	defer file.Close()

	client := req.C()
	client.SetDialTLS(func(ctx context.Context, network, addr string) (net.Conn, error) {
		plainConn, err := net.Dial(network, addr)
		if err != nil {
			return nil, err
		}
		colonPos := strings.LastIndex(addr, ":")
		if colonPos == -1 {
			colonPos = len(addr)
		}
		hostname := addr[:colonPos]
		utlsConfig := &utls.Config{ServerName: hostname, NextProtos: client.GetTLSClientConfig().NextProtos}
		conn := utls.UClient(plainConn, utlsConfig, utls.HelloAndroid_11_OkHttp)
		return &TLSConn{conn}, nil
	})

	endPoint := fmt.Sprintf("https://api.investing.com/api/financialdata/historical/%v?start-date=%v&end-date=%v&time-frame=%v&add-missing-rows=false", *marketCode, *startDate, *endDate, *timeFrame)

	response := client.R().SetHeader("domain-id", "www").MustGet(endPoint)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Fatalln("incorrect response status code" + response.Status)
	}

	content, contentError := io.ReadAll(response.Body)
	if contentError != nil {
		log.Fatalln("could not read the content of the response" + contentError.Error())
	}

	_, fileWriteError := io.WriteString(file, string(content))
	if fileWriteError != nil {
		log.Fatalln("could not write the content of the response to the output file" + contentError.Error())
	}

}

package anyproxy

//ProxyConnectionManager is an interface to enable multiple implementations of key proxy activity at the socket level.
// two implementations are included:
//		1. Default: DirectProxyConnection which simply echos the traffic to the target
//		2. LoggingProxyConnection which is a tool for debugging: echos traffic to both the target, and to session files.

import (
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getUniqueFilename(srcName string) string {
	return strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10) + srcName + "_src.stream"
}

type ProxyConnectionManager interface {
	CopyProxyConnection(io.ReadWriteCloser, io.ReadWriteCloser, string, string)
	SpawnBiDirectionalCopy(io.ReadWriteCloser, io.ReadWriteCloser, string, string)
}
type DirectProxyConnection struct {
	name string
}

func (into *DirectProxyConnection) SpawnBiDirectionalCopy(dst io.ReadWriteCloser, src io.ReadWriteCloser, dstName string, srcName string) {
	go into.CopyProxyConnection(dst, src, dstName, srcName)
	go into.CopyProxyConnection(src, dst, srcName, dstName)
}

func (into *DirectProxyConnection) CopyProxyConnection(dst io.ReadWriteCloser, src io.ReadWriteCloser, dstName string, srcName string) {
	if dst == nil {
		log.Debugf("copy(): oops, dst is nil!")
		return
	}
	if src == nil {
		log.Debugf("copy(): oops, src is nil!")
		return
	}
	var err error
	_, err = io.Copy(dst, src)
	if err != nil {
		if operr, ok := err.(*net.OpError); ok {
			if srcName == "directserver" || srcName == "proxyserver" {
				log.Debugf("copy(): %s->%s: Op=%s, Net=%s, Addr=%v, Err=%v", srcName, dstName, operr.Op, operr.Net, operr.Addr, operr.Err)
			}
			if operr.Op == "read" {
				if srcName == "proxyserver" {
					IncrProxyServerReadErr()
				}
				if srcName == "directserver" {
					IncrDirectServerReadErr()
				}
			}
			if operr.Op == "write" {
				if srcName == "proxyserver" {
					IncrProxyServerWriteErr()
				}
				if srcName == "directserver" {
					IncrDirectServerWriteErr()
				}
			}
		}
	}
	dst.Close()
	src.Close()
}

type LoggingProxyConnection struct {
	name string
}

func (into *LoggingProxyConnection) SpawnBiDirectionalCopy(dst io.ReadWriteCloser, src io.ReadWriteCloser, dstName string, srcName string) {
	go into.CopyProxyConnection(dst, src, dstName, srcName)
	go into.CopyProxyConnection(src, dst, srcName, dstName)
}

func (into *LoggingProxyConnection) CopyProxyConnection(dst io.ReadWriteCloser, src io.ReadWriteCloser, dstName string, srcName string) {
	if dst == nil {
		log.Debugf("copy(): oops, dst is nil!")
		return
	}
	if src == nil {
		log.Debugf("copy(): oops, src is nil!")
		return
	}
	var err error
	// RK duplicate stream
	myfilename := getUniqueFilename(srcName)
	log.Debugf("writing file", myfilename)
	f, err := os.Create(myfilename)
	check(err)
	var buf2 io.ReadWriteCloser
	buf2 = io.ReadWriteCloser(f)
	output := io.MultiWriter(dst, buf2)
	_, err = io.Copy(output, src)
	err2 := buf2.Close()
	check(err2)
	ReportStatistics(err, srcName, dstName)
	dst.Close()
	src.Close()
}

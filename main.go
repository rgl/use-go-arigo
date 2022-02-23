package main

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/myanimestream/arigo"
)

func main() {
	ariaUrl := "ws://localhost:6800/jsonrpc"

	log.Printf("connecting to aria at %s...", ariaUrl)
	c, err := arigo.Dial(ariaUrl, "")
	if err != nil {
		// see aria2c '--help=#rpc'
		// NB you can install aria2c as:
		//      debian/ubuntu: apt-get install -y aria2
		//      windows/msys2: pacman -Sy mingw-w64-x86_64-aria2
		//      windows/chocolatey: choco install -y aria2
		// NB you must start the aria2 before running this application.
		//    e.g. aria2c --enable-rpc --max-connection-per-server=4 --log=aria2.log
		log.Fatalf("failed to connect to the aria rpc server (you can start it with aria2c --enable-rpc --max-connection-per-server=4 --log=aria2.log): %v", err)
	}

	versionInfo, err := c.GetVersion()
	if err != nil {
		log.Fatalf("failed to get aria version: %v", err)
	}
	log.Printf("connected to aria %s (%s)", versionInfo.Version, strings.Join(versionInfo.EnabledFeatures, ", "))

	// // register to receive events.
	// // NB there is no progress report event.
	// //    see https://github.com/aria2/aria2/issues/839
	// c.Subscribe("downloadStart", func(event *arigo.DownloadEvent) {
	// 	log.Printf("EVENT downloadStart: %v", event)
	// })
	// c.Subscribe("downloadStop", func(event *arigo.DownloadEvent) {
	// 	log.Printf("EVENT downloadStop: %v", event)
	// })
	// c.Subscribe("downloadPause", func(event *arigo.DownloadEvent) {
	// 	log.Printf("EVENT downloadPause: %v", event)
	// })
	// c.Subscribe("downloadComplete", func(event *arigo.DownloadEvent) {
	// 	log.Printf("EVENT downloadComplete: %v", event)
	// })
	// c.Subscribe("downloadError", func(event *arigo.DownloadEvent) {
	// 	log.Printf("EVENT downloadError: %v", event)
	// })

	// // windows 2022 evaluation.
	// isoURL := "https://software-download.microsoft.com/download/sg/20348.169.210806-2348.fe_release_svc_refresh_SERVER_EVAL_x64FRE_en-us.iso"
	// isoChecksum := "sha-256=4f1457c4fe14ce48c9b2324924f33ca4f0470475e6da851b39ccbf98f44e7852"
	// isoFilename := filepath.Base(isoURL)

	// debian 11 netinst iso.
	isoURL := "http://mirrors.up.pt/debian-cd/11.2.0/amd64/iso-cd/debian-11.2.0-amd64-netinst.iso"
	isoChecksum := "sha-256=45c9feabba213bdc6d72e7469de71ea5aeff73faea6bfb109ab5bad37c3b43bd"
	isoFilename := filepath.Base(isoURL)

	// // download a torrent.
	// // XXX I'm not yet sure how to do this...
	// // e.g. aria2c --follow-torrent=mem --seed-time=0 https://cdimage.debian.org/debian-cd/current/amd64/bt-cd/debian-11.2.0-amd64-netinst.iso.torrent
	// isoURL := "https://cdimage.debian.org/debian-cd/current/amd64/bt-cd/debian-11.2.0-amd64-netinst.iso.torrent"
	// isoChecksum := "sha-256=3f0b3fe9e6f575d5055d3f566f3d2dcee432c3d1cf9fe662d803042eb9ad10d3"
	// isoFilename := strings.TrimSuffix(filepath.Base(isoURL), ".torrent")

	isoDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("failed to get the ISO absolution path: %v", err)
	}

	gid, err := c.AddURI(arigo.URIs(isoURL), &arigo.Options{
		Checksum: isoChecksum,
		Continue: true,
		Dir:      isoDir,
		Out:      isoFilename,
	})
	if err != nil {
		log.Fatalf("failed to queue download: %v", err)
	}
	log.Printf("%s download queued as %s", isoFilename, gid.GID)

	// wait for download to finish.
	downloadComplete := make(chan error, 1)
	go func() {
		err := gid.WaitForDownload()
		if err != nil {
			status, err := gid.TellStatus("status", "errorCode", "errorMessage")
			if err != nil {
				downloadComplete <- fmt.Errorf("failed to get download status: %v", err)
			} else {
				// TODO when a new version of arigo is published status.ErrorCode.String() to show a better error message.
				//      see https://github.com/myanimestream/arigo/blob/master/exitstatus_string.go#L49
				downloadComplete <- fmt.Errorf("failed to download: status=%s errorCode=%d errorMessage=%s", status.Status, status.ErrorCode, status.ErrorMessage)
			}
			return
		}
		downloadComplete <- nil
	}()
	for {
		select {
		case err := <-downloadComplete:
			if err != nil {
				log.Fatalf("%s download finished with error: %v", isoFilename, err)
			}
			status, err := gid.TellStatus("totalLength")
			if err != nil {
				log.Printf("%s download finished, but failed to get download status: %v", isoFilename, err)
			} else {
				log.Printf("%s download finished (%s)", isoFilename, humanize.Bytes(uint64(status.TotalLength)))
			}
			return
		case <-time.After(time.Second * 1):
			status, err := gid.TellStatus("totalLength", "completedLength", "verifiedLength")
			if err != nil {
				log.Printf("%s failed to get download status: %v", isoFilename, err)
			} else {
				if status.VerifiedLength == 0 {
					log.Printf("%s downloading %s of %s (%.0f %%)", isoFilename, humanize.Bytes(uint64(status.CompletedLength)), humanize.Bytes(uint64(status.TotalLength)), math.Ceil(100*float64(status.CompletedLength)/float64(status.TotalLength)))
				} else {
					log.Printf("%s verifying %s of %s (%.0f %%)", isoFilename, humanize.Bytes(uint64(status.VerifiedLength)), humanize.Bytes(uint64(status.TotalLength)), math.Ceil(100*float64(status.VerifiedLength)/float64(status.TotalLength)))
				}
			}
		}
	}

	// NB this is failing due to https://github.com/myanimestream/arigo/issues/2
	// status, err := c.Download(arigo.URIs(isoURL), &arigo.Options{
	// 	Checksum: isoChecksum,
	// 	Continue: true,
	// 	Dir:      isoDir,
	// 	Out:      isoFilename,
	// })
	// if err != nil {
	// 	log.Fatalf("failed to download: %v", err)
	// }
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/howeyc/fsnotify"
	"github.com/mkideal/cli"
)

type Incoming struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Title string `json:"title"`
	Color string `json:"color"`
}

type argT struct {
	Help  bool   `cli:"h,help" usage:"display help information"`
	Pubu  string `cli:"p,pubu" usage:"PubuIM Incoming URL"`
	File  string `cli:"f,file" usage:"your morintor file"`
	Color string `cli:"c,color" usage:"PubuIM Display Attachment Color [info|error]" dft:"info"`
}

var previousOffset int64 = 0
var previousFileSize int64 = 0

func main() {
	cli.Run(&argT{}, func(ctx *cli.Context) error {
		argv := ctx.Argv().(*argT)
		if argv.Help {
			ctx.String(ctx.Usage())
		} else {
			morintor(argv.File, argv.Pubu, argv.Color)
		}
		return nil
	})

}

func morintor(filePath string, hook string, color string) {
	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Fatal("notify error", err)
	}

	done := make(chan bool)

	// Process events
	go func() {

		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					getFileChangeContent(filePath, hook, color)
				}
			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Watch(filePath)
	if err != nil {
		fmt.Println(err)
	}

	// Hang so program doesn't exit
	<-done
	watcher.Close()
}

func getFileChangeContent(filePath string, hook string, color string) {
	fp, err := os.Open(filePath)

	if err != nil {
		fmt.Println("read path occured err:", err)
		return
	}

	defer fp.Close()
	fileInfo, err := fp.Stat()

	if err != nil {
		fmt.Println(err)
	}

	fileSize := fileInfo.Size()
	lastLineSize := 0

	if fileSize == previousFileSize {
		return
	}

	if previousFileSize == 0 {
		reader := bufio.NewReader(fp)

		for {
			line, _, err := reader.ReadLine()

			if err == io.EOF {
				break
			}
			lastLineSize = len(line)
		}
	} else {
		lastLineSize = int(fileSize - previousFileSize - 1)
	}

	buffer := make([]byte, lastLineSize)
	offset := fileSize - int64(lastLineSize+1)
	post, _ := fp.ReadAt(buffer, offset)

	// fmt.Printf("\n>size: %d, previousFileSize: %d,offset: %d, previousOffset :%d, linesize: %d \n", fileSize, previousFileSize, offset, previousOffset, lastLineSize)
	if previousOffset != offset {
		buffer = buffer[:post]
		jsonData := Incoming{filePath, []Attachment{{string(buffer), color}}}
		postData, _ := json.Marshal(jsonData)
		go sendPubuCloud(string(postData), hook)
		previousOffset = offset
		previousFileSize = fileSize
	}
}

func sendPubuCloud(data string, hook string) {
	request, _ := http.NewRequest("POST", hook, strings.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	re, err := http.DefaultClient.Do(request)
	fmt.Println("error", err)
	fmt.Println("re", re)
}

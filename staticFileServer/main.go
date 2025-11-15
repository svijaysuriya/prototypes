package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func FileHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("url path = ", r.URL.Path)
	osFile, err := os.Open(r.URL.Path)
	if err != nil {
		fmt.Println("error while opening the path ", r.URL.Path, err)
		w.WriteHeader(200)
		w.Write([]byte(`<html><body>Static File Server HW</body></html>`))
		return
	}
	fileStats, err := osFile.Stat()
	if err != nil {
		log.Fatalln("error while getting file stats ", osFile.Name(), err)
	}
	fmt.Println("fileStats => ", fileStats.Name(), fileStats.Mode(), fileStats.IsDir())
	content := ""
	prePendText := ""
	if fileStats.Name() != "/" {
		prePendText += fileStats.Name() + "/"
	}
	if fileStats.IsDir() {
		htmlContent := `<html><head><title>Static File Server</title></head><body><table>`
		dirs, err := osFile.ReadDir(0)
		fmt.Println("dirs length = ", len(dirs))
		if err != nil {
			log.Fatalln("error while getting directory contents ", osFile.Name(), err)
		}

		for _, dir := range dirs {
			fmt.Println("processing dir = ", dir.Name(), dir.Type())
			dirEntryInfo, err := dir.Info()
			if err != nil {
				fmt.Println("error while reading dirEntryInfo ", dir.Name(), err)
				w.WriteHeader(200)
				w.Write([]byte(`<html><body>Static File Server HW</body></html>`))
				return
			}
			li := fmt.Sprintf(`<tr><td><a href="%s">%s</a></td><td>%s</td><td>%d</td></tr>`, prePendText+dir.Name(), dir.Name(), dirEntryInfo.ModTime().String(), dirEntryInfo.Size())
			htmlContent += li
		}
		if len(dirs) == 0 {
			htmlContent += "<th>empty directory!</th>"
		}
		htmlContent += `</table></body></html>`
		content = htmlContent
	} else {
		buffer := make([]byte, fileStats.Size())
		n, err := osFile.Read(buffer)
		if err != nil {
			fmt.Println("error while reading the file ", fileStats.Name(), fileStats.Size())
			w.WriteHeader(200)
			w.Write([]byte(`<html><body>Static File Server HW</body></html>`))
			return
		}
		if n == int(fileStats.Size()) {
			fmt.Println("!! n == fileStats.Size() !!")
		}
		content = string(buffer)
	}

	w.WriteHeader(200)
	w.Write([]byte(content))
}

func main() {
	// http.Handle("/", http.FileServer(http.Dir("/")))
	http.HandleFunc("/", FileHandler)
	fmt.Println("go server listening on port 33333")
	http.ListenAndServe(":33333", nil)
}

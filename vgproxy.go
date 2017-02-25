package main

import (
    "net/http"
    "io"
    "log"
    "flag"
    "bufio"
    "strings"
    "encoding/base64"
    "bytes"
    "time"
    "sync"
    "strconv"

    // rolling file logging
    "github.com/natefinch/lumberjack"
    // optional gzip
    "github.com/NYTimes/gziphandler"
)

const Path = "/csv"

var (
    url = flag.String("url", "", "remote csv url to connect to (default: none)")
    port = flag.Int("port", 80, "port to bind HTTP to")
    addr = flag.String("addr", "", "addr to bind HTTP and HTTPS to (default: none i.e. 0.0.0.0)")
    sleep = flag.Int("sleep", 600, "seconds between refresh of cached csv")
    tlsCert = flag.String("cert", "", "cert file for HTTPS on port 443 (default: none)")
    tlsKey = flag.String("key", "", "key file for HTTPS on port 443 (default: none)")
    gzip = flag.Bool("gzip", false, "enable gzip support (default: disabled)")

    csvBody *string
    csvDate *string
    csvLength *string
    bodyLock = sync.RWMutex{}
)


func setBody(body *string) {
    now := time.Now().UTC().Format(time.RFC3339)
    length := strconv.Itoa(len(*body))

    bodyLock.Lock()
    csvBody = body
    csvDate = &now
    csvLength = &length
    bodyLock.Unlock()
}


func getBody() (body, length, date *string) {
    bodyLock.RLock()
    body = csvBody
    date = csvDate
    length = csvLength
    bodyLock.RUnlock()
    return
}


func loadBody() bool {

    log.Print("loadBody() starting download")
    resp, err := http.Get(*url)

    if err != nil {
        log.Printf("loadBody() request to url failed %v", err)
        return false
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        log.Printf("loadBody() request to url failed %d", resp.StatusCode)
        return false
    }

    // remove all comments from ovpn config files to reduce size
    csvScanner := bufio.NewScanner(resp.Body)
    body := ""

    for csvScanner.Scan() {
        line := csvScanner.Text()

        if (len(line) > 0 && line[0] != '#') {
            i := strings.LastIndex(line, ",")

            if (i > 0) {
                certRaw, err := base64.StdEncoding.DecodeString(line[i+1:])

                if err != nil {
                    log.Printf("loadBody() base decode failed %v", err)
                    return false
                }

                certScanner := bufio.NewScanner(bytes.NewReader(certRaw))
                cleanCert := ""
                for certScanner.Scan() {
                    certLine := certScanner.Text()
                    if (len(certLine) > 0 && certLine[0] != ';' && certLine[0] != '#') {
                        cleanCert += certLine + "\n"
                    }
                }

                line = line[:i+1] + base64.StdEncoding.EncodeToString([]byte(cleanCert))
            }
        }

        body += line + "\n"
    }

    if err := csvScanner.Err(); err != nil {
        log.Printf("loadBody() reading error: %v", err)
    } else if len(body) <= 0 {
        log.Print("loadBody() body is nil or empty")
    } else {
        log.Printf("loadBody() successfully finished: now %d bytes", len(body))
        setBody(&body)
        return true
    }

    return false
}


func refreshLoop() {
    timing := time.Tick(time.Duration(*sleep) * time.Second)

    for _ = range timing {
        go loadBody()
    }
}


func getCSV(w http.ResponseWriter, r *http.Request) {
    log.Printf("getCSV() answering request: %s", r.RemoteAddr)

    body, length, date := getBody()
    // custom header in response to see age of csv
    w.Header().Set("X-Source-Date", *date)
    w.Header().Set("Content-Length", *length)
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")

    // answer request
    io.WriteString(w, *body)
}


func main() {
    // parse arguments
    flag.Parse()
    if len(*url) == 0 {
        flag.PrintDefaults()
        return
    }

    // init logging
    log.SetOutput(&lumberjack.Logger{
        Filename:   "vg" + strconv.Itoa(*port) + ".log",
        MaxSize:    10, // MB
        MaxBackups: 10,
        MaxAge:     21, // days
    })

    // load csv before starting server
    for ! loadBody() {
        time.Sleep(1 * time.Second)
    }
    // start refresh
    go refreshLoop()


    if *gzip {
        // optional GZip
        log.Print("main() enable gzip support")
        handler := gziphandler.GzipHandler(http.HandlerFunc(getCSV))
        http.Handle(Path, handler)
    } else {
        http.HandleFunc(Path, getCSV)
    }

    if len(*tlsCert) > 0 || len(*tlsKey) > 0 {
        // optional HTTPS
        log.Print("main() enable HTTPS support")
        go func() { log.Fatal(http.ListenAndServeTLS(*addr + ":443", *tlsCert, *tlsKey, nil)) }()
    }
    // HTTP
    log.Fatal(http.ListenAndServe(*addr + ":" + strconv.Itoa(*port), nil))
}
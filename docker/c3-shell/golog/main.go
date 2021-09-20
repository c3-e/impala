package main

import (
    "bufio"
    "context"
    "database/sql"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "mime"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "syscall"
    "unsafe"

    log "github.com/sirupsen/logrus"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/kr/pty"
    impala "github.com/bippio/go-impala"
)

type windowSize struct {
    Rows uint16 `json:"rows"`
    Cols uint16 `json:"cols"`
    X    uint16
    Y    uint16
}

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

var impalad string
var test *bool

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
    l := log.WithField("remoteaddr", r.RemoteAddr)
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        l.WithError(err).Error("Unable to upgrade connection")
        return
    }

    defer func() {
        conn.Close()
    }()

    for {
        messageType, p, err := conn.ReadMessage()
        if err != nil {
            log.Println(err)
            return
        }

        msg := string(p)
        fmt.Println(messageType, msg)
        if strings.HasPrefix(msg, "QUERY ") {
            var str string
            if *test {
                str = testWithFile()
            } else {
                str = logs(msg[6:len(msg)])
            }
            conn.WriteMessage(websocket.BinaryMessage, []byte(str))
        }
    }
}

// not used, left for reference
func handleWebsocket2(w http.ResponseWriter, r *http.Request) {
    l := log.WithField("remoteaddr", r.RemoteAddr)
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        l.WithError(err).Error("Unable to upgrade connection")
        return
    }

    cmd := exec.Command("kubectl", "exec", "-it", "-n", "impala", "impala-shell-0", "-c", "impala-shell", "--", "bash")
    // cmd := exec.Command("/bin/bash", "-l")
    cmd.Env = append(os.Environ(), "TERM=xterm")

    tty, err := pty.Start(cmd)
    if err != nil {
        l.WithError(err).Error("Unable to start pty/cmd")
        conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
        return
    }
    defer func() {
        cmd.Process.Kill()
        cmd.Process.Wait()
        tty.Close()
        conn.Close()
    }()

    go func() {
        for {
            buf := make([]byte, 1024)
            read, err := tty.Read(buf)
            if err != nil {
                conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
                l.WithError(err).Error("Unable to read from pty/cmd")
                return
            }
            conn.WriteMessage(websocket.BinaryMessage, buf[:read])
        }
    }()

    for {
        messageType, reader, err := conn.NextReader()
        if err != nil {
            l.WithError(err).Error("Unable to grab next reader")
            return
        }

        if messageType == websocket.TextMessage {
            l.Warn("Unexpected text message")
            conn.WriteMessage(websocket.TextMessage, []byte("Unexpected text message"))
            continue
        }

        dataTypeBuf := make([]byte, 1)
        read, err := reader.Read(dataTypeBuf)
        if err != nil {
            l.WithError(err).Error("Unable to read message type from reader")
            conn.WriteMessage(websocket.TextMessage, []byte("Unable to read message type from reader"))
            return
        }

        if read != 1 {
            l.WithField("bytes", read).Error("Unexpected number of bytes read")
            return
        }

        switch dataTypeBuf[0] {
        case 0:
            copied, err := io.Copy(tty, reader)
            if err != nil {
                l.WithError(err).Errorf("Error after copying %d bytes", copied)
            }
        case 1:
            decoder := json.NewDecoder(reader)
            resizeMessage := windowSize{}
            err := decoder.Decode(&resizeMessage)
            if err != nil {
                conn.WriteMessage(websocket.TextMessage, []byte("Error decoding resize message: "+err.Error()))
                continue
            }
            log.WithField("resizeMessage", resizeMessage).Info("Resizing terminal")
            _, _, errno := syscall.Syscall(
                syscall.SYS_IOCTL,
                tty.Fd(),
                syscall.TIOCSWINSZ,
                uintptr(unsafe.Pointer(&resizeMessage)),
            )
            if errno != 0 {
                l.WithError(syscall.Errno(errno)).Error("Unable to resize terminal")
            }
        default:
            l.WithField("dataType", dataTypeBuf[0]).Error("Unknown data type")
        }
    }
}

func logs(query string) string {
    var line string
    db := connect()
    defer db.Close()

    ctx := context.Background()

    // query := "select * from ablogs.raw where service = 'c3_server' order by ts desc limit 1000;"
    log.Println(query)
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    r := struct {
        ts    string
        hostid string
        file    string
        message string
        cluster    string
        service    string
        host string
        dt string
    }{}

    for rows.Next() {
        if err := rows.Scan(&r.ts, &r.hostid, &r.file, &r.message, &r.dt, &r.cluster, &r.service, &r.host); err != nil {
            log.Fatal(err)
        }
        if line == "" {
            line = r.host + " ";
        } else {
            line = line + "\r\n" + r.host + " ";
        }
        line = line + strings.Replace(r.message, "~n", "\r\n" + r.host + " ", -1)
        // log.Printf(line)
    }
    if err := rows.Err(); err != nil {
        log.Fatal(err)
    }

    return line
}

func testWithFile() string {
    var line string

    file, err := os.Open("./c3_server.log")
    // file, err := os.Open("/home/sahn//go/src/github.com/freman/golog/c3_server.log")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    // optionally, resize scanner's capacity for lines over 64K, see next example
    var a int = 0
    for scanner.Scan() {
        // fmt.Println(scanner.Text())
        if line != "" {
            line = line + "\r\n"
        }
        line = line + strings.Replace(scanner.Text(), "~n", "\r\n", -1)
        a++
        if a > 5000 {
            break
        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

    return line
}

func abfs() string {
    var line string
    db := connect()
    defer db.Close()

    ctx := context.Background()

    query := "show files in abfs.dev_scm;"
    rows, err := db.QueryContext(ctx, query)
    if err != nil {
        log.Fatal(err)
    }

    var name, size, partition string
    for rows.Next() {
        if err := rows.Scan(&name, &size, &partition); err != nil {
            log.Fatal(err)
        }
        line = line + "\r" + name
    }
    if err := rows.Err(); err != nil {
        log.Fatal(err)
    }

    return line
}

func connect() *sql.DB {
    opts := impala.DefaultOptions

    opts.Host = impalad
    opts.Port = "21050"

    // enable LDAP authentication:
    opts.UseLDAP = true
    opts.Username = "impala"
    opts.Password = "c3impala"

    // enable TLS
    opts.UseTLS = false
    opts.CACertPath = "/path/to/cacert"

    connector := impala.NewConnector(&opts)
    db := sql.OpenDB(connector)

    return db
}

func main() {
    var listen = flag.String("listen", "127.0.0.1:3000", "Host:port to listen on")
    var assetsPath = flag.String("assets", "./assets", "Path to assets")
    impalad = *flag.String("impalad", "localhost", "Host name for impalad")
    test = flag.Bool("test", false, "Enable to test with a local file")

    flag.Parse()

    fmt.Printf("assets=%s\n", *assetsPath)
    fmt.Printf("test=%t\n", *test)

    mime.AddExtensionType(".css", "text/css; charset=utf-8")

    r := mux.NewRouter()

    r.HandleFunc("/term", handleWebsocket)
    r.PathPrefix("/").Handler(http.FileServer(http.Dir(*assetsPath)))

    log.Info("Demo Websocket/Xterm terminal")
    log.Warn("Warning, this is a completely insecure daemon that permits anyone to connect and control your computer, please don't run this anywhere")

    if !(strings.HasPrefix(*listen, "127.0.0.1") || strings.HasPrefix(*listen, "localhost")) {
        log.Warn("Danger Will Robinson - This program has no security built in and should not be exposed beyond localhost, you've been warned")
    }

    if err := http.ListenAndServe(*listen, r); err != nil {
        log.WithError(err).Fatal("Something went wrong with the webserver")
    }
}

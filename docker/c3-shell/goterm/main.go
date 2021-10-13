package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "mime"
    "net/http"
    "os"
    "os/exec"
    "syscall"
    "time"
    "unsafe"

    "app"
    log "github.com/sirupsen/logrus"
    "login"
    "middlewares"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/creack/pty"
    // Sean A. kr does not work
    // "github.com/kr/pty"
    oktaUtils "goterm/utils"
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

var auth *string
var assetsPath *string
var kubectl *bool
var authCallback *string
var useNonce *bool

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
    log.Println("Accepting a ws request...")

    l := log.WithField("remoteaddr", r.RemoteAddr)
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        l.WithError(err).Error("Unable to upgrade connection")
        return
    }

    var cmd *exec.Cmd
    if *kubectl {
        cmd = exec.Command("kubectl", "exec", "-it", "-n", "impala", "impala-shell-0", "-c", "impala-shell", "--", "bash")
    } else {
        cmd = exec.Command("/bin/bash", "-l")
    }
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

func Wrap(handler http.Handler) http.HandlerFunc {
    var etagHeaders = []string{
        "ETag",
        "If-Modified-Since",
        "If-Match",
        "If-None-Match",
        "If-Range",
        "If-Unmodified-Since",
    }

    return func(w http.ResponseWriter, r *http.Request) {
        for _, v := range etagHeaders {
            if r.Header.Get(v) != "" {
                r.Header.Del(v)
            }
        }
        w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
        w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
        w.Header().Set("Pragma", "no-cache")

        if *auth == "azure" {
            middlewares.IsAuthenticated(w, r, login.LoginHandler)
            handler.ServeHTTP(w, r)
        } else {
            if false {
                handler.ServeHTTP(w,r)
            } else if isAuthenticated(r) {
                handler.ServeHTTP(w, r)
                return
            } else {
                http.Redirect(w, r, "login", 301)
            }
        }
    }
}

func main() {
    var listen = flag.String("listen", "0.0.0.0:3000", "Host:port to listen on")
    assetsPath = flag.String("assets", "./assets", "Path to assets")
    kubectl = flag.Bool("kubectl", false, "Kubectl exec for local testing")

    auth = flag.String("auth", "azure", "auth provider, azure or okta")
    authCallback = flag.String("authcallback", "http://localhost:3000/authorization-code/callback", "Authentication Callback URL")
    useNonce = flag.Bool("nonce", false, "validate nonce")

    flag.Parse()
    fmt.Printf("assets=%s\n", *assetsPath)
    fmt.Printf("kubectl=%t\n", *kubectl)
    fmt.Printf("auth=%s\n", *auth)
    fmt.Printf("authCallback=%s\n", *authCallback)
    fmt.Printf("nonce=%t\n", *useNonce)

    mime.AddExtensionType(".css", "text/css; charset=utf-8")

    if *auth == "azure" {
        app.Init()
        StartServer()
    } else {
        oktaUtils.ParseEnvironment()

        r := mux.NewRouter()
        r.HandleFunc("/login", LoginHandler)
        r.HandleFunc("/authorization-code/callback", AuthCodeCallbackHandler)
        // r.HandleFunc("/profile", ProfileHandler)
        r.HandleFunc("/logout", LogoutHandler)
        r.HandleFunc("/term", handleWebsocket)
        r.PathPrefix("/").Handler(Wrap(http.FileServer(http.Dir(*assetsPath))))
        // r.PathPrefix("/").Handler(http.FileServer(http.Dir(*assetsPath)))

        if err := http.ListenAndServe(*listen, r); err != nil {
            log.WithError(err).Fatal("Something went wrong with the webserver")
        }
    }

    // log.Info("Demo Websocket/Xterm terminal")
    // log.Warn("Warning, this is a completely insecure daemon that permits anyone to connect and control your computer, please don't run this anywhere")

    // if !(strings.HasPrefix(*listen, "127.0.0.1") || strings.HasPrefix(*listen, "localhost")) {
    //     log.Warn("Danger Will Robinson - This program has no security built in and should not be exposed beyond localhost, you've been warned")
    // }
}

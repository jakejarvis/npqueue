package main

import (
  "bufio"
  "bytes"
  "encoding/json"
  "fmt"
  "log"
  "net"
  "net/http"
  "os"
  "strconv"
  "strings"
  "time"

  "github.com/gin-gonic/contrib/static"
  "github.com/gin-contrib/secure"
  "github.com/gin-gonic/gin"
  _ "github.com/heroku/x/hmetrics/onload"
)

type ServerDetailsStruct struct {
  Players `json:"players"`
  ServerQueue
}

type ServerQueue struct {
  CurrentPlayers int64 `json:"currentPlayers"`
  CurrentQueue   int64 `json:"currentQueue"`
}

type Players []Player

// Player details
type Player struct {
  ID          int64    `json:"id"`
  Identifiers []string `json:"identifiers"`
  Name        string   `json:"name"`
  Ping        int64    `json:"ping"`
}

type Nopixeldata []NoPixelPlayer

type NoPixelPlayer struct {
  ID        int    `json:"id"`
  Name      string `json:"name"`
  NoPixelID string `json:"noPixelID"`
  SteamID   string `json:"steamID"`
  Twitch    string `json:"twitch"`
}

var (
  jsonGet = &http.Client{Timeout: 10 * time.Second}
  // Using an environment variable to protect IP
  ServerAddress = os.Getenv("SERVER_IP")
  // ServerDetails struct to hold PlayerList & ServerDetails struct
  ServerDetails = &ServerDetailsStruct{}
  // NoPixelData struct
  NoPixelData Nopixeldata
)

// getPlayerList sends HTTP get request to get list of players from /players.json
func getPlayerList() (err error) {
  server := strings.Builder{}
  fmt.Fprintf(&server, "http://%s/players.json", ServerAddress)

  req, err := jsonGet.Get(server.String())
  if err != nil {
    return err
  }
  defer req.Body.Close()

  err = json.NewDecoder(req.Body).Decode(&ServerDetails.Players)
  if err != nil {
    return err
  }

  return
}

// getServerQueue opens UDP socket to get queue count
func getServerQueue() (err error) {
  serverData := make([]byte, 256)
  serverConnection, err := net.Dial("udp", ServerAddress)
  if err != nil {
    return err
  } else {
    defer serverConnection.Close()
  }

  // UDP voodoo to get server info -- https://github.com/LiquidObsidian/fivereborn-query/blob/master/index.js#L54
  fmt.Fprintf(serverConnection, "\xFF\xFF\xFF\xFFgetinfo f")
  _, err = bufio.NewReader(serverConnection).Read(serverData)

  if err == nil {
    serverData := bytes.Split(serverData, []byte("\n"))
    serverDetails := bytes.Split(serverData[1], []byte("\\"))
    serverQueue := bytes.FieldsFunc(serverDetails[12], func(c rune) bool { return c == '[' || c == ']' })

    currentPlayerValues, _ := strconv.ParseInt(string(serverDetails[4]), 0, 64)
    currentserverQueueValues, _ := strconv.ParseInt(string(serverQueue[0]), 0, 64)
    ServerDetails.ServerQueue.CurrentPlayers = currentPlayerValues
  
    if currentserverQueueValues >= 1 {
      ServerDetails.ServerQueue.CurrentQueue = currentserverQueueValues
    }
  } else {
    return err
  }

  return
}

func steam64toSteam(input int64) (steamid string) {
  legacySteamid := ((input - 76561197960265728) / 2)
  steamid = fmt.Sprintf("STEAM_0:%d:%d", (input % 2), legacySteamid)

  return
}

func parsePlayers() (err error) {
  var steamIDs []string
  for i, v := range ServerDetails.Players {
    steamIDs = nil
    for ii, vv := range v.Identifiers {
      if ii == 0 {
        hexID := strings.Replace(vv, "steam:", "0x", -1)
        steamID, _ := strconv.ParseInt(hexID, 0, 64)
        s := strconv.FormatInt(steamID, 10)
        p := getPlayerNoPixelInformation(s)

        steamIDs = append(steamIDs,
          p.Name,
          steam64toSteam(steamID),
          fmt.Sprintf("%d", steamID),
          p.Twitch,
          p.NoPixelID)
      }
    }
    ServerDetails.Players[i].Identifiers = steamIDs
  }

  return
}

func loadPlayersJSON() (err error) {
  jsonFile, err := jsonGet.Get("https://github.com/jakejarvis/npqueue/raw/master/directory.json")
  if err != nil {
    return
  }

  err = json.NewDecoder(jsonFile.Body).Decode(&NoPixelData)
  if err != nil {
    return err
  }

  return
}

func getPlayerNoPixelInformation(id string) (p NoPixelPlayer) {
  for i := range NoPixelData {
    if NoPixelData[i].SteamID == id {
      return NoPixelData[i]
    }
  }

  return
}

// List handler for /api/list route
func ListHandler(c *gin.Context) {
  // Load players JSON
  err := loadPlayersJSON()
  if err != nil {
    log.Fatalf("Failed to load players JSON:  %v", err)
    return
  }

  // Get player list
  err = getPlayerList()
  if err != nil {
    log.Fatalf("Failed to get player list:  %v", err)
    return
  }

  // Get server queue count
  err = getServerQueue()
  if err != nil {
    log.Fatalf("Failed to get server queue count:  %v", err)
    return
  }

  // Parse players JSON
  err = parsePlayers()
  if err != nil {
    log.Fatalf("Failed to parse players JSON:  %v", err)
    return
  }

  c.Header("Content-Type", "application/json")
  c.Header("Access-Control-Allow-Origin", "*")

  c.JSON(http.StatusOK, ServerDetails)
}

func main() {
  port := ":" + os.Getenv("PORT")
  router := gin.Default()

  router.Use(secure.New(secure.Config{
    SSLRedirect:           true,
    SSLHost:               "np.pogge.rs",
    STSSeconds:            315360000,
    STSIncludeSubdomains:  false,
    FrameDeny:             true,
    ContentTypeNosniff:    true,
    BrowserXssFilter:      true,
    SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
  }))

  // Serve frontend static files
  router.Use(static.Serve("/", static.LocalFile("./public", true)))

  // Setup route group for the API
  api := router.Group("/api")
  {
    api.GET("/", func(c *gin.Context) {
      c.JSON(http.StatusOK, gin.H {
        "message": "Nothing to see here.",
      })
    })
  }

  // List handler for /api/list
  api.GET("/list", ListHandler)

  // Run the Gin router
  if err := router.Run(port); err != nil {
    log.Fatalf("Gin fatal error: %v", err)
  } else {
    log.Printf("Listening on %s...\n", port)
  }
}
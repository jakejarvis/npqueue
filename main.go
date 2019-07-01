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
  defer serverConnection.Close()
  if err != nil {
    return err
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

// List handler for now.sh /api/list route
func List(w http.ResponseWriter, r *http.Request) {
  err := loadPlayersJSON()
  if err != nil {
    fmt.Fprintf(w, "failed to load JSON file %v", err)
    return
  }

  getPlayerList()
  getServerQueue()
  parsePlayers()

  w.Header().Set("Content-Type", "application/json")
  w.Header().Set("Access-Control-Allow-Origin", "*")

  json.NewEncoder(w).Encode(ServerDetails)
}

func main() {
  addr := ":" + os.Getenv("PORT")
  
  http.HandleFunc("/api/list", List)
  http.Handle("/", http.FileServer(http.Dir("./public")))

  log.Printf("Listening on %s...\n", addr)
  if err := http.ListenAndServe(addr, nil); err != nil {
    panic(err)
  }
}
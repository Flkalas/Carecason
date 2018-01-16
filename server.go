package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "io/ioutil"
	"log"
	_ "math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type RequestJSON struct {
	Req   string `json:"req"`
	Param string `json:"param"`
}

type MapRequest struct {
	XStart int64 `json:"xStart"`
	XEnd   int64 `json:"xEnd"`
	YStart int64 `json:"yStart"`
	YEnd   int64 `json:"yEnd"`
}

type MoveRequest struct {
	Direction int `json:"direction"`
}

type TileData struct {
	Res  string `json:"res"`
	Data [6]int `json:"data"`
	PosX int64  `json:"posX"`
	PosY int64  `json:"posY"`
}

type UserPositionData struct {
	Res string `json:"res"`
	X   int64  `json:"posX"`
	Y   int64  `json:"posY"`
}

type AckJSON struct {
	Res string `json:"res"`
}

type MapChunk struct {
	Data [16][16][6]int
	PosX int64
	PosY int64
}

var userPos = UserPositionData{"USER_POS", 0, 0}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	rand.Seed(time.Now().UnixNano())

	initMapTest()

	initMapFolder()
	makeFirstMap()
	fmt.Println(existChunk(0, 0))

	for i := 0; i < 10; i++ {
		list := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		fmt.Println(sample(list, 3))
	}

	http.HandleFunc("/map", mapHandler)
	http.ListenAndServe(":8080", nil)
	fmt.Println("Server Started.")
}

func initMapTest() {
	err := os.RemoveAll("./MapData")
	if err != nil {
		return
	}
}

func mapHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for true {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var req RequestJSON
		err = json.Unmarshal(p, &req)
		fmt.Println(req)

		if req.Req == "MAP" {
			responseMap(req.Param, conn)
		} else if req.Req == "MOVE" {
			responseMove(req.Param, conn)
		} else if req.Req == "USER_INIT" {
			var initUser = UserPositionData{"USER_INIT", 0, 0}
			initUser.X = userPos.X
			initUser.Y = userPos.Y
			fmt.Println(initUser)
			js, err := json.Marshal(initUser)
			if err != nil {
				return
			}
			fmt.Println(string(js))
			if err = conn.WriteMessage(websocket.TextMessage, js); err != nil {
				return
			}

		}

		fmt.Println(messageType, string(p))
		if err = conn.WriteMessage(messageType, p); err != nil {
			return
		}
	}

	//	xStart, _ := strconv.ParseInt(r.FormValue("xfrom"), 10, 64)
	//	yStart, _ := strconv.ParseInt(r.FormValue("yfrom"), 10, 64)
	//	xEnd, _ := strconv.ParseInt(r.FormValue("xto"), 10, 64)
	//	yEnd, _ := strconv.ParseInt(r.FormValue("yto"), 10, 64)

	//	sorted := sortCoords(xStart, xEnd, yStart, yEnd)
	//	var mapData []TileData

}

func responseMove(param string, conn *websocket.Conn) {
	var moveReq MoveRequest
	json.Unmarshal([]byte(param), &moveReq)
	fmt.Println(moveReq)

	if moveReq.Direction == 37 {
		userPos.X = userPos.X - 1
		//left
	} else if moveReq.Direction == 38 {
		userPos.Y = userPos.Y + 1
		//up
	} else if moveReq.Direction == 39 {
		userPos.X = userPos.X + 1
		//right
	} else if moveReq.Direction == 40 {
		//down
		userPos.Y = userPos.Y - 1
	}

	fmt.Println(userPos)
	js, err := json.Marshal(userPos)
	if err != nil {
		return
	}
	fmt.Println(string(js))
	if err = conn.WriteMessage(websocket.TextMessage, js); err != nil {
		return
	}

}

func absInt(a int64) int64 {
	var i int64 = a

	if a < 0 {
		i = -a
	}

	return i
}

func responseMap(param string, conn *websocket.Conn) {
	var mapReq MapRequest
	json.Unmarshal([]byte(param), &mapReq)
	fmt.Println(mapReq)

	sorted := sortCoords(mapReq)
	fmt.Println(sorted)

	tData := make([]TileData, 0)
	for i := sorted[0]; i < sorted[1]+1; i++ {
		for j := sorted[2]; j < sorted[3]+1; j++ {
			var t TileData

			t.Res = "MAP"
			t.Data = [6]int{0, 0, 0, 0, 0, 0}
			t.Data[absInt(i)%6] = 1
			t.Data[absInt(i+j)%6] = 1
			t.PosX = i
			t.PosY = j

			tData = append(tData, t)
		}
	}

	fmt.Println(tData)

	for i := range tData {
		js, err := json.Marshal(tData[i])
		if err != nil {
			return
		}
		fmt.Println(string(js))
		if err = conn.WriteMessage(websocket.TextMessage, js); err != nil {
			return
		}
	}
	sendMapAckJSON(conn)
}

func initMapFolder() {
	_, err := os.Stat("./MapData")
	if os.IsNotExist(err) {
		err = os.Mkdir("./MapData", os.ModeDir)
		if err != nil {
			return
		}
	}

}

func initMapChunk(mC *MapChunk) {
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			for k := 0; k < 6; k++ {
				mC.Data[i][j][k] = -1
			}
		}
	}
}

func makeFirstMap() {
	var nowMap MapChunk
	initMapChunk(&nowMap)

	nowMap.PosX = 0
	nowMap.PosY = 0

	nowMap.Data[0][0] = [6]int{0, 1, 0, 0, 1, 0}

	fmt.Println(nowMap.Data[0][0])
	fmt.Println(nowMap.Data[0][1])

	makeNextTiles(nowMap, 0, 0)
}

func isCounterSameChunk(mC MapChunk, tilePosX int64, tilePosY int64, index int) bool {
	nextX, nextY := getNextTileCoord(tilePosX, tilePosY, index)
	nextMapX := getMapBorderChange(nextX, mC.PosX)
	nextMapY := getMapBorderChange(nextY, mC.PosY)

	//fmt.Println(index, mC.Data[tilePosX][tilePosY][index], nextX, nextY, nextMapX, nextMapY, mC.PosX, mC.PosY)

	return (nextMapX == mC.PosX) && (nextMapY == mC.PosY)
}

func makeNextTiles(mC MapChunk, tilePosX int64, tilePosY int64) {

	for i := 0; i < 6; i++ {
		//fmt.Println(mC.Data[tilePosX][tilePosY][i], isCounterSameChunk(mC, tilePosX, tilePosY, i))
		if (mC.Data[tilePosX][tilePosY][i] == 1) && isCounterSameChunk(mC, tilePosX, tilePosY, i) {
			x, y := getNextTileCoord(tilePosX, tilePosY, i)
			mC.Data[x][y] = makeNextTile(mC, x, y, i)
			fmt.Println(mC.Data[x][y])
		}

	}
}

func makeNextTile(mC MapChunk, tilePosX int64, tilePosY int64, index int) [6]int {
	var tData [6]int

	occupiedIndex := getCounterIndex(index)
	tData[occupiedIndex] = 1

	existanceNextTiles := getExistNextTiles(tilePosX, tilePosY, mC)

	var leftEnableBranch = make([]int, 0)
	for i := 0; i < 6; i++ {
		if (i != occupiedIndex) || (!existanceNextTiles[i]) {
			leftEnableBranch = append(leftEnableBranch, i)
		}
	}

	next := sample(leftEnableBranch, getNumberOfBranch())
	for _, nextIndex := range next {
		tData[nextIndex] = 1
	}

	return tData
}

func getExistNextTiles(PosX int64, PosY int64, mC MapChunk) [6]bool {
	var existance = make([]bool, 0)

	var arrExist [6]bool

	existance = append(existance, getCounterExist(PosX, PosY, 0, mC))

	copy(arrExist[:], existance[:6])

	return arrExist
}

func existChunk(PosX int64, PosY int64) bool {
	buff := bytes.NewBufferString("./MapData/")
	buff.WriteString(strconv.FormatInt(PosX, 10))
	buff.WriteString("_")
	buff.WriteString(strconv.FormatInt(PosY, 10))
	buff.WriteString(".dat")

	fmt.Println(buff.String())

	_, err := os.Stat(buff.String())
	return os.IsExist(err)
}

func getCounterExist(PosX int64, PosY int64, index int, mC MapChunk) bool {
	x, y := getNextTileCoord(PosX, PosY, index)

	if isCounterSameChunk(mC, PosX, PosY, index) {

		return mC.Data[x][y][0] != -1
	} else {
		mapX := getMapBorderChange(x, mC.PosX)
		mapY := getMapBorderChange(y, mC.PosY)
		return existChunk(mapX, mapY)
	}

	return true
}

func getMapBorderChange(x int64, MapX int64) int64 {
	if x < 0 {
		return MapX - 1
	} else if x > 15 {
		return MapX + 1
	}
	return MapX
}

func getNextTileCoord(PosX int64, PosY int64, index int) (int64, int64) {
	var adjust int64 = 0

	if absInt(PosX)%2 == 1 {
		adjust = 1
	}

	if index == 0 {
		return PosX + 1, PosY - 1 + adjust
	} else if index == 1 {
		return PosX, PosY - 1
	} else if index == 2 {
		return PosX - 1, PosY - 1 + adjust
	} else if index == 3 {
		return PosX - 1, PosY + adjust
	} else if index == 4 {
		return PosX, PosY + 1
	} else if index == 5 {
		return PosX + 1, PosY + adjust
	}

	return PosX, PosY
}

func sample(list []int, num int) []int {
	left := make([]int, 0)

	for len(left) != num {
		selected := rand.Intn(len(list))

		left = append(left, list[selected])
		list = append(list[:selected], list[selected+1:]...)
	}

	sort.Ints(left)

	return left
}

func getCounterIndex(index int) int {
	if index == 0 {
		return 3
	} else if index == 1 {
		return 4
	} else if index == 2 {
		return 5
	} else if index == 3 {
		return 0
	} else if index == 4 {
		return 1
	} else if index == 5 {
		return 2
	}

	return -1
}

func getNumberOfBranch() int {
	selected := rand.Float64()

	//0.37 0.43 0.1 0.05 0.03 0.02

	if selected < 0.37 {
		return 0
	} else if selected < 0.8 {
		return 1
	} else if selected < 0.9 {
		return 2
	} else if selected < 0.95 {
		return 3
	} else if selected < 0.98 {
		return 4
	} else {
		return 5
	}
}

func loadMaps() {
	var mapX int64 = userPos.X % 16
	var mapY int64 = userPos.Y % 16

	for i := mapX - 1; i < mapX+2; i++ {
		for j := mapY - 1; j < mapY+2; j++ {
			if isExistMap(i, j) {

			} else {

			}
		}
	}
}

func isExistMap(x int64, y int64) bool {
	var isExist bool = false

	return isExist
}

func makeMap(x int64, y int64) {
	var isExistNextMap [6]bool = [6]bool{false, false, false, false, false, false}

	isExistNextMap[0] = isExistMap(x+1, y-1)
	isExistNextMap[1] = isExistMap(x, y-1)
	isExistNextMap[2] = isExistMap(x-1, y-1)
	isExistNextMap[3] = isExistMap(x-1, y)
	isExistNextMap[4] = isExistMap(x, y+1)
	isExistNextMap[5] = isExistMap(x+1, y)

}

func sendMapAckJSON(conn *websocket.Conn) {
	var ack = AckJSON{"MAP_SEND_END"}

	js, err := json.Marshal(ack)
	if err != nil {
		return
	}
	fmt.Println(string(js))
	if err = conn.WriteMessage(websocket.TextMessage, js); err != nil {
		return
	}
}

func sortCoords(mapReq MapRequest) []int64 {
	sorted := make([]int64, 0)

	sortOneAxis(mapReq.XEnd, mapReq.XStart, &sorted)
	sortOneAxis(mapReq.YEnd, mapReq.YStart, &sorted)

	return sorted
}

func sortOneAxis(x1 int64, x2 int64, a *[]int64) {
	if x1 > x2 {
		*a = append(*a, x2)
		*a = append(*a, x1)
	} else {
		*a = append(*a, x1)
		*a = append(*a, x2)
	}
}

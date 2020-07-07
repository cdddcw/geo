package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var clients []*Client

//省province 市city 县district 镇town 村 village 街street
//id,ext_path,geo,polygon
type Client struct { // Our example struct, you can use "-" to ignore a field
	Id        int    `json:"adcode"`
	Level     string `json:"level"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Geo       *LonLat   `json:"center"`
	ExtPath   []string `json:"-"`
	Polygon   []*LonLat `json:"-"`
	For       []*LonLat `json:"-"`
	SubClient []*Client `json:"-"`
}
type LonLat struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

//
//func pnpoly(nvert int, vertx []float64, verty []float64, testx float64, testy float64) int {
//	i, j, c := 0, 0, 0
//	for i, j = 0, nvert-1; i < nvert; i++ {
//		//fmt.Println(vertx[j],vertx[i])
//		//fmt.Println(i,j,verty[i] > testy,verty[j] > testy)
//		//return 0
//		if ((verty[i] > testy) != (verty[j] > testy)) &&
//			(testx < (vertx[j]-vertx[i])*(testy-verty[i])/(verty[j]-verty[i])+vertx[i]) {
//			c++
//			fmt.Println(c)
//		}
//		j = i
//
//	}
//	return c % 2
//
//}

func pnpoly(vert []*LonLat, testt *LonLat) int {
	i, j, c := 0, 0, 0
	nvert := len(vert)
	for i, j = 0, nvert-1; i < nvert; i++ {
		if ((vert[i].Lat > testt.Lat) != (vert[j].Lat > testt.Lat)) &&
			(testt.Lon < (vert[j].Lon-vert[i].Lon)*(testt.Lat-vert[i].Lat)/(vert[j].Lat-vert[i].Lat)+vert[i].Lon) {
			c++
		}
		j = i
	}
	return c % 2

}

func Str2LonLat(str string) *LonLat {
	lonlat := strings.Split(str, " ")
	if len(lonlat) == 2 {
		lon, err := strconv.ParseFloat(lonlat[0], 64)
		if err != nil {
			return nil
		}
		lat, err := strconv.ParseFloat(lonlat[1], 64)
		if err != nil {
			return nil
		}
		return &LonLat{Lon: lon, Lat: lat}
	}
	return nil

}
func Str2LonLats(str string) []*LonLat {
	lonlats := make([]*LonLat, 0)
	for _, v := range strings.Split(str, ",") {
		if lonlat := Str2LonLat(v); lonlat != nil {
			lonlats = append(lonlats, lonlat)
		}
	}
	return lonlats
}

func checkPolygon(clients []*Client, test LonLat) *Client {
	for _, client := range clients {
		if pnpoly(client.For, &test) == 1 {
			if client.SubClient == nil {
				if pnpoly(client.Polygon, &test) == 1 {
					return client
				}
			} else {
				if c := checkPolygon(client.SubClient, test); c != nil {
					return c
				} else {
					if  pnpoly(client.Polygon, &test) == 1 {
						return client
					}
				}
			}
		}
	}
	return nil
}

func init() {
	t := time.Now()
	clientsFile, err := os.OpenFile("ok_geo.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	defer clientsFile.Close()

	clients = []*Client{}
	cr := csv.NewReader(bufio.NewReader(clientsFile))
	for {
		d, err := cr.Read()
		if err != nil {
			break
		}
		id, err := strconv.Atoi(d[0])
		if err != nil {
			continue
		}
		client := &Client{Id: id, ExtPath: strings.Split(d[1], " "), Geo: Str2LonLat(d[2]), Polygon: Str2LonLats(d[3])}
		var maxX, minX, maxY, minY float64
		for i, v := range client.Polygon {
			if i == 0 {
				maxX, minX = v.Lon, v.Lon
				maxY, minY = v.Lat, v.Lat
			} else {
				if maxX < v.Lon {
					maxX = v.Lon
				}
				if minX > v.Lon {
					minX = v.Lon
				}
				if maxY < v.Lat {
					maxY = v.Lat
				}
				if minY > v.Lat {
					minY = v.Lat
				}
			}
		}
		client.For = []*LonLat{&LonLat{minX, minY}, &LonLat{minX, maxY}, &LonLat{maxX, maxY}, &LonLat{maxX, minY}}

		if client.Id < 100 {
			client.Level = "province"
			client.Province = client.ExtPath[0]
			clients = append(clients, client)
		} else if client.Id < 10000 {
			client.Level = "city"
			client.Province = client.ExtPath[0]
			client.City = client.ExtPath[1]
			lv1Id := client.Id / 100
			for _, v := range clients {
				if v.Id == lv1Id {
					if v.SubClient == nil {
						v.SubClient = make([]*Client, 0)
					}
					v.SubClient = append(v.SubClient, client)
				}
			}
		} else if client.Id < 1000000 {
			client.Level = "district"
			client.Province = client.ExtPath[0]
			client.City = client.ExtPath[1]
			if len(client.ExtPath) > 2 {
				client.District = client.ExtPath[2]
			} else {
				client.District = client.ExtPath[1]
			}

			lv1Id := client.Id / 10000
			lv2Id := client.Id / 100
			for _, v := range clients {
				if v.Id == lv1Id {
					for _, v2 := range v.SubClient {
						if v2.Id == lv2Id {
							if v2.SubClient == nil {
								v2.SubClient = make([]*Client, 0)
							}
							v2.SubClient = append(v2.SubClient, client)
						}
					}
				}
			}
		}
	}
	fmt.Println(time.Since(t))
}

func geoHandle(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	lon, _ := strconv.ParseFloat(query.Get("lon"), 64)
	lat, _ := strconv.ParseFloat(query.Get("lat"), 64)
	test := LonLat{lon, lat}
	t := time.Now()
	client := checkPolygon(clients, test)
	fmt.Println(time.Since(t))
	rst := RstData{}
	if client != nil {
		rst.Code=0
		rst.Msg = "success"
		rst.Data = client
	} else {
		rst.Code = 1
		rst.Msg = "fail"
		//rst.Data = struct {}{}
	}
	ret, _ := json.Marshal(&rst)
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Write(ret)
}

type RstData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func main() {
	http.HandleFunc("/geo", geoHandle)       // 设置访问的路由
	err := http.ListenAndServe(":9090", nil) // 设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

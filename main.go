package main

import (
	// "errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/parnurzeal/gorequest"
)

type Futures struct {
	Contract     string `json:"contract"`
	Price        string `json:"price"`
	Volume       string `json:"ttlvol"`
	ContractName string `json:"contractName"`
	Updown       string `json:"updown"`
}

type StockPrice struct {
	Open  int
	High  int
	Low   int
	Close int
}

const (
	URL = "http://www.taifex.com.tw/cht/quotesApi/getQuotes"
)

var pre_vol int
var pre_ud int
var pre_future []Futures
var pre_data_init bool = false
var m2 int = 0
var keep_m1 = []int{0, 0, 0, 0, 0, 0}

func main() {
	var bOpened bool
	pbDetail := flag.Bool("detail", false, "detail")
	psTime := flag.String("time", "auto", "day or night or auto")
	pbWait := flag.Bool("wait", false, "wait forever to open")
	pbHelp := flag.Bool("help", false, "help")
	flag.Parse()

	if *pbHelp {
		fmt.Println("--detail for bool default false\n--time for day/night/auto default auto")
		return
	}

RESTART:
	pre_vol = 0
	pre_ud = 0
	pre_future = nil
	pre_data_init = false
	m2 = 0
	for i := range keep_m1 {
		keep_m1[i] = 0
	}

	bOpened = isOpen()
	fmt.Println("isOpen:", bOpened)
	time.Sleep(1 * time.Second)

	if !bOpened {
		fmt.Printf("Wait to open.......")
		time.Sleep(1 * time.Second)
		if !*pbWait {
			return
		} else {
			for {
				diffH, diffM, diffS := getDiffToNextOpenTime()
				fmt.Fprintf(os.Stderr, "\rWait %2dh% 2dm% 2ds to open.......", diffH, diffM, diffS)
				time.Sleep(1 * time.Second)
				if isOpen() {
					fmt.Printf("\r\n")
					time.Sleep(1 * time.Second)
					break
				}
			}
		}
	}

	// fetch_retry := 0
	for true {
		url := getURL(*psTime)
		futrues, err := fetch(url)
		if !isOpen() {
			if !*pbWait {
				return
			} else {
				goto RESTART
			}
		}

		if err != nil {
			fmt.Println(err)
			if !*pbWait {
				return
			} else {
				goto RESTART
			}
		}
		if len(futrues) == 0 {
			time.Sleep(1 * time.Second)
			// fetch_retry++
			continue
		}
		if len(pre_future) > 0 {
			if futrues[0].Volume == pre_future[0].Volume {
				continue
			}
		}
		if *pbDetail {
			printDetail(futrues)
		} else {
			printBrief(futrues)
		}
		time.Sleep(time.Duration(5000+rand.Intn(1000)) * time.Millisecond)
		pre_future = futrues
	}
}

func printDetail(futrues []Futures) {
	for _, future := range futrues {
		s := fmt.Sprintf("c:%s, p:% 5s, vol:% 7s, name:%s, range:% 5s", future.Contract, future.Price, future.Volume, future.ContractName, future.Updown)
		fmt.Println(s)
	}
}

func StrToInt(s string) int {
	num, err := strconv.Atoi(strings.Replace(s, ",", "", -1))
	if err == nil {
		return num
	}
	return -1
}

func printBrief(futrues []Futures) {
	future := futrues[0]
	vol := StrToInt(future.Volume)
	price := StrToInt(future.Price)
	updown := StrToInt(future.Updown)
	bIsDay := isDay()
	timeStr := time.Now().Format("2006-01-02 15:04:05")

	var th int
	if bIsDay {
		th = 8000
	} else {
		th = 1000
	}

	if pre_data_init == false {
		pre_vol = vol
		pre_ud = updown
	}
	m1 := (vol - pre_vol) * (updown - pre_ud)
	keep_m1 = keep_m1[1:]
	keep_m1 = append(keep_m1, m1)
	if (Sum(keep_m1) > th && m1 > th) || (Sum(keep_m1) < -th && m1 < -th) {
		m2 = 0
		err := beeep.Notify("GOGOGO", timeStr, "")
		if err != nil {
			panic(err)
		}
	} else {
		m2 += m1
	}
	s := fmt.Sprintf("[%s] p:% 5d, v:% 7d, r:% 5d, v_dif:% 5d, r_dif:% 4d, m1:% 9d, m2:% 9d",
		timeStr,
		price,
		vol,
		updown,
		vol-pre_vol,
		updown-pre_ud,
		m1,
		m2)
	pre_vol = vol
	pre_ud = updown
	pre_data_init = true
	fmt.Println(s)
}

func fetch(url string) (futrues []Futures, err error) {
	_, _, errs := gorequest.New().Get(url).EndStruct(&futrues)

	// if resp.StatusCode != 200 {
	// err = errors.New("fetch failed")
	// }
	if errs != nil {
		return
	}
	return
}

func getURL(time string) (url string) {
	switch time {
	case "day":
		return fmt.Sprintf("%s?objId=2", URL)
	case "night":
		return fmt.Sprintf("%s?objId=12", URL)
	default:
		if isDay() {
			return fmt.Sprintf("%s?objId=2", URL)
		}
		return fmt.Sprintf("%s?objId=12", URL)
	}
}

func isOpen() bool {
	t := time.Now()
	h := float64(t.Hour())
	m := float64(t.Minute())
	s := float64(t.Second())
	t_in_min := h*60 + m + s/60

	if t.Weekday() == 0 || (t.Weekday() == 6 && t_in_min > 300) {
		return false
	}

	// 05:00 = 300
	// 08:45 = 525
	// 13:45 = 825
	// 15:00 = 900

	if t_in_min < 525 && t_in_min > 300 {
		return false
	}
	if t_in_min > 825 && t_in_min < 900 {
		return false
	}
	return true
}

func isDay() bool {
	t := time.Now()
	if t.Weekday() == 0 || t.Weekday() == 6 {
		return false
	}
	h := t.Hour()
	m := t.Minute()
	s := t.Second()

	t_in_min := h*60 + m + s/60.0

	// 05:00 = 300
	// 08:45 = 525
	// 13:45 = 825
	// 15:00 = 900

	if t_in_min >= 525 && t_in_min <= 825 {
		return true
	}
	return false
}

func getDiffToNextOpenTime() (H int, M int, S int) {
	t := time.Now()

	if t.Weekday() == 0 || t.Weekday() == 6 {
		return 0, 0, 0
	}
	h := float64(t.Hour())
	m := float64(t.Minute())
	s := float64(t.Second())

	t_in_min := h*60 + m + s/60

	// 08:45 = 525
	// 15:00 = 900
	aOpenTime := [2]float64{525, 900}
	openTime := aOpenTime[0]
	for _, t := range aOpenTime {
		if t_in_min < t {
			openTime = t
			break
		}
	}
	H = int(openTime-t_in_min) / 60
	M = int(openTime-t_in_min) % 60
	S = int(59 - s)
	return
}

func Sum(a []int) int {
	s := 0
	for _, v := range a {
		s += v
	}
	return s
}
